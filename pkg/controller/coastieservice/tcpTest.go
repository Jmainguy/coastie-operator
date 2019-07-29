package coastieservice

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	k8sv1alpha1 "github.com/jmainguy/coastie-operator/pkg/apis/k8s/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func runTcpTest(instance *k8sv1alpha1.CoastieService, r *ReconcileCoastieService, reqLogger logr.Logger) (err error, retry bool) {
	retry = false
	// Define a new DaemonSet object
	tcpserver := tcpServer(instance)
	// Set CoastieService instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, tcpserver, r.scheme); err != nil {
		return err, retry
	}

	// Check if this DaemonSet already exists
	found := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: tcpserver.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", tcpserver.Namespace, "DaemonSet.Name", tcpserver.Name)
		err = r.client.Create(context.TODO(), tcpserver)
		if err != nil {
			return err, retry
		}
		// DaemonSet created successfully - return and requeue
		retry = true
		return nil, retry
	} else if err != nil {
		return err, retry
	}

	// Else if No errors, and DS already exists, check its status
	tcpTestStatus := k8sv1alpha1.Test{Name: "tcp", Status: "Fail"}
	if found.Status.DesiredNumberScheduled == found.Status.NumberReady {
		// All pods are now running, run test against them
		// Spin up service
		tcpServerService := tcpServerService(instance)
		// Set CoastieService instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, tcpServerService, r.scheme); err != nil {
			return err, retry
		}
		// Check if Service exists
		err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: tcpServerService.Name}, tcpServerService)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating a new Service", "Service.Namespace", tcpServerService.Namespace, "Service.Name", tcpServerService.Name)
			err = r.client.Create(context.TODO(), tcpServerService)
			if err != nil {
				return err, retry
			}
			// Service created successfully - return and requeue
			retry = true
			return nil, retry
		}
		// Service Exists, how do we connect to it?
		tcpServerClusterIP := tcpServerService.Spec.ClusterIP
		// Use client to connect to service, try 5 times if fail
		// If this is still true later, fail with message
		tcpFail := true
		tcpStatus := ""
		for i := 0; i < 5; i++ {
			tcpStatus = tcpClient(tcpServerClusterIP, "8081")
			if strings.Contains(tcpStatus, "SUCCESS") {
				tcpTestStatus = k8sv1alpha1.Test{Name: "tcp", Status: "Pass"}
				tcpFail = false
				// Exit loop
				i = 5
			} else {
				// Pods are running, but failing test, give them a few seconds
				time.Sleep(2 * time.Second)
			}
		}
		if tcpFail {
			tcpTestStatus = k8sv1alpha1.Test{Name: "tcp", Status: "Fail"}
			message := fmt.Sprintf("Coastie Operator: TCP Test failed. %s", tcpStatus)
			// Alarm slack if failed
			notifySlack(instance.Spec.SlackToken, instance.Spec.SlackChannelID, message)
			// Requeue
			return nil, retry
		}
	} else {
		reqLogger.Info("DaemonSet is not ready", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", found.Name)
		retry = true
		return nil, retry
	}
	if len(instance.Status.Tests) == 0 {
		instance.Status.Tests = append(instance.Status.Tests, tcpTestStatus)
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update CoastieService status")
			return err, retry
		}
	} else {
		for k, v := range instance.Status.Tests {
			if v.Name == "tcp" {
				if !reflect.DeepEqual(tcpTestStatus, instance.Status.Tests[k]) {
					instance.Status.Tests[k] = tcpTestStatus
					err := r.client.Status().Update(context.TODO(), instance)
					if err != nil {
						reqLogger.Error(err, "Failed to update CoastieService status")
						return err, retry
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
	//reqLogger.Info("Skip reconcile of TCP test: DaemonSet already exists", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", found.Name)
	reqLogger.Info("Reached end of TCPTest", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", found.Name)
	return nil, retry
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
							Name:  name,
							Image: "hub.soh.re/jmainguy/tcpserver",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8081,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("0.5"),
									"memory": resource.MustParse("100M"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("0.5"),
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

func tcpServerService(cr *k8sv1alpha1.CoastieService) *corev1.Service {
	name := "tcpserver"
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "tcpserver",
					Protocol: "TCP",
					Port:     8081,
				},
			},
			Selector: map[string]string{
				"app": name,
			},
			//Type: "NodePort",
		},
	}
}

func tcpClient(ip, port string) (status string) {
	// Node + port
	uri := fmt.Sprintf("%s:%s", ip, port)
	// Connect
	c, err := net.Dial("tcp", uri)
	if err != nil {
		status = "ERROR: TCP unable to connect"
		return
	}
	// Send message
	question := fmt.Sprintln("Annie, are you ok?\n")
	c.Write([]byte(question))
	// Read response
	message, _ := bufio.NewReader(c).ReadString('\n')
	if message == "So, Annie are you ok?\n" {
		c.Close()
		status = "SUCCESS: TCP is working"
		return
	} else if message != "" {
		c.Close()
		status = fmt.Sprintf("ERROR: TCP Failed - Server: %s", message)
		return
	}
	status = "ERROR: Should never reach this"
	return
}