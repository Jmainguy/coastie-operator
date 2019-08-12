package coastieservice

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"
    "net/http"

    instr "k8s.io/apimachinery/pkg/util/intstr"
	"github.com/go-logr/logr"
	k8sv1alpha1 "github.com/jmainguy/coastie-operator/pkg/apis/k8s/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func runHttpTest(instance *k8sv1alpha1.CoastieService, r *ReconcileCoastieService, reqLogger logr.Logger) (err error, retry bool) {
	retry = false
	name := fmt.Sprintf("%s-http", instance.Name)
	// Define a new DaemonSet object
	httpDaemonSet := httpServer(instance, name)
	// Set CoastieService instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, httpDaemonSet, r.scheme); err != nil {
		return err, retry
	}

	// Check if this DaemonSet already exists
	found := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: name}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", httpDaemonSet.Namespace, "DaemonSet.Name", name)
		err = r.client.Create(context.TODO(), httpDaemonSet)
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
	httpTestStatus := k8sv1alpha1.Test{Name: "http", Status: "Fail"}
	if found.Status.DesiredNumberScheduled == found.Status.NumberReady {
		// All pods are now running, run test against them
		// Spin up service
		httpIngress := httpServerIngress(instance, name)
		// Set CoastieService instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, httpIngress, r.scheme); err != nil {
			return err, retry
		}
		// Check if Ingress exists
		err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: name}, httpIngress)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Creating a new Ingress", "Ingress.Namespace", httpIngress.Namespace, "Ingress.Name", name)
			err = r.client.Create(context.TODO(), httpIngress)
			if err != nil {
				return err, retry
			}
			// Ingress created successfully - return and requeue
			retry = true
			return nil, retry
		}
		// Ingress Exists, how do we connect to it?
		// Use client to connect to service, try 5 times if fail
		// If this is still true later, fail with message
		httpFail := true
		httpStatus := ""
		for i := 0; i < 5; i++ {
			httpStatus = httpClient(instance.Spec.HostURL)
			if strings.Contains(httpStatus, "SUCCESS") {
				httpTestStatus = k8sv1alpha1.Test{Name: "http", Status: "Pass"}
				httpFail = false
				// Exit loop
				i = 5
			} else {
				// Pods are running, but failing test, give them a few seconds
				time.Sleep(2 * time.Second)
			}
		}
		if httpFail {
			httpTestStatus = k8sv1alpha1.Test{Name: "http", Status: "Fail"}
			message := fmt.Sprintf("Coastie Operator: HTTP Test failed. %s", httpStatus)
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
        podList := &corev1.PodList{}
        r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace}, podList)
        nodes := getPodNode(name, podList.Items)
        fmt.Println(podList.Items)
        fmt.Println(nodes)
        i := 0
        for i < 5 {
            // Wait 60 seconds
            time.Sleep(60 * time.Second)
            found := &appsv1.DaemonSet{}
            err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: name}, found)
            if err != nil {
			    message := fmt.Sprintf("Coastie Operator: Failed to get DaemonSet status")
    			// Alarm slack if failed
    			err := notifySlack(instance.Spec.SlackToken, instance.Spec.SlackChannelID, message)
                if err != nil {
                    reqLogger.Error(err, "Failed to send slack message")
                    return err, retry
                }
            }
            if found.Status.DesiredNumberScheduled == found.Status.NumberReady {
                break
            }
            // Else
    		reqLogger.Info("DaemonSet is not ready", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", name)
            i++
        }
        if i == 5 {
            // If here, means Daemonset to not become ready withing 5 minutes

	        message := fmt.Sprintf("Coastie Operator: DaemonSet took longer than 5 minutes to become ready")
    		// Alarm slack if failed
    		err := notifySlack(instance.Spec.SlackToken, instance.Spec.SlackChannelID, message)
            if err != nil {
                reqLogger.Error(err, "Failed to send slack message")
                return err, retry
            }
            retry = true
			return nil, retry
        }
	}
	if len(instance.Status.Tests) == 0 {
		instance.Status.Tests = append(instance.Status.Tests, httpTestStatus)
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update CoastieService status")
			return err, retry
		}
	} else {
		for k, v := range instance.Status.Tests {
			if v.Name == "http" {
				if !reflect.DeepEqual(httpTestStatus, instance.Status.Tests[k]) {
					instance.Status.Tests[k] = httpTestStatus
					err := r.client.Status().Update(context.TODO(), instance)
					if err != nil {
						reqLogger.Error(err, "Failed to update CoastieService status")
						return err, retry
					}
				}
			}
		}
	}

	reqLogger.Info("Reached end of HTTPTest", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", name)
	return nil, retry
}

func httpServer(cr *k8sv1alpha1.CoastieService, name string) *appsv1.DaemonSet {
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
							Image: "hub.soh.re/jmainguy/httpserver",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
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
}

func httpServerIngress(cr *k8sv1alpha1.CoastieService, name string) *extensionsv1beta1.Ingress {
    port := instr.FromInt(80)
	return &extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
		Spec: extensionsv1beta1.IngressSpec{
            Backend: &extensionsv1beta1.IngressBackend{
                ServiceName: "httpserver-service",
                ServicePort: port,
            },
            Rules: []extensionsv1beta1.IngressRule{
                {
                    Host: cr.Spec.HostURL,
                },
            },
		},
	}
}

func httpClient(hostURL string) (status string) {
    url := fmt.Sprintf("http://%s/ruok", hostURL)
    resp, err := http.Get(url)
    if err != nil {
		status = fmt.Sprintf("ERROR: HTTP Failed - Server: %s", err)
		return
    }
    defer resp.Body.Close()
    if resp.StatusCode == 200 {
		status = "SUCCESS: HTTP is working"
		return
    } else {
		status = fmt.Sprintf("ERROR: HTTP Failed - StatusCode Returned was : %s", resp.StatusCode)
		return
    }
    status = "ERROR: Should never reach this"
    return
}

func deleteHttpTest(instance *k8sv1alpha1.CoastieService, r *ReconcileCoastieService, reqLogger logr.Logger) (err error) {
	err = nil
	name := fmt.Sprintf("%s-http", instance.Name)
	// Delete DaemonSet
	httpDaemonSet := httpServer(instance, name)
	err = r.client.Delete(context.TODO(), httpDaemonSet)
	if err != nil {
		return err
	}
	// Delete Ingress
	httpIngress := httpServerIngress(instance, name)
	err = r.client.Delete(context.TODO(), httpIngress)
	if err != nil {
		return err
	}
	return
}

func getPodNode(name string, pods []corev1.Pod) []string {
	var nodes []string
	for _, pod := range pods {
        for _, v := range pod.ObjectMeta.OwnerReferences {
            fmt.Println(v)
            if v.Name == name {
                nodes = append(nodes, pod.Status.HostIP)
            }
        }
	}
    return nodes
}
