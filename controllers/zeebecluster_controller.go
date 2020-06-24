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
	"fmt"
	"github.com/go-logr/logr"
	uuid2 "github.com/google/uuid"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/test/builder"
	"github.com/zeebe-io/zeebe/clients/go/pkg/pb"
	"github.com/zeebe-io/zeebe/clients/go/pkg/zbc"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
	"time"
	zeebev1 "zeebe-operator/api/v1"
)

// ZeebeClusterReconciler reconciles a ZeebeCluster object
type ZeebeClusterReconciler struct {
	Scheme *runtime.Scheme
	client.Client
	k8s kubernetes.Clientset
	pr  PipelineRunner
	Log logr.Logger
}

type PipelineRunner struct {
	tekton tekton.Clientset
	Log    logr.Logger
}

// TaskRunStateFn is a condition function on TaskRun used polling functions
type TaskRunStateFn func(r *v1alpha1.TaskRun) (bool, error)

const (
	interval = 1 * time.Second
	timeout  = 10 * time.Minute
)

var pipelinesNamespace = os.Getenv("PIPELINES_NAMESPACE") // This is related to the Role, RB, and SA to run pipelines
var pipelinesServiceAccountName = os.Getenv("PIPELINES_SA")
var builderImage = os.Getenv("BUILDER_IMAGE") //old one gcr.io/jenkinsxio/builder-go:2.0.1028-359
var versionStream = os.Getenv("VERSION_STREAM")

func (p *PipelineRunner) checkForTask(name string) bool {
	options := metav1.GetOptions{}
	t, err := p.tekton.TektonV1alpha1().Tasks(pipelinesNamespace).Get(name, options)
	if err == nil && t != nil {
		return true
	}
	return false
}

//@TODO:  Do as initialization of the Operator ..
func (p *PipelineRunner) initPipelineRunner(namespace string) {
	log := p.Log.WithValues("pipelineresource", namespace)

	pipelineResourceZeebeCluster := builder.PipelineResource("zeebe-version-stream", namespace,
		builder.PipelineResourceSpec(v1alpha1.PipelineResourceType("git"),
			builder.PipelineResourceSpecParam("revision", "master"),
			builder.PipelineResourceSpecParam("url", versionStream)))

	log.Info("> Creating PipelineResource for ZeebeCluster: ", "pipelineResourceZeebeCluster", pipelineResourceZeebeCluster)
	p.tekton.TektonV1alpha1().PipelineResources(namespace).Create(pipelineResourceZeebeCluster)

	//@TODO: END
}

func (p *PipelineRunner) createTaskAndTaskRunInstall(namespace string, zeebeCluster zeebev1.ZeebeCluster, r ZeebeClusterReconciler) error {
	log := p.Log.WithValues("createTaskAndRun", namespace)
	task := builder.Task("install-task-"+zeebeCluster.Name, namespace,
		builder.TaskSpec(
			builder.TaskInputs(builder.InputsResource("zeebe-version-stream", "git")),
			builder.Step("clone-base-helm-chart", builderImage,
				builder.StepCommand("make", "-C", "/workspace/zeebe-version-stream/", "build", "install"),
				builder.StepEnvVar("CLUSTER_NAME", zeebeCluster.Name),
				builder.StepEnvVar("OPERATE_ENABLED", strconv.FormatBool(zeebeCluster.Spec.OperateEnabled)),
				builder.StepEnvVar("NAMESPACE", zeebeCluster.Name))))

	if err := ctrl.SetControllerReference(&zeebeCluster, task, r.Scheme); err != nil {
		log.Error(err, "unable set owner to task")
	}

	_, errorTask := p.tekton.TektonV1alpha1().Tasks(namespace).Create(task)
	if errorTask != nil {
		log.Error(errorTask, "Error Creating task")
	}

	log.Info("> Creating Task: ", "task", task)

	taskRun := builder.TaskRun("install-task-run-"+zeebeCluster.Name, namespace,
		builder.TaskRunSpec(
			builder.TaskRunServiceAccountName(pipelinesServiceAccountName),
			builder.TaskRunDeprecatedServiceAccount(pipelinesServiceAccountName, pipelinesServiceAccountName), // This require a SA being created for it to run

			builder.TaskRunTaskRef("install-task-"+zeebeCluster.Name),
			builder.TaskRunInputs(builder.TaskRunInputsResource("zeebe-version-stream",
				builder.TaskResourceBindingRef("zeebe-version-stream")))))

	if err := ctrl.SetControllerReference(&zeebeCluster, taskRun, r.Scheme); err != nil {
		log.Error(err, "unable set owner to taskRun")
	}
	log.Info("> Creating TaskRun: ", "taskrun", taskRun)
	_, errorTaskRun := p.tekton.TektonV1alpha1().TaskRuns(namespace).Create(taskRun)

	if errorTaskRun != nil {
		log.Error(errorTaskRun, "Error Creating taskRun")
	}

	return p.WaitForTaskRunState("install-task-run-"+zeebeCluster.Name, TaskRunSucceed("install-task-run-"+zeebeCluster.Name), "TaskRunSucceed")
}

