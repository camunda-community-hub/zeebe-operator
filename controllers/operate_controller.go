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
	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/test/builder"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	zeebev1 "zeebe-operator/api/v1"
)

// OperateReconciler reconciles a Operate object
type OperateReconciler struct {
	Scheme *runtime.Scheme
	client.Client
	Log logr.Logger
	pr  PipelineRunner
	k8s kubernetes.Clientset
}

func (p *PipelineRunner) createTaskAndTaskRunInstallOperate(namespace string, operate zeebev1.Operate, r OperateReconciler) error {
	log := p.Log.WithValues("createTaskAndRunOperate", namespace)
	task := builder.Task("install-task-operate-"+operate.Name, namespace,
		builder.TaskSpec(
			builder.TaskInputs(builder.InputsResource("operate-version-stream", "git")),
			builder.Step("clone-base-helm-chart-operate", builderImage,
				builder.StepCommand("make", "-C", "/workspace/operate-version-stream/", "build", "install"),
				builder.StepEnvVar("OPERATE", operate.Name),
				builder.StepEnvVar("CLUSTER_NAME", operate.Spec.ZeebeClusterName),
				builder.StepEnvVar("NAMESPACE", operate.Spec.ZeebeClusterName))))

	if err := ctrl.SetControllerReference(&operate, task, r.Scheme); err != nil {
		log.Error(err, "unable set owner to task")
	}

	_, errorTask := p.tekton.TektonV1alpha1().Tasks(namespace).Create(task)
	if errorTask != nil {
		log.Error(errorTask, "Error Creating task")
	}

	log.Info("> Creating Task: ", "task", task)

	taskRun := builder.TaskRun("install-task-run-operate-"+operate.Name, namespace,
		builder.TaskRunSpec(
			builder.TaskRunServiceAccountName(pipelinesServiceAccountName),
			builder.TaskRunDeprecatedServiceAccount(pipelinesServiceAccountName, pipelinesServiceAccountName), // This require a SA being created for it to run

			builder.TaskRunTaskRef("install-task-operate-"+operate.Name),
			builder.TaskRunInputs(builder.TaskRunInputsResource("operate-version-stream",
				builder.TaskResourceBindingRef("operate-version-stream")))))

	if err := ctrl.SetControllerReference(&operate, taskRun, r.Scheme); err != nil {
		log.Error(err, "unable set owner to taskRun")
	}
	log.Info("> Creating TaskRun: ", "taskrun", taskRun)
	_, errorTaskRun := p.tekton.TektonV1alpha1().TaskRuns(namespace).Create(taskRun)

	if errorTaskRun != nil {
		log.Error(errorTaskRun, "Error Creating taskRun")
	}

	return p.WaitForTaskRunState("install-task-run-operate-"+operate.Name, TaskRunSucceed("install-task-run-operate-"+operate.Name), "TaskRunSucceed")
}

// +kubebuilder:rbac:groups=zeebe.io,resources=operates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zeebe.io,resources=operates/status,verbs=get;update;patch

func (r *OperateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	log := r.Log.WithValues(">>> Reconcile: operate", req.NamespacedName)
	var operate zeebev1.Operate

	req.Namespace = pipelinesNamespace
	if err := r.Get(ctx, req.NamespacedName, &operate); err != nil {
		// it might be not found if this is a delete request
		if ignoreNotFound(err) == nil {
			log.Info("Hey there.. deleting operate happened: " + req.NamespacedName.Name)

			//r.pr.createTaskAndTaskRunDelete(req.NamespacedName.Name, req.Namespace)

			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch operate")

		return ctrl.Result{}, err
	}

	var operateName = req.Name
	// Check if tasks needs to be created .. if not avoid
	if !r.pr.checkForTask("install-task-" + operateName) { // If the task was created before avoid creating it again
		// Check if pipeline runner was initialized before
		r.pr.initPipelineRunner(pipelinesNamespace)


		if err := r.pr.createTaskAndTaskRunInstallOperate(pipelinesNamespace, operate, *r); err != nil {
			//setCondition(&operate.Status.Conditions, zeebev1.StatusCondition{
			//	Type:    "InstallationFailed",
			//	Status:  zeebev1.ConditionStatusUnhealthy,
			//	Reason:  "Installation Pipelines Failed",
			//	Message: "Zeebe Cluster Installation Failed",
			//})
			operate.Status.StatusName = "FailedToInstall"
		}

	}

	return ctrl.Result{}, nil
}

func (r *OperateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// create the clientset
	clientSet, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		panic(err.Error())
	}
	r.k8s = *clientSet

	tektonClientSet, _ := tekton.NewForConfig(mgr.GetConfig())
	r.pr.tekton = *tektonClientSet
	r.pr.Log = r.Log
	r.Scheme = mgr.GetScheme()
	return ctrl.NewControllerManagedBy(mgr).
		For(&zeebev1.Operate{}).
		Complete(r)
}
