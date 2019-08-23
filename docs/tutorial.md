# Tutorial

## Installation

### Create a namespace
```/bin/bash
oc new-project coastie
```

### Deploy yamls
```/bin/bash
oc create -f deploy/crds/k8s_v1alpha1_coastie_crd.yaml
oc create -f deploy/cluster_role_openshift.yaml
oc create -f deploy/service_account.yaml
oc create -f deploy/cluster_rolebinding.yaml
oc create -f deploy/operator.yaml
```

Congratulations, the operator is now up and running, and watching all namespaces for the Coastie CustomResource

## Configure Coastie CustomResource to be watched

### Create a namespace
```/bin/bash
oc new-project coastie-test
```
### Edit namespace to support all nodes for the daemonSet
```/bin/bash
oc edit namespace coastie-test

# Set this to allow all nodes to be requested in the metadata.annotations
openshift.io/node-selector: ""
```

### Setup quota
each pod takes .1 cpu and ram, and one pod per node

multiple by number of tests you will be running, so for tcp, udp, and http on a 9 node cluster

you would need 9 x 3 pods (27), 9 x 3 x 0.1 cpu/memory limits and requests (2.7)

```/bin/bash
oc edit quota
# Set the following for the above specs
spec:
  hard:
    limits.cpu: "3"
    limits.memory: 3Gi
    pods: "27"
    requests.cpu: "3"
    requests.memory: 3Gi
```
