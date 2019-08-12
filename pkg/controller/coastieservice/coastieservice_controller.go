package coastieservice

import (
	"context"
	"time"

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

var log = logf.Log.WithName("controller_coastieservice")

// Add creates a new CoastieService Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCoastieService{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("coastieservice-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CoastieService
	err = c.Watch(&source.Kind{Type: &k8sv1alpha1.CoastieService{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CoastieService
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &k8sv1alpha1.CoastieService{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCoastieService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCoastieService{}

// ReconcileCoastieService reconciles a CoastieService object
type ReconcileCoastieService struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CoastieService object and makes changes based on the state read
// and what is in the CoastieService.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCoastieService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CoastieService")

	// Fetch the CoastieService instance
	instance := &k8sv1alpha1.CoastieService{}
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

	// Check for tests
	tests := instance.Spec.Tests
	for _, v := range tests {
		if v == "tcp" {
			err, retry := runTcpTest(instance, r, reqLogger)
			if err != nil {
				return reconcile.Result{}, err
			} else if retry {
				return reconcile.Result{Requeue: true}, nil
			}
		} else if v == "udp" {
			err, retry := runUdpTest(instance, r, reqLogger)
			if err != nil {
				return reconcile.Result{}, err
			} else if retry {
				return reconcile.Result{Requeue: true}, nil
			}
		} else if v == "http" {
			err, retry := runHttpTest(instance, r, reqLogger)
			if err != nil {
				return reconcile.Result{}, err
			} else if retry {
				return reconcile.Result{Requeue: true}, nil
			}
		}
	}

	reqLogger.Info("Reconcile of CoastieService complete")
	// Clean up old deployments
	for _, v := range tests {
		if v == "tcp" {
			err = deleteTcpTest(instance, r, reqLogger)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else if v == "udp" {
			err = deleteUdpTest(instance, r, reqLogger)
			if err != nil {
				return reconcile.Result{}, err
			}
		} else if v == "http" {
			err = deleteHttpTest(instance, r, reqLogger)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

    return reconcile.Result{RequeueAfter: time.Second*300}, nil
}
