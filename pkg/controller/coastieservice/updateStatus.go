package coastieservice

import (
    "context"
    "fmt"

    "github.com/go-logr/logr"
    k8sv1alpha1 "github.com/jmainguy/coastie-operator/pkg/apis/k8s/v1alpha1"
)

func updateCoastieStatus(instance *k8sv1alpha1.CoastieService, TestStatus k8sv1alpha1.Test, TestName string, reqLogger logr.Logger, r *ReconcileCoastieService) (err error) {
    fmt.Printf("TestName: %s\n", TestName)
    fmt.Printf("TestStatus: %s\n", TestStatus)
    if instance.Status.Tests == nil {
        fmt.Println("Instance status tests was nil")
        instance.Status.Tests = make(map[string]k8sv1alpha1.Test)
    }
    instance.Status.Tests[TestName] = TestStatus
    fmt.Printf("instance.Status.Tests[%s]: %s\n", TestName, instance.Status.Tests[TestName])
    fmt.Println(instance)
    //err = r.client.Status().Update(context.TODO(), instance)
    err = r.client.Update(context.TODO(), instance)
    fmt.Printf("instance.Status.Tests[%s]: %s\n", TestName, instance.Status.Tests[TestName])
    if err != nil {
        reqLogger.Error(err, "Failed to update CoastieService status")
        return err
    }
    return
}
