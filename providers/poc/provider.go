package poc

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"github.com/virtual-kubelet/virtual-kubelet/manager"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	REMOTE_POD_ANNOTATION_NAME  = "virtual-kube-type"
	REMOTE_POD_ANNOTATION_VALUE = "poc"
)

// Remote client
var rc kubernetes.Interface

// Local client
var lc kubernetes.Interface

// FargateProvider implements the virtual-kubelet provider interface.
type PocProvider struct {
	resourceManager    *manager.ResourceManager
	nodeName           string
	operatingSystem    string
	internalIP         string
	daemonEndpointPort int32
}

var (
	errNotImplemented = fmt.Errorf("not implemented by Poc provider")
)

func TranslateToRemotePod(pod *corev1.Pod) (*corev1.Pod, error) {
	var rpod *corev1.Pod
	rpod, _ = rc.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})

	// If remote pod doesn't exist
	if len(rpod.Name) == 0 {
		annotations := make(map[string]string)
		annotations[REMOTE_POD_ANNOTATION_NAME] = REMOTE_POD_ANNOTATION_VALUE

		containers := make([]corev1.Container, 0, len(pod.Spec.Containers))
		for _, c := range pod.Spec.Containers {
			cntr := corev1.Container{
				Name:       c.Name,
				Image:      c.Image,
				Command:    c.Command,
				Args:       c.Args,
				Resources:  c.Resources,
				Ports:      c.Ports,
				Env:        c.Env,
				WorkingDir: c.WorkingDir,
			}

			containers = append(containers, cntr)
		}

		rpod = &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   pod.Namespace,
				Name:        pod.Name,
				Annotations: annotations,
			},
			Spec: corev1.PodSpec{
				Volumes:       []corev1.Volume{},
				Containers:    containers,
				RestartPolicy: pod.Spec.RestartPolicy,
			},
		}
	}

	return rpod, nil
}

func UpdateToLocalPod(pod *corev1.Pod) (*corev1.Pod, error) {
	rpod, err := rc.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
	if err != nil {
		return pod, err
	}

	pod.Status = rpod.Status
	pod.Spec = rpod.Spec

	return pod, err
}

// NewFargateProvider creates a new Fargate provider.
func NewPocProvider(
	config string,
	rm *manager.ResourceManager,
	nodeName string,
	operatingSystem string,
	internalIP string,
	daemonEndpointPort int32) (*PocProvider, error) {

	// Create the Fargate provider.
	log.Println("Creating Poc provider.")

	p := PocProvider{
		resourceManager:    rm,
		nodeName:           nodeName,
		operatingSystem:    operatingSystem,
		internalIP:         internalIP,
		daemonEndpointPort: daemonEndpointPort,
	}

	// Load config
	c, err := NewConfig(config)
	if err != nil {
		return nil, err
	}

	// Load client
	rc = GetClient("out", c.remoteKubeConfig)
	lc = GetClient("out", c.localKubeConfig)

	lp, _ := lc.CoreV1().Pods("kube-system").Get("storage-provisioner", metav1.GetOptions{})
	log.Printf("local ======  %+v.\n", lp)

	rp, _ := rc.CoreV1().Pods("default").Get("busybox", metav1.GetOptions{})
	log.Printf("remote ======  %+v.\n", rp)

	log.Printf("Created Poc provider: %+v", p)

	return &p, nil
}

// CreatePod takes a Kubernetes Pod and deploys it within the Fargate provider.
func (p *PocProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	log.Printf("Received CreatePod request for %+v.\n", pod)
	pd, err := TranslateToRemotePod(pod)
	if err != nil {
		return err
	}

	log.Printf("------------++++ remote pod %+v.\n", pd)
	rd, err := rc.CoreV1().Pods(pod.Namespace).Create(pd)

	log.Printf("created remote pod %+v.\n", rd)
	return err
}

// UpdatePod takes a Kubernetes Pod and updates it within the provider.
func (p *PocProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	log.Printf("Received UpdatePod request for %+v.\n", pod)

	pd, err := TranslateToRemotePod(pod)
	if err != nil {
		return err
	}
	_, err = rc.CoreV1().Pods(pod.Namespace).Update(pd)
	return nil
}

// DeletePod takes a Kubernetes Pod and deletes it from the provider.
func (p *PocProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	log.Printf("Received DeletePod request for %s/%s.\n", pod.Namespace, pod.Name)

	return rc.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
}

