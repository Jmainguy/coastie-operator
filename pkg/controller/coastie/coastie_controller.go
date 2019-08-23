package coastie

import (
	"context"
	"strings"
	"time"

	"github.com/go-logr/logr"
	k8sv1alpha1 "github.com/jmainguy/coastie-operator/pkg/apis/k8s/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_coastie")

// Add creates a new Coastie Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCoastie{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("coastie-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Coastie
	err = c.Watch(&source.Kind{Type: &k8sv1alpha1.Coastie{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Coastie
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &k8sv1alpha1.Coastie{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCoastie implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCoastie{}

// ReconcileCoastie reconciles a Coastie object
type ReconcileCoastie struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Coastie object and makes changes based on the state read
// and what is in the Coastie.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCoastie) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Coastie")

	// Fetch the Coastie instance
	instance := &k8sv1alpha1.Coastie{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	runTests(instance, r, reqLogger)
	cleanUpTests(instance, r, reqLogger)
	reqLogger.Info("Reconciliation of Coastie complete")
	// RequeueAfter is not working, its requeing instantly on openshift 3.11
	// For that reason we will just sleep 300 seconds and then try and requeue
	time.Sleep(300 * time.Second)
	return reconcile.Result{
		RequeueAfter: 1 * time.Second,
	}, nil
}

func runTests(instance *k8sv1alpha1.Coastie, r *ReconcileCoastie, reqLogger logr.Logger) {
	// Check for tests
	tests := instance.Spec.Tests
	for _, v := range tests {
		if v == "tcp" {
			reqLogger.Info("Begining Test", "TestName", strings.ToUpper("tcp"))
			retry := true
			for retry {
				retry = runTest("tcp", instance, r, reqLogger)
			}
		} else if v == "udp" {
			reqLogger.Info("Begining Test", "TestName", strings.ToUpper("udp"))
			retry := true
			for retry {
				retry = runTest("udp", instance, r, reqLogger)
			}
		} else if v == "http" {
			reqLogger.Info("Begining Test", "TestName", strings.ToUpper("http"))
			retry := true
			for retry {
				retry = runTest("http", instance, r, reqLogger)
			}
		}
	}

}

func cleanUpTests(instance *k8sv1alpha1.Coastie, r *ReconcileCoastie, reqLogger logr.Logger) {
	// Clean up old deployments
	tests := instance.Spec.Tests
	for _, v := range tests {
		if v == "tcp" {
			reqLogger.Info("Cleaning Up Test", "TestName", strings.ToUpper("tcp"))
			retry := true
			for retry {
				retry = cleanUpTest("tcp", instance, r, reqLogger)
			}
		} else if v == "udp" {
			reqLogger.Info("Cleaning Up Test", "TestName", strings.ToUpper("udp"))
			retry := true
			for retry {
				retry = cleanUpTest("udp", instance, r, reqLogger)
			}
		} else if v == "http" {
			reqLogger.Info("Cleaning Up Test", "TestName", strings.ToUpper("http"))
			retry := true
			for retry {
				retry = cleanUpTest("http", instance, r, reqLogger)
			}
		}
	}

}

func runTest(testName string, instance *k8sv1alpha1.Coastie, r *ReconcileCoastie, reqLogger logr.Logger) (retry bool) {
	switch testName {
	case "tcp":
		err, retry := runTcpUdpTest(instance, r, reqLogger, "tcp")
		if err != nil {
			retry = true
			reqLogger.Error(err, "TCP test encountered an error: ")
		}
		return retry
	case "udp":
		err, retry := runTcpUdpTest(instance, r, reqLogger, "udp")
		if err != nil {
			retry = true
			reqLogger.Error(err, "UDP test encountered an error: ")
		}
		return retry
	case "http":
		err, retry := runHttpTest(instance, r, reqLogger)
		if err != nil {
			retry = true
			reqLogger.Error(err, "HTTP test encountered an error: ")
		}
		return retry
	}
	// Shouldnt reach here unless an an invalid case was passed
	retry = false
	return retry
}

func cleanUpTest(testName string, instance *k8sv1alpha1.Coastie, r *ReconcileCoastie, reqLogger logr.Logger) (retry bool) {
	switch testName {
	case "tcp":
		err := deleteTcpUdpTest(instance, r, reqLogger, "tcp")
		if err != nil {
			retry = true
			reqLogger.Error(err, "TCP Cleanup encountered an error: ")
		}
		return retry
	case "udp":
		err := deleteTcpUdpTest(instance, r, reqLogger, "udp")
		if err != nil {
			retry = true
			reqLogger.Error(err, "UDP Cleanup encountered an error: ")
		}
		return retry
	case "http":
		err := deleteHttpTest(instance, r, reqLogger)
		if err != nil {
			retry = true
			reqLogger.Error(err, "HTTP Cleanup encountered an error: ")
		}
		return retry
	}
	// Shouldnt reach here unless an an invalid case was passed
	retry = false
	return retry
}