func (p *PipelineRunner) createTaskAndTaskRunDelete(release string, namespace string) error {
	log := p.Log.WithValues("zeebecluster", namespace)
	uuid, _ := uuid2.NewUUID()
	task := builder.Task("delete-task-"+release+"-"+uuid.String(), namespace,
		builder.TaskSpec(
			builder.TaskInputs(builder.InputsResource("zeebe-version-stream", "git")),
			builder.Step("clone-base-helm-chart", builderImage,
				builder.StepCommand("make", "-C", "/workspace/zeebe-version-stream/", "delete"),
				builder.StepEnvVar("CLUSTER_NAME", release))))

	_, errorTask := p.tekton.TektonV1alpha1().Tasks(namespace).Create(task)
	if errorTask != nil {
		log.Error(errorTask, "Error Creating task")
	}

	log.Info("> Creating Task: ", "task", task)

	taskRun := builder.TaskRun("delete-task-run-"+release, namespace,
		builder.TaskRunSpec(
			builder.TaskRunServiceAccountName(pipelinesServiceAccountName),
			builder.TaskRunDeprecatedServiceAccount(pipelinesServiceAccountName, pipelinesServiceAccountName), // This require a SA being created for it to run

			builder.TaskRunTaskRef("delete-task-"+release+"-"+uuid.String()),
			builder.TaskRunInputs(builder.TaskRunInputsResource("zeebe-version-stream",
				builder.TaskResourceBindingRef("zeebe-version-stream")))))

	log.Info("> Creating TaskRun: ", "taskrun", taskRun)
	_, errorTaskRun := p.tekton.TektonV1alpha1().TaskRuns(namespace).Create(taskRun)

	if errorTaskRun != nil {
		log.Error(errorTaskRun, "Error Creating taskRun")
	}

	return p.WaitForTaskRunState("delete-task-run-"+release, TaskRunSucceed("delete-task-run-"+release), "TaskRunSucceed")
}

func (p *PipelineRunner) WaitForTaskRunState(name string, inState TaskRunStateFn, desc string) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		r, err := p.tekton.TektonV1alpha1().TaskRuns(pipelinesNamespace).Get(name, metav1.GetOptions{})
		p.Log.Info("> Checking for Task Run Succeed! " + name)
		if err != nil {
			return true, err
		}
		return inState(r)
	})

}

func (p *PipelineRunner) cleanUpTaskAndTaskRun(clusterName string) {
	options := new(metav1.DeleteOptions)
	errorTask := p.tekton.TektonV1alpha1().Tasks(pipelinesNamespace).Delete("delete-task-"+clusterName, options)

	if errorTask != nil {
		p.Log.Error(errorTask, "Error Deleting task", "task", "delete-task-"+clusterName)
	}
	errorTaskRun := p.tekton.TektonV1alpha1().TaskRuns(pipelinesNamespace).Delete("delete-task-run-"+clusterName, options)

	if errorTaskRun != nil {
		p.Log.Error(errorTaskRun, "Error Deleting taskRun", "taskrun", "delete-task-run-"+clusterName)
	}
}

