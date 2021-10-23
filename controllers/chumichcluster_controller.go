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
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrastructurev1alpha4 "github.com/chu-yik/cluster-api-provider-chumich/api/v1alpha4"
	"github.com/pkg/errors"
)

// ChumichClusterReconciler reconciles a ChumichCluster object
type ChumichClusterReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recipient string
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=chumichclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=chumichclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=chumichclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ChumichCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ChumichClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("[MC] Reconcile ChumichCluster")

	// Fetch ChumichCluster
	chumichCluster := &infrastructurev1alpha4.ChumichCluster{}
	if err := r.Get(ctx, req.NamespacedName, chumichCluster); err != nil {
		// handle deletion
		if apierrors.IsNotFound(err) {
			logger.Info("Cluster is deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error reconciling")
		return ctrl.Result{}, err
	}

	// Fetch Cluster
	cluster, err := util.GetOwnerCluster(ctx, r.Client, chumichCluster.ObjectMeta)
	if err != nil {
		logger.Error(err, "Error getting Owner Cluster")
		return ctrl.Result{}, err
	}

	if cluster == nil {
		logger.Info("Cluster Controller has not yet set OwnerRef")
		return ctrl.Result{}, err
	}

	if annotations.IsPaused(cluster, chumichCluster) {
		logger.Info("Paused, won't reconcile")
		return ctrl.Result{}, nil
	}

	if chumichCluster.Status.MessageID != nil {
		logger.Info("We have already reconcilled this cluster:", "messageId", chumichCluster.Status.MessageID)
		return ctrl.Result{}, nil
	}

	subject := fmt.Sprintf("[%s] New Cluster %s requested", chumichCluster.Spec.Priority, cluster.Name)
	body := fmt.Sprintf("Hello! One ChumichCluster please. \n\n%s\n", chumichCluster.Spec.Request)
	logger.Info("Reconciling")
	logger.Info(subject)
	logger.Info(body)

	// patch
	helper, err := patch.NewHelper(chumichCluster, r.Client)
	if err != nil {
		logger.Error(err, "Failed creating patch helper")
		return ctrl.Result{}, err
	}

	messageId := "123456"
	chumichCluster.Status.MessageID = &messageId
	if err := helper.Patch(ctx, chumichCluster); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "couldn't patch cluster %q", chumichCluster.Name)
	}
	logger.Info("Reconcilled using hardcoded ID:", "messageId", messageId)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChumichClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha4.ChumichCluster{}).
		Complete(r)
}