// GetPod retrieves a pod by name from the provider (can be cached).
func (p *PocProvider) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	log.Printf("Received GetPod request for %s/%s.\n", namespace, name)
	pod, err := rc.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})

	log.Printf("Got remote pod ====  %+v.\n", pod)
	log.Printf("%s", err)

	if err != nil {
		return nil, errdefs.NotFoundf("pod %s/%s is not found", namespace, name)
	}

	//	return UpdateToLocalPod(pod)
	return pod, nil
}

// GetContainerLogs retrieves the logs of a container by name from the provider.
func (p *PocProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	return ioutil.NopCloser(strings.NewReader("not support in POC Provider")), nil
}

// RunInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *PocProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error {
	return errNotImplemented
}

// GetPodStatus retrieves the status of a pod by name from the provider.
func (p *PocProvider) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	log.Printf("Received GetPodStatus request for %s/%s.\n", namespace, name)

	pod, err := rc.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return &corev1.PodStatus{Phase: corev1.PodUnknown}, nil
	}

	status := pod.Status

	log.Printf("Responding to GetPodStatus: %+v.\n", status)

	return &status, nil
}

// GetPods retrieves a list of all pods running on the provider (can be cached).
func (p *PocProvider) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	log.Println("Received GetPods request.")

	list, err := rc.CoreV1().Pods("").List(metav1.ListOptions{FieldSelector: fmt.Sprintf("%s=%s", REMOTE_POD_ANNOTATION_NAME, REMOTE_POD_ANNOTATION_VALUE)})

	if err != nil {
		log.Printf("Failed to get pods: %v.\n", err)
		return nil, err
	}

	var result []*corev1.Pod

	for _, pod := range list.Items {
		/*
			pd, err := UpdateToLocalPod(&pod)
			if err != nil {
				return result, err
			}*/
		result = append(result, &pod)
	}

	log.Printf("Responding to GetPods: %+v.\n", result)

	return result, nil
}

// Capacity returns a resource list with the capacity constraints of the provider.
func (p *PocProvider) Capacity(ctx context.Context) corev1.ResourceList {
	log.Println("Received Capacity request.")

	return corev1.ResourceList{
		"cpu":    resource.MustParse("100"),
		"memory": resource.MustParse("50Gi"),
		"pods":   resource.MustParse("100"),
	}
}

// NodeConditions returns a list of conditions (Ready, OutOfDisk, etc), which is polled
// periodically to update the node status within Kubernetes.
func (p *PocProvider) NodeConditions(ctx context.Context) []corev1.NodeCondition {
	log.Println("Received NodeConditions request.")

	lastHeartbeatTime := metav1.Now()
	lastTransitionTime := metav1.Now()
	lastTransitionReason := "Poc cluster is ready"
	lastTransitionMessage := "ok"

	// Return static thumbs-up values for all conditions.
	return []corev1.NodeCondition{
		{
			Type:               corev1.NodeReady,
			Status:             corev1.ConditionTrue,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
		{
			Type:               corev1.NodeOutOfDisk,
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
		{
			Type:               corev1.NodeMemoryPressure,
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
		{
			Type:               corev1.NodeDiskPressure,
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
		{
			Type:               corev1.NodeNetworkUnavailable,
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
		{
			Type:               "KubeletConfigOk",
			Status:             corev1.ConditionTrue,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
	}
}

// NodeAddresses returns a list of addresses for the node status within Kubernetes.
func (p *PocProvider) NodeAddresses(ctx context.Context) []corev1.NodeAddress {
	log.Println("Received NodeAddresses request.")

	return []corev1.NodeAddress{
		{
			Type:    corev1.NodeInternalIP,
			Address: p.internalIP,
		},
	}
}

// NodeDaemonEndpoints returns NodeDaemonEndpoints for the node status within Kubernetes.
func (p *PocProvider) NodeDaemonEndpoints(ctx context.Context) *corev1.NodeDaemonEndpoints {
	log.Println("Received NodeDaemonEndpoints request.")

	return &corev1.NodeDaemonEndpoints{
		KubeletEndpoint: corev1.DaemonEndpoint{
			Port: p.daemonEndpointPort,
		},
	}
}

// OperatingSystem returns the operating system the provider is for.
func (p *PocProvider) OperatingSystem() string {
	log.Println("Received OperatingSystem request.")

	return p.operatingSystem
}
