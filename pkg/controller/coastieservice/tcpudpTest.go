package coastieservice

import (
	"bufio"
	"context"
	"fmt"
	"net"
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

func runTcpUdpTest(instance *k8sv1alpha1.CoastieService, r *ReconcileCoastieService, reqLogger logr.Logger, tcpudp string) (err error, retry bool) {
	reqLogger.Info("Begining Test", "TestName", strings.ToUpper(tcpudp))
	retry = false
	name := fmt.Sprintf("%s-%s", instance.Name, tcpudp)
	// Define a new DaemonSet object
	DaemonSet, containerPort := tcpudpServer(instance, name, tcpudp)
	// Set CoastieService instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, DaemonSet, r.scheme); err != nil {
		return err, retry
	}

	// Check if this DaemonSet already exists
	TestStatus := instance.Status.TestResults[tcpudp]
	found := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: name}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", DaemonSet.Namespace, "DaemonSet.Name", name)
		err = r.client.Create(context.TODO(), DaemonSet)
		if err != nil {
			return err, retry
		}
		// DaemonSet created successfully - return and requeue
		now := time.Now()
		dsct := now.Format(time.RFC3339)

		TestStatus.DaemonSetCreationTime = dsct
		TestStatus.Status = "Running"
		err = updateCoastieStatus(instance, TestStatus, tcpudp, reqLogger, r)
		if err != nil {
			return err, retry
		}

		retry = true
		return nil, retry
	} else if err != nil {
		return err, retry
	}

	// Else if No errors, and DS already exists, check its status
	if found.Status.DesiredNumberScheduled == found.Status.NumberReady {
		// All pods are now running, run test against them
		// Spin up service
		tcpudpService := tcpudpServerService(instance, name, tcpudp)
		// Set CoastieService instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, tcpudpService, r.scheme); err != nil {
			return err, retry
		}
		// Check if Service exists
		err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: name}, tcpudpService)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating a new Service", "Service.Namespace", tcpudpService.Namespace, "Service.Name", name)
			err = r.client.Create(context.TODO(), tcpudpService)
			if err != nil {
				return err, retry
			}
			// Service created successfully - return and requeue
			retry = true
			return nil, retry
		}
		// Service Exists, how do we connect to it?
		ServerClusterIP := tcpudpService.Spec.ClusterIP
		// Use client to connect to service, try 5 times if fail
		// If this is still true later, fail with message
		Fail := true
		Status := ""
		for i := 0; i < 5; i++ {
			Status = tcpudpClient(ServerClusterIP, tcpudp, containerPort)
			if strings.Contains(Status, "SUCCESS") {
				TestStatus.Status = "Passed"
				err = updateCoastieStatus(instance, TestStatus, tcpudp, reqLogger, r)
				if err != nil {
					return err, retry
				}
				Fail = false
				// Exit loop
				i = 5
			} else {
				// Pods are running, but failing test, give them a few seconds
				time.Sleep(2 * time.Second)
			}
		}
		if Fail {
			TestStatus.Status = "Failed"
			err = updateCoastieStatus(instance, TestStatus, tcpudp, reqLogger, r)
			if err != nil {
				return err, retry
			}
			message := fmt.Sprintf("Coastie Operator: %s Test failed. %s", strings.ToUpper(tcpudp), Status)
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

	reqLogger.Info("Reached end of Test", "TestName", strings.ToUpper(tcpudp))
	return nil, retry
}

func tcpudpServer(cr *k8sv1alpha1.CoastieService, name, tcpudp string) (ds *appsv1.DaemonSet, containerPort int32) {
	var image string
	if tcpudp == "udp" {
		containerPort = 8082
		image = "hub.soh.re/jmainguy/udpserver"
	} else {
		containerPort = 8081
		image = "hub.soh.re/jmainguy/tcpserver"
	}
	ds = &appsv1.DaemonSet{
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
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: containerPort,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("0.1"),
									"memory": resource.MustParse("100M"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("0.1"),
									"memory": resource.MustParse("100M"),
								},
							},
						},
					},
				},
			},
		},
	}
	return ds, containerPort
}

func tcpudpServerService(cr *k8sv1alpha1.CoastieService, name, tcpudp string) *corev1.Service {
	var containerName string
	var containerPort int32
	var protocol corev1.Protocol
	if tcpudp == "udp" {
		containerPort = 8082
		containerName = "udpserver"
		protocol = "UDP"
	} else {
		containerPort = 8081
		containerName = "tcpserver"
		protocol = "TCP"
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     containerName,
					Protocol: protocol,
					Port:     containerPort,
				},
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}
}

func tcpudpClient(ip, tcpudp string, port int32) (status string) {
	var question string
	var expectedResponse string
	if tcpudp == "tcp" {
		question = fmt.Sprintln("Annie, are you ok?\n")
		expectedResponse = "So, Annie are you ok?\n"
	} else {
		question = fmt.Sprintln("ruok?")
		expectedResponse = "imok\n"
	}
	// Node + port
	uri := fmt.Sprintf("%s:%d", ip, port)
	// Connect
	c, err := net.Dial(tcpudp, uri)
	if err != nil {
		status = fmt.Sprintf("ERROR: %s unable to connect", strings.ToUpper(tcpudp))
		return
	}
	// Send message
	c.Write([]byte(question))
	// Read response
	message, _ := bufio.NewReader(c).ReadString('\n')
	if message == expectedResponse {
		c.Close()
		status = fmt.Sprintf("SUCCESS: %s is working", strings.ToUpper(tcpudp))
		return
	} else if message != "" {
		c.Close()
		status = fmt.Sprintf("ERROR: %s Failed - Server: %s", strings.ToUpper(tcpudp), message)
		return
	}
	status = "ERROR: Should never reach this"
	return
}

func deleteTcpUdpTest(instance *k8sv1alpha1.CoastieService, r *ReconcileCoastieService, reqLogger logr.Logger, tcpudp string) (err error) {
	err = nil
	name := fmt.Sprintf("%s-%s", instance.Name, tcpudp)
	// Delete DaemonSet
	DaemonSet, _ := tcpudpServer(instance, name, tcpudp)
	err = r.client.Delete(context.TODO(), DaemonSet)
	if err != nil {
		return err
	}
	// Delete Service
	tcpudpService := tcpudpServerService(instance, name, tcpudp)
	err = r.client.Delete(context.TODO(), tcpudpService)
	if err != nil {
		return err
	}
	return
}