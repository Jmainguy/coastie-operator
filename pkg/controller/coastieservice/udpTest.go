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

func runUdpTest(instance *k8sv1alpha1.CoastieService, r *ReconcileCoastieService, reqLogger logr.Logger) (err error, retry bool) {
	retry = false
	name := fmt.Sprintf("%s-udp", instance.Name)
	// Define a new DaemonSet object
	udpDaemonSet := udpServer(instance, name)
	// Set CoastieService instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, udpDaemonSet, r.scheme); err != nil {
		return err, retry
	}

	// Check if this DaemonSet already exists
	found := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: name}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", udpDaemonSet.Namespace, "DaemonSet.Name", name)
		err = r.client.Create(context.TODO(), udpDaemonSet)
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
	udpTestStatus := k8sv1alpha1.Test{Name: "udp", Status: "Fail"}
	if found.Status.DesiredNumberScheduled == found.Status.NumberReady {
		// All pods are now running, run test against them
		// Spin up service
		udpService := udpServerService(instance, name)
		// Set CoastieService instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, udpService, r.scheme); err != nil {
			return err, retry
		}
		// Check if Service exists
		err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: name}, udpService)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating a new Service", "Service.Namespace", udpService.Namespace, "Service.Name", name)
			err = r.client.Create(context.TODO(), udpService)
			if err != nil {
				return err, retry
			}
			// Service created successfully - return and requeue
			retry = true
			return nil, retry
		}
		// Service Exists, how do we connect to it?
		udpServerClusterIP := udpService.Spec.ClusterIP
		// Use client to connect to service, try 5 times if fail
		// If this is still true later, fail with message
		udpFail := true
		udpStatus := ""
		for i := 0; i < 5; i++ {
			udpStatus = udpClient(udpServerClusterIP, "8082")
			if strings.Contains(udpStatus, "SUCCESS") {
				udpTestStatus = k8sv1alpha1.Test{Name: "udp", Status: "Pass"}
				udpFail = false
				// Exit loop
				i = 5
			} else {
				// Pods are running, but failing test, give them a few seconds
				time.Sleep(2 * time.Second)
			}
		}
		if udpFail {
			udpTestStatus = k8sv1alpha1.Test{Name: "udp", Status: "Fail"}
			message := fmt.Sprintf("Coastie Operator: UDP Test failed. %s", udpStatus)
			// Alarm slack if failed
			err := notifySlack(instance.Spec.SlackToken, instance.Spec.SlackChannelID, message)
            if err != nil {
                reqLogger.Error(err, "Failed to send slack message")
            }

			// Requeue
            retry = true
			return nil, retry
		}
	} else {
		reqLogger.Info("DaemonSet is not ready", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", name)
		retry = true
		return nil, retry
	}
    udpAppend := true
	if len(instance.Status.Tests) != 0 {
		for k, v := range instance.Status.Tests {
			if v.Name == "udp" {
                udpAppend = false
				if !reflect.DeepEqual(udpTestStatus, instance.Status.Tests[k]) {
					instance.Status.Tests[k] = udpTestStatus
					err := r.client.Status().Update(context.TODO(), instance)
					if err != nil {
						reqLogger.Error(err, "Failed to update CoastieService status")
						return err, retry
					}
				}
			}
		}
	}
    if udpAppend {
		instance.Status.Tests = append(instance.Status.Tests, udpTestStatus)
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update CoastieService status")
			return err, retry
		}
    }

	reqLogger.Info("Reached end of UDPTest", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", name)
	return nil, retry
}

func udpServer(cr *k8sv1alpha1.CoastieService, name string) *appsv1.DaemonSet {
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
							Image: "hub.soh.re/jmainguy/udpserver",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8082,
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

func udpServerService(cr *k8sv1alpha1.CoastieService, name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "udpserver",
					Protocol: "UDP",
					Port:     8082,
				},
			},
			Selector: map[string]string{
				"app": name,
			},
			//Type: "NodePort",
		},
	}
}

func udpClient(ip, port string) (status string) {
	// Node + port
	uri := fmt.Sprintf("%s:%s", ip, port)
	// Connect
	c, err := net.Dial("udp", uri)
	if err != nil {
		status = "ERROR: UDP unable to connect"
		return
	}
	// Send message
    question := fmt.Sprintln("ruok?")
	c.Write([]byte(question))
	// Read response
	message, _ := bufio.NewReader(c).ReadString('\n')
    if message == "imok\n" {
		c.Close()
		status = "SUCCESS: UDP is working"
		return
	} else if message != "" {
		c.Close()
		status = fmt.Sprintf("ERROR: UDP Failed - Server: %s", message)
		return
	}
	status = "ERROR: Should never reach this"
	return
}

func deleteUdpTest(instance *k8sv1alpha1.CoastieService, r *ReconcileCoastieService, reqLogger logr.Logger) (err error) {
	err = nil
	name := fmt.Sprintf("%s-udp", instance.Name)
	// Delete DaemonSet
	udpDaemonSet := udpServer(instance, name)
	err = r.client.Delete(context.TODO(), udpDaemonSet)
	if err != nil {
		return err
	}
	// Delete Service
	udpService := udpServerService(instance, name)
	err = r.client.Delete(context.TODO(), udpService)
	if err != nil {
		return err
	}
	return
}
