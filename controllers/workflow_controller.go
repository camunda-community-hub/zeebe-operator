/*

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
	"github.com/go-logr/logr"
	"github.com/zeebe-io/zeebe/clients/go/pkg/pb"
	"github.com/zeebe-io/zeebe/clients/go/pkg/zbc"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "zeebe-operator/api/v1"
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=api.zeebe.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.zeebe.io,resources=workflows/status,verbs=get;update;patch

func (r *WorkflowReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("workflow", req.NamespacedName)

	var workflow apiv1.Workflow
	if err := r.Get(ctx, req.NamespacedName, &workflow); err != nil {
		// it might be not found if this is a delete request
		if ignoreNotFound(err) == nil {
			log.Info("No workflow found: " + req.NamespacedName.Name)

			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch cluster")

		return ctrl.Result{}, err
	}
	// your logic here

	zbClient, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         workflow.Spec.ClusterName + "-zeebe-gateway." + workflow.Spec.ClusterName + ".svc.cluster.local:26500",
		UsePlaintextConnection: true,
	})

	if err != nil {
		panic(err)
	}

	deploy, err := zbClient.NewDeployWorkflowCommand().
		AddResource([]byte(workflow.Spec.WorkflowDefinitionContent),
			workflow.Spec.WorkflowDefinitionName,
			pb.WorkflowRequestObject_BPMN).
		Send(ctx)
	if err != nil {
		panic(err)
	}

	log.Info("Workflow Deployed: ", "key", deploy.Key)

	return ctrl.Result{}, nil
}

func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Workflow{}).
		Complete(r)
}
