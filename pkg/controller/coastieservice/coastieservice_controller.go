package coastieservice

import (
    //"fmt"
	"context"
    "reflect"

	k8sv1alpha1 "github.com/jmainguy/coastie-operator/pkg/apis/k8s/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    appsv1 "k8s.io/api/apps/v1"
    resource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_coastieservice")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

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
	        // Define a new DaemonSet object
        	tcpserver := tcpServer(instance)
        	// Set CoastieService instance as the owner and controller
        	if err := controllerutil.SetControllerReference(instance, tcpserver, r.scheme); err != nil {
        		return reconcile.Result{}, err
        	}

        	// Check if this DaemonSet already exists
        	found := &appsv1.DaemonSet{}
        	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: tcpserver.Name}, found)
        	if err != nil && errors.IsNotFound(err) {
        		reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", tcpserver.Namespace, "DaemonSet.Name", tcpserver.Name)
        		err = r.client.Create(context.TODO(), tcpserver)
        		if err != nil {
        			return reconcile.Result{}, err
        		}
        		// DaemonSet created successfully - return and requeue
        		return reconcile.Result{Requeue: true}, nil
        	} else if err != nil {
        		return reconcile.Result{}, err
        	}

            // Else if No errors, and DS already exists, check its status
            tcpTestStatus := k8sv1alpha1.Test{Name: "tcp", Status: "Fail"}
            if found.Status.DesiredNumberScheduled == found.Status.NumberReady {
                tcpTestStatus = k8sv1alpha1.Test{Name: "tcp", Status: "Pass"}
            }
            if len(instance.Status.Tests) == 0 {
                instance.Status.Tests = append(instance.Status.Tests, tcpTestStatus)
            } else {
                for k, v := range instance.Status.Tests {
                    if v.Name == "tcp" {
                        if !reflect.DeepEqual(tcpTestStatus, instance.Status.Tests[k]) {
                            instance.Status.Tests[k] = tcpTestStatus
       		                err := r.client.Status().Update(context.TODO(), instance)
                       		if err != nil {
       			                reqLogger.Error(err, "Failed to update CoastieService status")
                       			return reconcile.Result{}, err
                       		}
                        }
                    }
                }
            }


            // Delete DaemonSet, test is finished
            //err = r.client.Delete(context.TODO(), tcpserver)
            //if err != nil {
            //    return reconcile.Result{}, err
            //}


        	// tcpserver already exists - don't requeue
        	reqLogger.Info("Skip reconcile of TCP test: DaemonSet already exists", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", found.Name)
        }
    }

	return reconcile.Result{}, nil
}


func tcpServer(cr *k8sv1alpha1.CoastieService) *appsv1.DaemonSet {
    name := "tcpserver"
    return &appsv1.DaemonSet{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: cr.Namespace,
        },
        Spec: appsv1.DaemonSetSpec{
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": name,
                },
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app": name,
                    },
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:    name,
                            Image:   "hub.soh.re/jmainguy/tcpserver",
                            Ports: []corev1.ContainerPort{
                                {
                                    ContainerPort: 8081,
                                },
                            },
                            Resources: corev1.ResourceRequirements{
                                Limits: corev1.ResourceList{
                                    "cpu": resource.MustParse("0.5"),
                                    "memory": resource.MustParse("100M"),
                                },
                                Requests: corev1.ResourceList{
                                    "cpu": resource.MustParse("0.5"),
                                    "memory": resource.MustParse("100M"),
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *k8sv1alpha1.CoastieService) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
