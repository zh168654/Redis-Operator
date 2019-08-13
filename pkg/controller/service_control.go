package controller

import (
	"fmt"

	kapiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	rapi "github.com/zh168654/Redis-Operator/pkg/api/redis/v1"
	"github.com/zh168654/Redis-Operator/pkg/controller/pod"
	"strconv"
	"strings"
)

// ServicesControlInterface inferface for the ServicesControl
type ServicesControlInterface interface {
	// CreateRedisClusterService used to create the Kubernetes Services needed to access the Redis Cluster
	CreateRedisClusterService(redisCluster *rapi.RedisCluster) ([]*kapiv1.Service, error)
	// DeleteRedisClusterService used to delete the Kubernetes Services linked to the Redis Cluster
	DeleteRedisClusterService(redisCluster *rapi.RedisCluster) error
	// GetRedisClusterService used to retrieve the Kubernetes Services associated to the RedisCluster
	GetRedisClusterService(redisCluster *rapi.RedisCluster) ([]*kapiv1.Service, error)

	// AddRedisPodService used to add a nodePort service for the redis pod
	AddRedisPodService(redisCluster *rapi.RedisCluster, currentPods int32) (*kapiv1.Service, error)
	// RemoveRedisPodService used to remove a nodePort service for the redis pod
	RemoveRedisPodService(redisCluster *rapi.RedisCluster, currentPods int32) error
}

// ServicesControl contains all information for managing Kube Services
type ServicesControl struct {
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

// NewServicesControl builds and returns new ServicesControl instance
func NewServicesControl(client clientset.Interface, rec record.EventRecorder) *ServicesControl {
	ctrl := &ServicesControl{
		KubeClient: client,
		Recorder:   rec,
	}

	return ctrl
}

// GetRedisClusterService used to retrieve the Kubernetes Service associated to the RedisCluster
func (s *ServicesControl) GetRedisClusterService(redisCluster *rapi.RedisCluster) ([]*kapiv1.Service, error) {
	serviceName := getServiceName(redisCluster)
	svc, err := s.KubeClient.CoreV1().Services(redisCluster.Namespace).Get(serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if svc.Name != serviceName {
		return nil, fmt.Errorf("Couldn't find service named %s", serviceName)
	}
	var svcList []*kapiv1.Service
	svcList = append(svcList, svc)

	numberOfMaster := getNumberOfMaster(redisCluster)
	replicationFactor := getReplicationFactor(redisCluster)
	if strings.EqualFold(getServiceType(redisCluster), string(rapi.ServiceTypeExternal)) {
		for i := 0; i < int(numberOfMaster*replicationFactor); i++ {
			podServiceName := serviceName + "-external-" + strconv.Itoa(i)
			pod_svc, err := s.KubeClient.CoreV1().Services(redisCluster.Namespace).Get(podServiceName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			if pod_svc.Name != podServiceName {
				return nil, fmt.Errorf("Couldn't find service named %s", podServiceName)
			}
			svcList = append(svcList, pod_svc)
		}
	}
	return svcList, nil
}

// CreateRedisClusterService used to create the Kubernetes Service needed to access the Redis Cluster
func (s *ServicesControl) CreateRedisClusterService(redisCluster *rapi.RedisCluster) ([]*kapiv1.Service, error) {
	defer func() {
		if err := recover(); err != nil {
			if createdServiceList, ok := err.([]*kapiv1.Service); ok {
				for _, createdService := range createdServiceList {
					s.KubeClient.CoreV1().Services(redisCluster.Namespace).Delete(createdService.Name, nil)
				}
			}
		}
	}()
	serviceName := getServiceName(redisCluster)
	desiredlabels, err := pod.GetLabelsSet(redisCluster)
	if err != nil {
		return nil, err
	}

	desiredAnnotations, err := pod.GetAnnotationsSet(redisCluster)
	if err != nil {
		return nil, err
	}

	var serviceList []*kapiv1.Service
	newService := &kapiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          desiredlabels,
			Annotations:     desiredAnnotations,
			Name:            serviceName,
			OwnerReferences: []metav1.OwnerReference{pod.BuildOwnerReference(redisCluster)},
		},
		Spec: kapiv1.ServiceSpec{
			ClusterIP: "None",
			Ports:     []kapiv1.ServicePort{{Port: 6379, Name: "redis"}},
			Selector:  desiredlabels,
		},
	}

	numberOfMaster := getNumberOfMaster(redisCluster)
	replicationFactor := getReplicationFactor(redisCluster)
	serviceNodePortStart := getServiceNodePortStart(redisCluster)

	// Create nodePort services for each pod
	if serviceNodePortStart != "" && strings.EqualFold(getServiceType(redisCluster), string(rapi.ServiceTypeExternal)) {
		for i := 0; i < int(numberOfMaster*replicationFactor); i++ {
			if nodePortStart, err := strconv.ParseInt(serviceNodePortStart, 10, 32); err == nil {
				desiredPodlabels, err := pod.GetPodLabelsSet(redisCluster, int32(i))
				if err != nil {
					return nil, err
				}
				newPodService := &kapiv1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Labels:          desiredPodlabels,
						Annotations:     desiredAnnotations,
						Name:            serviceName + "-external-" + strconv.Itoa(i),
						OwnerReferences: []metav1.OwnerReference{pod.BuildOwnerReference(redisCluster)},
					},
					Spec: kapiv1.ServiceSpec{
						Ports:    []kapiv1.ServicePort{{Port: 6379, Name: "redis", NodePort: int32(nodePortStart)+int32(i)}},
						Selector: desiredPodlabels,
					},
				}

				if _, err := s.KubeClient.CoreV1().Services(redisCluster.Namespace).Create(newService); err != nil {
					panic(serviceList)
					return serviceList, err
				}
				serviceList = append(serviceList, newPodService)
			}
		}
	}
	if _, err := s.KubeClient.CoreV1().Services(redisCluster.Namespace).Create(newService); err != nil {
		panic(serviceList)
		return serviceList, err
	}
	serviceList = append(serviceList, newService)

	return serviceList, err
}