// TaskRunSucceed provides a poll condition function that checks if the TaskRun
// has successfully completed.
func TaskRunSucceed(name string) TaskRunStateFn {
	return func(tr *v1alpha1.TaskRun) (bool, error) {
		c := tr.Status.GetCondition(apis.ConditionSucceeded)
		if c != nil {
			if c.Status == coreV1.ConditionTrue {
				return true, nil
			} else if c.Status == coreV1.ConditionFalse {
				return true, fmt.Errorf("task run %q failed!", name)
			}
		}
		return false, nil
	}
}

// TaskRunFailed provides a poll condition function that checks if the TaskRun
// has failed.
func TaskRunFailed(name string) TaskRunStateFn {
	return func(tr *v1alpha1.TaskRun) (bool, error) {
		c := tr.Status.GetCondition(apis.ConditionSucceeded)
		if c != nil {
			if c.Status == coreV1.ConditionTrue {
				return true, fmt.Errorf("task run %q succeeded!", name)
			} else if c.Status == coreV1.ConditionFalse {
				return true, nil
			}
		}
		return false, nil
	}
}

/* Reconcile should do:
	1) get CRD Cluster
    2) run pipeline to install/update
    3) update CRD status based on pods
    4) update URL
*/

// +kubebuilder:rbac:groups=zeebe.io,resources=zeebeclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zeebe.io,resources=zeebeclusters/status,verbs=get;update;patch

// CRUD core: namespaces, events, secrets, services and configmaps
// +kubebuilder:rbac:groups=core,resources=services;configmaps;namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status;configmaps/status,verbs=get

// LIST core: endpoints
// +kubebuilder:rbac:groups=core,resources=endpoints;pods,verbs=list;watch

// CRUD apps: deployments and statefulsets
// +kubebuilder:rbac:groups=apps,resources=statefulsets;deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status;deployments/status,verbs=get

// CRUD tekton: tasks / taskruns
// +kubebuilder:rbac:groups=tekton.dev,resources=tasks;taskruns;pipelineresources,verbs=get;list;watch;create;update;patch;delete

