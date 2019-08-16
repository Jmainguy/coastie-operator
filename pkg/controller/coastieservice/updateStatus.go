package coastieservice

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	k8sv1alpha1 "github.com/jmainguy/coastie-operator/pkg/apis/k8s/v1alpha1"
)

func updateCoastieStatus(instance *k8sv1alpha1.CoastieService, TestStatus k8sv1alpha1.TestResult, TestName string, reqLogger logr.Logger, r *ReconcileCoastieService) (err error) {
	if instance.Status.TestResults == nil {
		instance.Status.TestResults = make(map[string]k8sv1alpha1.TestResult)
	}
	instance.Status.TestResults[TestName] = TestStatus
	err = r.client.Status().Update(context.TODO(), instance)
	//err = r.client.Update(context.TODO(), instance)
	if err != nil {
		time.Sleep(5 * time.Second)
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update CoastieService status")
			return err
		}
	}
	return
}