// DeleteRedisClusterService used to delete the Kubernetes Service linked to the Redis Cluster
func (s *ServicesControl) DeleteRedisClusterService(redisCluster *rapi.RedisCluster) error {
	serviceName := getServiceName(redisCluster)
	numberOfMaster := getNumberOfMaster(redisCluster)
	replicationFactor := getReplicationFactor(redisCluster)

	if err := s.KubeClient.CoreV1().Services(redisCluster.Namespace).Delete(serviceName, nil); err != nil {
		return err
	}

	if strings.EqualFold(getServiceType(redisCluster), string(rapi.ServiceTypeExternal)) {
		for i := 0; i < int(numberOfMaster*replicationFactor); i++ {
			podServiceName := serviceName + "-external-" + strconv.Itoa(i)
			err := s.KubeClient.CoreV1().Services(redisCluster.Namespace).Delete(podServiceName, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// AddRedisPodService used to add a nodePort service for the redis pod
func (s *ServicesControl) AddRedisPodService(redisCluster *rapi.RedisCluster, currentPods int32) (*kapiv1.Service, error) {
	serviceNodePortStart := getServiceNodePortStart(redisCluster)
	serviceName := getServiceName(redisCluster)
	if serviceNodePortStart != "" && strings.EqualFold(getServiceType(redisCluster), string(rapi.ServiceTypeExternal)) {
		desiredPodlabels, err := pod.GetPodLabelsSet(redisCluster, currentPods)
		if err != nil {
			return nil, err
		}
		desiredAnnotations, err := pod.GetAnnotationsSet(redisCluster)
		if err != nil {
			return nil, err
		}
		if nodePortStart, err := strconv.ParseInt(serviceNodePortStart, 10, 32); err == nil {
			newPodService := &kapiv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Labels:          desiredPodlabels,
					Annotations:     desiredAnnotations,
					Name:            serviceName + "-external-" + string(currentPods),
					OwnerReferences: []metav1.OwnerReference{pod.BuildOwnerReference(redisCluster)},
				},
				Spec: kapiv1.ServiceSpec{
					Ports:    []kapiv1.ServicePort{{Port: 6379, Name: "redis", NodePort: int32(nodePortStart)+currentPods}},
					Selector: desiredPodlabels,
				},
			}
			if svc, err := s.KubeClient.CoreV1().Services(redisCluster.Namespace).Create(newPodService); err != nil {
				return svc, err
			}
		}
	}
	return nil, nil
}

// RemoveRedisPodService used to remove a nodePort service for the redis pod
func (s *ServicesControl) RemoveRedisPodService(redisCluster *rapi.RedisCluster, currentPods int32) error {
	if strings.EqualFold(getServiceType(redisCluster), string(rapi.ServiceTypeExternal)) {
		serviceName := getServiceName(redisCluster)
		podServiceName := serviceName + "-external-" + string(currentPods)
		err := s.KubeClient.CoreV1().Services(redisCluster.Namespace).Delete(podServiceName, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func getServiceName(redisCluster *rapi.RedisCluster) string {
	serviceName := redisCluster.Name
	if redisCluster.Spec.ServiceName != "" {
		serviceName = redisCluster.Spec.ServiceName
	}
	return serviceName
}

func getServiceType(redisCluster *rapi.RedisCluster) string {
	serviceType := redisCluster.Name
	if redisCluster.Spec.ServiceType != "" {
		serviceType = redisCluster.Spec.ServiceType
	}
	return serviceType
}

func getNumberOfMaster(redisCluster *rapi.RedisCluster) int32 {
	return *redisCluster.Spec.NumberOfMaster
}

func getReplicationFactor(redisCluster *rapi.RedisCluster) int32 {
	return *redisCluster.Spec.ReplicationFactor
}

func getServiceNodePortStart(redisCluster *rapi.RedisCluster) string {
	serviceNodePortStart := ""
	if redisCluster.Spec.ServiceNodePortStart != "" {
		serviceNodePortStart = redisCluster.Spec.ServiceNodePortStart
	}
	return serviceNodePortStart
}
