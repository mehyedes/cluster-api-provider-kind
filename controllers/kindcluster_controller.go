/*
Copyright 2021.

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

package controllers

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	capiKubeconfig "sigs.k8s.io/cluster-api/util/kubeconfig"
	"sigs.k8s.io/cluster-api/util/patch"
	capiSecret "sigs.k8s.io/cluster-api/util/secret"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	KindClusterService "sigs.k8s.io/kind/pkg/cluster"

	infrastructurev1alpha4 "github.com/mehyedes/cluster-api-provider-kind/api/v1alpha4"
	kindUtil "github.com/mehyedes/cluster-api-provider-kind/util"
)

// KINDClusterReconciler reconciles a KINDCluster object
type KINDClusterReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Kubeconfig   string
	KindProvider *KindClusterService.Provider
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kindclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kindclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kindclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KINDCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *KINDClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// your logic here

	// Fetch the KINDCluster instance
	log.Info("Fetching KINDCluster instance")
	kindCluster := &infrastructurev1alpha4.KINDCluster{}
	err := r.Get(ctx, req.NamespacedName, kindCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the cluster
	log.Info("Fetching the CAPI cluster resource")
	cluster, err := util.GetOwnerCluster(ctx, r.Client, kindCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info("Cluster controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, kindCluster) {
		log.Info("KINDCluster or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	helper, err := patch.NewHelper(kindCluster, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		perr := helper.Patch(ctx, kindCluster)
		if perr != nil {
			log.Error(perr, "Failed while patching KINDCluster resource")
		}
	}()

	// Handle deleted clusters
	if !kindCluster.DeletionTimestamp.IsZero() {
		log.Info("Deleting cluster")
		return reconcileDelete(kindCluster, r.KindProvider, r.Kubeconfig)
	}

	// Handle non-deleted clusters

	// If the KINDCluster doesn't have a ClusterFinalizer, add it.
	controllerutil.AddFinalizer(kindCluster, infrastructurev1alpha4.ClusterFinalizer)

	createOption := KindClusterService.CreateWithKubeconfigPath(r.Kubeconfig)

	// Check if a KIND cluster with the same name already exists
	clusterExists, kerror := kindUtil.AlreadyExists(r.KindProvider, kindCluster.Spec.Name)
	if kerror != nil {
		return ctrl.Result{}, fmt.Errorf(kerror.Error())
	}

	if clusterExists {
		if kindCluster.Spec.ControlPlaneEndpoint != nil {
			log.Info("Cluster up-to-date: nothing to do here..")
			return ctrl.Result{}, nil
		}
		log.Info("A KIND cluster already exists with the same name. Not creating")
		kindCluster.Status.FailureReason = "KIND cluster already exists with the same name"
	}

	log.Info("Creating KIND cluster")

	err = r.KindProvider.Create(kindCluster.Spec.Name, createOption)
	if err != nil {
		// Populate the failureMessage field
		kindCluster.Status.FailureReason = err.Error()
		return ctrl.Result{}, fmt.Errorf("failed to create KIND cluster: %w", err)
	}

	// Set controlPlaneEndpoint for the cluster
	kubeconfigData, err := r.KindProvider.KubeConfig(kindCluster.Spec.Name, false)
	host, port, err := kindUtil.GetControlPlane(kubeconfigData)
	kindCluster.Spec.ControlPlaneEndpoint = &clusterv1.APIEndpoint{
		Host: host,
		Port: port,
	}

	// Add create kubeconfig secret if it doesn't exist
	_, err = capiSecret.GetFromNamespacedName(ctx, r.Client, req.NamespacedName, capiSecret.Kubeconfig)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		if createErr := createKubeconfigSecret(
			ctx,
			cluster,
			r.Client,
			kubeconfigData,
		); createErr != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create kubeconfig secret: %w", err)
		}
	}

	kindCluster.Status.Ready = true

	return ctrl.Result{}, nil
}

func reconcileDelete(kindCluster *infrastructurev1alpha4.KINDCluster, kindProvider *KindClusterService.Provider, kubeconfig string) (reconcile.Result, error) {
	derror := kindProvider.Delete(kindCluster.Spec.Name, kubeconfig)
	if derror != nil {
		return ctrl.Result{}, derror
	}
	controllerutil.RemoveFinalizer(kindCluster, infrastructurev1alpha4.ClusterFinalizer)
	return ctrl.Result{}, nil
}

func createKubeconfigSecret(ctx context.Context, cluster *clusterv1.Cluster, c client.Client, kubeconfigData string) error {
	kubeconfigSecret := capiKubeconfig.GenerateSecret(cluster, []byte(kubeconfigData))

	// Write the secret to the kubernetes cluster
	err := c.Create(ctx, kubeconfigSecret)
	if err != nil {
		return fmt.Errorf("Could not create kubeconfig secret for cluster: %w", err)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KINDClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha4.KINDCluster{}).
		Complete(r)
}