func (r *ZeebeClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues(">>> Reconcile: zeebecluster", req.NamespacedName)
	var zeebeCluster zeebev1.ZeebeCluster
	req.Namespace = pipelinesNamespace
	if err := r.Get(ctx, req.NamespacedName, &zeebeCluster); err != nil {
		// it might be not found if this is a delete request
		if ignoreNotFound(err) == nil {
			log.Info("Hey there.. deleting cluster happened: " + req.NamespacedName.Name)

			r.pr.createTaskAndTaskRunDelete(req.NamespacedName.Name, req.Namespace)

			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch cluster")

		return ctrl.Result{}, err
	}

	// process the request, make some changes to the cluster, // set some status on `zeebeCluster`, etc
	// update status, since we probably changed it above
	log.Info("> Zeebe Cluster: ", "cluster", zeebeCluster)

	//create namespace if required
	namespace := new(coreV1.Namespace)
	namespace.SetName(zeebeCluster.GetName())
	_, err := ctrl.CreateOrUpdate(ctx, r.Client, namespace, func() error {
		//util.AppendLabels(namespace, zb)
		return ctrl.SetControllerReference(&zeebeCluster, namespace, r.Scheme)
	})
	if err != nil {
		log.Error(err, "unable to create namespace")
		return ctrl.Result{}, err
	}

	var clusterName = zeebeCluster.Name

	// Check if tasks needs to be created .. if not avoid
	if !r.pr.checkForTask("install-task-" + clusterName) { // If the task was created before avoid creating it again
		// Check if pipeline runner was initialized before
		r.pr.initPipelineRunner(pipelinesNamespace)

		if err := r.pr.createTaskAndTaskRunInstall(pipelinesNamespace, zeebeCluster, *r); err != nil {
			setCondition(&zeebeCluster.Status.Conditions, zeebev1.StatusCondition{
				Type:    "InstallationFailed",
				Status:  zeebev1.ConditionStatusUnhealthy,
				Reason:  "Installation Pipelines Failed",
				Message: "Zeebe Cluster Installation Failed",
			})
			zeebeCluster.Status.StatusName = "FailedToInstall"
		}

	}

	zeebeCluster.Status.ClusterName = clusterName

	log.Info("> Zeebe Cluster Name: " + clusterName)
	if zeebeCluster.Status.StatusName != "FailedToInstall" {
		if len(zeebeCluster.Spec.StatefulSetName) > 0 {
			var statefulSet appsV1.StatefulSet
			var statefulSetNamespacedName types.NamespacedName
			statefulSetNamespacedName.Name = zeebeCluster.Spec.StatefulSetName
			statefulSetNamespacedName.Namespace = namespace.Name
			if err := r.Get(ctx, statefulSetNamespacedName, &statefulSet); err != nil {
				// it might be not found if this is a delete request
				if ignoreNotFound(err) == nil {
					log.Error(err, "Not Found! ")
					return ctrl.Result{}, nil
				}
				log.Error(err, "unable to fetch cluster")

				return ctrl.Result{}, err
			}
			r.Log.Info("Found StatefulSet replicas: ", "Replicas: ", statefulSet.Status.Replicas)
			r.Log.Info("Found StatefulSet replicas: ", "readyReplicas: ", statefulSet.Status.ReadyReplicas)
			if statefulSet.Status.ReadyReplicas == statefulSet.Status.Replicas {
				setCondition(&zeebeCluster.Status.Conditions, zeebev1.StatusCondition{
					Type:    "Ready",
					Status:  zeebev1.ConditionStatusHealthy,
					Reason:  fmt.Sprintf("%s%d/%d", "Replicas ", statefulSet.Status.ReadyReplicas, statefulSet.Status.Replicas),
					Message: "Zeebe Cluster Ready",
				})
				zeebeCluster.Status.StatusName = "Ready"

				if zeebeCluster.Spec.ZeebeHealthChecks {
					zbClient, err := zbc.NewClient(&zbc.ClientConfig{
						GatewayAddress:         clusterName + "-zeebe-gateway." + clusterName + ".svc.cluster.local:26500",
						UsePlaintextConnection: true,
					})

					if err != nil {
						panic(err)
					}

					topology, err := zbClient.NewTopologyCommand().Send(ctx)
					if err != nil {
						panic(err)
					}

					for _, broker := range topology.Brokers {
						fmt.Println("Broker", broker.Host, ":", broker.Port)
						for _, partition := range broker.Partitions {
							fmt.Println("  Partition", partition.PartitionId, ":", roleToString(partition.Role))
						}
					}
				}

			} else {
				setCondition(&zeebeCluster.Status.Conditions, zeebev1.StatusCondition{
					Type:    "Pending",
					Status:  zeebev1.ConditionStatusUnhealthy,
					Reason:  fmt.Sprintf("%s%d/%d", "Replicas ", statefulSet.Status.ReadyReplicas, statefulSet.Status.Replicas),
					Message: "Zeebe Cluster Starting",
				})
				zeebeCluster.Status.StatusName = fmt.Sprint("Pending ", statefulSet.Status.ReadyReplicas, "/", statefulSet.Status.Replicas)
			}
		} else {
			setCondition(&zeebeCluster.Status.Conditions, zeebev1.StatusCondition{
				Type:    "Creating",
				Status:  zeebev1.ConditionStatusUnhealthy,
				Reason:  "Booting..",
				Message: "Zeebe Cluster Being Created",
			})
			zeebeCluster.Status.StatusName = "Creating"
		}

	}

	if err := r.Status().Update(ctx, &zeebeCluster); err != nil {
		log.Error(err, "unable to update cluster spec")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func roleToString(role pb.Partition_PartitionBrokerRole) string {
	switch role {
	case pb.Partition_LEADER:
		return "Leader"
	case pb.Partition_FOLLOWER:
		return "Follower"
	default:
		return "Unknown"
	}
}

func (r *ZeebeClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {

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
		For(&zeebev1.ZeebeCluster{}).
		Owns(&coreV1.ConfigMap{}).
		Owns(&coreV1.Service{}).
		Owns(&appsV1.StatefulSet{}).
		Watches(&source.Kind{Type: &appsV1.StatefulSet{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []ctrl.Request {
				statefulSet, ok := obj.Object.(*appsV1.StatefulSet)
				if !ok {
					r.Log.Info("ERROR: unexpected type")
				}

				var zeebeClusterList zeebev1.ZeebeClusterList
				if err := r.List(context.Background(), &zeebeClusterList); err != nil {
					r.Log.Info("unable to get zeebe clusters for statefulset", "statefulset", obj.Meta.GetName())
					return nil
				}

				for i := 0; i < len(zeebeClusterList.Items); i++ {
					r.Log.Info("Comparing: clusterName =  " + zeebeClusterList.Items[i].Name + " -> statefulSet labels: " + statefulSet.GetLabels()["app.kubernetes.io/instance"])
					if zeebeClusterList.Items[i].Name == statefulSet.GetLabels()["app.kubernetes.io/instance"] {
						// I need to set up the ownership to be notified about the changes on the replicas
						if statefulSet.OwnerReferences == nil {
							r.Log.Info("Zeebe Cluster found, updating statefulset ownership ",
								"cluster", zeebeClusterList.Items[i].Name,
								"namespace", zeebeClusterList.Items[i].Namespace)
							_, err := ctrl.CreateOrUpdate(context.Background(), r.Client, statefulSet, func() error {

								//Set Ownership
								ctrl.SetControllerReference(&zeebeClusterList.Items[i], statefulSet, r.Scheme)
								return nil
							})
							if err != nil {
								r.Log.Error(err, "Error setting up owner for statefulset", "cluster", zeebeClusterList.Items[0].Name, "namespace", zeebeClusterList.Items[0].Namespace)
							}
						}

						if len(zeebeClusterList.Items[0].Spec.StatefulSetName) == 0 {
							r.Log.Info("Zeebe Cluster found, updating statefulset reference ",
								"cluster", zeebeClusterList.Items[i].Name,
								"namespace", zeebeClusterList.Items[i].Namespace,
								"statefulSet Name", statefulSet.Name)
							ctrl.CreateOrUpdate(context.Background(), r.Client, &zeebeClusterList.Items[i], func() error {

								//Set Ownership
								zeebeClusterList.Items[i].Spec.ServiceName = zeebeClusterList.Items[i].Name + "-zeebe"
								zeebeClusterList.Items[i].Spec.StatefulSetName = statefulSet.Name
								return nil
							})
							if err != nil {
								r.Log.Error(err, "Error assigning statefulset to cluster")
							}
							return nil
						}

					}
				}

				return nil

			}),
		}).
		Complete(r)
}

func ignoreNotFound(err error) error {
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func setCondition(conds *[]zeebev1.StatusCondition, targetCond zeebev1.StatusCondition) {
	var outCond *zeebev1.StatusCondition
	for i, cond := range *conds {
		if cond.Type == targetCond.Type {
			outCond = &(*conds)[i]
			break
		}
	}
	if outCond == nil {
		*conds = append(*conds, targetCond)
		outCond = &(*conds)[len(*conds)-1]
		outCond.LastTransitionTime = metav1.Now()
	} else {
		lastState := outCond.Status
		lastTrans := outCond.LastTransitionTime
		*outCond = targetCond
		if outCond.Status != lastState {
			outCond.LastTransitionTime = metav1.Now()
		} else {
			outCond.LastTransitionTime = lastTrans
		}
	}

	outCond.LastProbeTime = metav1.Now()
}

func ownedByOther(obj metav1.Object, apiVersion schema.GroupVersion, kind, name string) *metav1.OwnerReference {
	if ownerRef := metav1.GetControllerOf(obj); ownerRef != nil && (ownerRef.Name != name || ownerRef.Kind != kind || ownerRef.APIVersion != apiVersion.String()) {
		return ownerRef
	}
	return nil
}
