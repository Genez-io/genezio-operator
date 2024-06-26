/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	initv1alpha1 "github.com/Genez-io/genezio-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// GenezioManagerReconciler reconciles a GenezioManager object
type GenezioManagerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// Definitions to manage status conditions
const (
	// typeAvailableGenezioManager represents the status of the Deployment reconciliation
	typeAvailableGenezioManager = "Available"
	// typeDegradedGenezioManager represents the status used when the custom resource is deleted and the finalizer operations are must to occur.
	typeDegradedGenezioManager = "Degraded"
)

const geneziomanagerFinalizer = "finalizer.init.genezio.com"

//+kubebuilder:rbac:groups=init.genezio.com,resources=geneziomanagers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=init.genezio.com,resources=geneziomanagers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=init.genezio.com,resources=geneziomanagers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GenezioManager object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *GenezioManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the GenezioManager instance
	geneziomanager := &initv1alpha1.GenezioManager{}
	err := r.Get(ctx, req.NamespacedName, geneziomanager)
	if err != nil {
		log.Error(err, "unable to fetch GenezioManager")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Let's just set the status as Unknown when no status are available
	if geneziomanager.Status.Conditions == nil || len(geneziomanager.Status.Conditions) == 0 {
		meta.SetStatusCondition(&geneziomanager.Status.Conditions, metav1.Condition{
			Type:    typeAvailableGenezioManager,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		})
		if err = r.Status().Update(ctx, geneziomanager); err != nil {
			log.Error(err, "Failed to update geneziomanager status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the geneziomanager Custom Resource after update the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raise the issue "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, geneziomanager); err != nil {
			log.Error(err, "Failed to re-fetch geneziomanager")
			return ctrl.Result{}, err
		}
	}

	// Let's add a finalizer. Then, we can define some operations which should
	// occurs before the custom resource to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if !controllerutil.ContainsFinalizer(geneziomanager, geneziomanagerFinalizer) {
		log.Info("Adding Finalizer for geneziomanager")
		if ok := controllerutil.AddFinalizer(geneziomanager, geneziomanagerFinalizer); !ok {
			log.Error(err, "Failed to add finalizer into the custom resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.Update(ctx, geneziomanager); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if the geneziomanager instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isgeneziomanagerMarkedToBeDeleted := geneziomanager.GetDeletionTimestamp() != nil

	if isgeneziomanagerMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(geneziomanager, geneziomanagerFinalizer) {
			log.Info("Performing Finalizer Operations for geneziomanager before delete CR")

			// Let's add here an status "Downgrade" to define that this resource begin its process to be terminated.
			meta.SetStatusCondition(&geneziomanager.Status.Conditions, metav1.Condition{Type: typeDegradedGenezioManager,
				Status: metav1.ConditionUnknown, Reason: "Finalizing",
				Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", geneziomanager.Name)})

			if err := r.Status().Update(ctx, geneziomanager); err != nil {
				log.Error(err, "Failed to update geneziomanager status")
				return ctrl.Result{}, err
			}

			// Perform all operations required before remove the finalizer and allow
			// the Kubernetes API to remove the custom resource.
			// r.doFinalizerOperationsForgeneziomanager(geneziomanager)

			// TODO(user): If you add operations to the doFinalizerOperationsForgeneziomanager method
			// then you need to ensure that all worked fine before deleting and updating the Downgrade status
			// otherwise, you should requeue here.

			// Re-fetch the geneziomanager Custom Resource before update the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raise the issue "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, geneziomanager); err != nil {
				log.Error(err, "Failed to re-fetch geneziomanager")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&geneziomanager.Status.Conditions, metav1.Condition{Type: typeDegradedGenezioManager,
				Status: metav1.ConditionTrue, Reason: "Finalizing",
				Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", geneziomanager.Name)})

			if err := r.Status().Update(ctx, geneziomanager); err != nil {
				log.Error(err, "Failed to update geneziomanager status")
				return ctrl.Result{}, err
			}

			log.Info("Removing Finalizer for geneziomanager after successfully perform the operations")
			if ok := controllerutil.RemoveFinalizer(geneziomanager, geneziomanagerFinalizer); !ok {
				log.Error(err, "Failed to remove finalizer for geneziomanager")
				return ctrl.Result{Requeue: true}, nil
			}

			if err := r.Update(ctx, geneziomanager); err != nil {
				log.Error(err, "Failed to remove finalizer for geneziomanager")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: geneziomanager.Name, Namespace: geneziomanager.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForGenezioManager(geneziomanager)
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for GenezioManager")

			// The following implementation will update the status
			meta.SetStatusCondition(&geneziomanager.Status.Conditions, metav1.Condition{Type: typeAvailableGenezioManager,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", geneziomanager.Name, err)})

			if err := r.Status().Update(ctx, geneziomanager); err != nil {
				log.Error(err, "Failed to update GenezioManager status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		// Deployment created successfully
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	// The following implementation will update the status
	meta.SetStatusCondition(&geneziomanager.Status.Conditions, metav1.Condition{Type: typeAvailableGenezioManager,
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) created successfully", geneziomanager.Name)})

	if err := r.Status().Update(ctx, geneziomanager); err != nil {
		log.Error(err, "Failed to update GenezioManager status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// imageForGenezioManager gets the Operand image which is managed by this controller
// from the GENEZIO_MANAGER_IMAGE environment variable defined in the config/manager/manager.yaml
func imageForGenezioManager() (string, error) {
	var imageEnvVar = "GENEZIO_MANAGER_IMAGE"
	image, found := os.LookupEnv(imageEnvVar)
	if !found {
		return "", fmt.Errorf("unable to find %s environment variable with the image", imageEnvVar)
	}
	return image, nil
}

func labelsForGenezioManager(name string) map[string]string {
	var imageTag string
	image, err := imageForGenezioManager()
	if err == nil {
		imageTag = strings.Split(image, ":")[1]
	}
	return map[string]string{"app.kubernetes.io/name": "GenezioManager",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/part-of":    "genezio-operator",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

// deploymentForGenezioManager returns a GenezioManager Deployment object
func (r *GenezioManagerReconciler) deploymentForGenezioManager(
	geneziomanager *initv1alpha1.GenezioManager) (*appsv1.Deployment, error) {
	ls := labelsForGenezioManager(geneziomanager.Name)
	// Get the Operand image
	image, err := imageForGenezioManager()
	if err != nil {
		return nil, err
	}

	// Extract the git provider configuration
	var gitUser, gitURL, gitPassword, gitToken string
	switch geneziomanager.Spec.GitConfig.Provider {
	case "gitea":
		gitUser = geneziomanager.Spec.GitConfig.Gitea.Username
		gitURL = geneziomanager.Spec.GitConfig.Gitea.URL
		gitPassword = geneziomanager.Spec.GitConfig.Gitea.Password
		gitToken = geneziomanager.Spec.GitConfig.Gitea.Token
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      geneziomanager.Name,
			Namespace: geneziomanager.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						// IMPORTANT: seccomProfile was introduced with Kubernetes 1.19
						// If you are looking for to produce solutions to be supported
						// on lower versions you must remove this option.
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image: image,
						Name:  "genezio-manager",
						Env: []corev1.EnvVar{
							{
								Name:  "REGION",
								Value: geneziomanager.Spec.Region,
							},
							{
								Name:  "DOMAIN",
								Value: "local",
							},
							{
								Name:  "CHART_REPO",
								Value: geneziomanager.Spec.ChartRepo,
							},
							{
								Name:  "CHART_TARGET_REVISION",
								Value: geneziomanager.Spec.ChartRev,
							},
							{
								Name:  "DEPLOYMENT_REPO_NAME",
								Value: geneziomanager.Spec.GitConfig.DeployementRepoName,
							},
							{
								Name:  "ARGOCD_URL",
								Value: geneziomanager.Spec.ArgoCDConfig.URL,
							},
							{
								Name:  "ARGOCD_TOKEN",
								Value: geneziomanager.Spec.ArgoCDConfig.Password,
							},
							// Container registry data
							{
								Name:  "REGISTRY_URL",
								Value: geneziomanager.Spec.ContainerRegistryConfig.URL,
							},
							{
								Name:  "REGISTRY_USER",
								Value: geneziomanager.Spec.ContainerRegistryConfig.Username,
							},
							{
								Name:  "REGISTRY_PASSWORD",
								Value: geneziomanager.Spec.ContainerRegistryConfig.Password,
							},
							// Git data
							{
								Name:  "GIT_USER",
								Value: gitUser,
							},
							{
								Name:  "GIT_URL",
								Value: gitURL,
							},
							{
								Name:  "GIT_PASSWORD",
								Value: gitPassword,
							},
							{
								Name:  "GIT_TOKEN",
								Value: gitToken,
							},
						},

						ImagePullPolicy: corev1.PullAlways,
						// Ensure restrictive context for the container
						// More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
						SecurityContext: &corev1.SecurityContext{
							// WARNING: Ensure that the image used defines an UserID in the Dockerfile
							// otherwise the Pod will not run and will fail with "container has runAsNonRoot and image has non-numeric user"".
							// If you want your workloads admitted in namespaces enforced with the restricted mode in OpenShift/OKD vendors
							// then, you MUST ensure that the Dockerfile defines a User ID OR you MUST leave the "RunAsNonRoot" and
							// "RunAsUser" fields empty.
							RunAsNonRoot: &[]bool{true}[0],
							// The GenezioManager image does not use a non-zero numeric user as the default user.
							// Due to RunAsNonRoot field being set to true, we need to force the user in the
							// container to a non-zero numeric user. We do this using the RunAsUser field.
							// However, if you are looking to provide solution for K8s vendors like OpenShift
							// be aware that you cannot run under its restricted-v2 SCC if you set this value.
							RunAsUser:                &[]int64{1001}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: geneziomanager.Spec.ContainerPort,
							Name:          "genezio-manager",
						}},
					}},
				},
			},
		},
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(geneziomanager, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GenezioManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&initv1alpha1.GenezioManager{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
