# Testing

This was written on how I currently build and release Coastie. This will need to get adjusted to be more generic.

## Build
```/bin/bash
export GO111MODULE="on"
operator-sdk build push.soh.re/soh.re/coastie-operator
```

## Push
```/bin/bash
docker push push.soh.re/soh.re/coastie-operator
```

## Pull into cluster
push.soh.re and hub.soh.re resolve to the same domain, one lets me push, the other lets anybody in the world pull.

```/bin/bash
oc delete -f deploy/operator.yaml
oc create -f deploy/operator.yaml
```

## Watch what the operator is doing
```/bin/bash
oc get pods
oc logs -f PODNAME
```
