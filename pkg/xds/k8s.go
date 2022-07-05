package xds

import (
	"context"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PodEndPoint struct {
	Name      string
	Namespace string
	IP        string
	Port      int32
	PortName  string
}

func NewK8sClient() *kubernetes.Clientset {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	// creates the clientSet
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientSet
}

func ListenNamespaces(clientSet *kubernetes.Clientset, services []string, namespace string, endpointMap chan<- map[string][]PodEndPoint) {
	watch, err := clientSet.CoreV1().Endpoints(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		logrus.Fatalf("Error listening services %v", err)
		return
	}
	for event := range watch.ResultChan() {
		k8sEndPoints := make(map[string][]PodEndPoint)
		for _, serviceName := range services {
			endPoint, ok := event.Object.(*v1.Endpoints)
			if !ok {
				logrus.Fatalf("Object can not convert to v1.Endpoints %v, %#v", event.Type, event.Object)
			}
			logrus.Debugf("Watching services endpoint: %v", endPoint.String())
			name := endPoint.GetObjectMeta().GetName()
			if name == serviceName {
				var ips []string
				var ports []int32
				for _, subset := range endPoint.Subsets {
					for _, address := range subset.Addresses {
						ips = append(ips, address.IP)
					}
					for _, port := range subset.Ports {
						ports = append(ports, port.Port)
					}
				}
				logrus.Debug("Endpoint", zap.String("name", name), zap.Any("IP Address", ips), zap.Any("Ports", ports))
				var podEndPoints []PodEndPoint
				for _, port := range ports {
					for _, ip := range ips {
						podEndPoints = append(podEndPoints, PodEndPoint{Name: name, Namespace: namespace, IP: ip, Port: port})
					}
				}
				k8sEndPoints[serviceName] = podEndPoints
			}
		}
		if len(k8sEndPoints) > 0 {
			endpointMap <- k8sEndPoints
		}
	}
}

func ListEndpoints(clientSet *kubernetes.Clientset, services []string, namespace string) map[string][]PodEndPoint {
	endpoints, err := clientSet.CoreV1().Endpoints(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logrus.Fatalf("Error listening services %v", err)
		return nil
	}
	logrus.Debugf("List endpoints %v", endpoints.String())
	k8sEndPoints := make(map[string][]PodEndPoint)
	for _, serviceName := range services {
		for _, endpoint := range endpoints.Items {
			name := endpoint.GetObjectMeta().GetName()
			if name == serviceName {
				var ips, portNames []string
				var ports []int32
				for _, subset := range endpoint.Subsets {
					for _, address := range subset.Addresses {
						ips = append(ips, address.IP)
					}
					for _, port := range subset.Ports {
						ports = append(ports, port.Port)
						portNames = append(portNames, port.Name)
					}
				}
				logrus.Debug("Endpoint", zap.String("name", name), zap.Any("IP Address", ips), zap.Any("Ports", ports))
				var podEndPoints []PodEndPoint
				for k, port := range ports {
					for _, ip := range ips {
						podEndPoints = append(podEndPoints, PodEndPoint{Name: name, Namespace: namespace, IP: ip, Port: port, PortName: portNames[k]})
					}
				}
				k8sEndPoints[serviceName] = podEndPoints
			}
		}
	}
	return k8sEndPoints
}
