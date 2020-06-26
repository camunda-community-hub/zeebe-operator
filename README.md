# Zeebe Kubernetes Operator
The purpose of this component is to facilitate the creation of new Zeebe Clusters and related components. 
This is achieved by defining a new Custom Resource Definition for ZeebeCluster and other types. This allows us to extend Kubernetes capabilities to now understand about Zeebe Clusters. 
This component is responsible for understanding and managing ZeebeClusters CRDs. 

## Installation
The Zeebe Kubernetes Operator can be installed using Helm

> helm repo add zeebe http://helm.zeebe.io
> helm install zeebe-operator zeebe/zeebe-operator

## Basic Usage

The Zeebe Kubernetes Operator was designed to provision and monitor multiple Zeebe Clusters.

A new cluster can be created after installing the operator by applying the following manifest:
``` simple-zeebe-cluster.yaml
apiVersion: zeebe.io/v1
kind: ZeebeCluster
metadata:
  name: my-zeebe-cluster
```

You can do that by running: 
> kubectl apply -f simple-zeebe-cluster.yaml

You can list all the Zeebe Clusters with:
> kubectl get zb

If you want to delete an existing Zeebe Cluster you can just delete the `zb` resource:

> kubectl delete zb my-zeebe-cluster

Each ZeebeCluster is created on its own `Kubernetes Namespace` named after the ZeebeCluster resource (in this case `my-zeebe-cluster`) that means that you can list all the PODS for the cluster and associated resources by running
> kubectl get pods -n my-zeebe-cluster

## Architecture

From an architectural point of view the Zeebe Kubernetes Operator tries to reuse as much as possible existing resources and tools to provision Zeebe Cluster and associated components. 
Some of the tools that are used are:
- KubeBuilder v2 
- Tekton CD
- Helm 
- Zeebe Helm Charts
- Third Party Helm Charts such as: ElasticSearch, Kibana, Prometheus. 

The architecture of the Operator is quite simple, it creates Tekton Pipelines to Deploy the existing Zeebe Helm Charts into the cluster where the Operator is installed.
 
It does this by using a version stream repository (source of truth) which define which versions of the charts needs to be used to provision a concrete cluster. 
This Version Stream repository contains a parent chart that will be installed when a cluster wants to be provisioned. You can find this repository [here](https://github.com/zeebe-io/zeebe-version-stream-helm)

Once the cluster is provisioned (by installing the [Zeebe Helm Charts](http://helm.zeebe.io)) the Operator is in charge of monitoring the resources that were created. For a Zeebe Cluster this are: 
- Broker(s): one or more brokers created by an StatefulSet
- Gateway: Standalone Gateway which is created by a Deployment
- Operate: If enabled a Deployment is created



## References and Links
- [First iteration Blog Post](https://salaboy.com/2019/12/20/zeebe-kubernetes-operator/)
  





