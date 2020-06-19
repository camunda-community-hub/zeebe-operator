# Zeebe Kubernetes Operator
The purpose of this component is to facilitate the creation of new Zeebe Clusters and related components. 
This is achieved by defining a new Custom Resource Definition for ZeebeCluster and other types. This allows us to extend Kubernetes capabilities to now understand about Zeebe Clusters. 
This component is responsible for understanding and managing ZeebeClusters CRDs. 

A new cluster can be created after installing the operator by applying the following manifst:
```yaml
apiVersion: zeebe.io/v1
kind: ZeebeCluster
metadata:
  name: my-zeebe-cluster
```

