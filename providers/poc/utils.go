// Utililities being used by core
package poc

import (
	"log"
	"os"
	"reflect"
	"strings"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Get kubernetes client based on config type
// Default in-cluster config
// Options: "out" for out-of-cluster config
func GetClient(configType, path string) kubernetes.Interface {
	var config *rest.Config

	switch configType {
	case "out":
		config = GetConfigOutOfCluster(path)
	case "in":
		fallthrough
	default:
		config = GetConfigInCluster()
	}

	return CreateClient(config)
}

// Create kubernetes client
func CreateClient(config *rest.Config) kubernetes.Interface {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Can not create kubernetes client: %v", err)
	}

	return clientset
}

// Get in-cluster config
func GetConfigInCluster() *rest.Config {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Can not get kubernetes in-cluster config: %v", err)
	}

	return config
}

// Get out-of-cluster config
// Will look for
// - ENV
// - $HOME/.kube/config
func GetConfigOutOfCluster(kubeconfigPath string) *rest.Config {
	if len(kubeconfigPath) == 0 {
		kubeconfigPath = os.Getenv("KUBE_CONFIG_PATH")
	}
	if kubeconfigPath == "" {
		log.Fatal("Missing 'KUBE_CONFIG_PATH' environment variable for config type 'out'")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatalf("Can not get kubernetes out-of-cluster config: %v", err)
	}

	return config
}

// Find element in slice
func InArray(niddle interface{}, haystack interface{}) (bool, int) {
	switch reflect.TypeOf(haystack).Kind() {
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		hs := reflect.ValueOf(haystack)
		for i := 0; i < hs.Len(); i++ {
			if reflect.DeepEqual(niddle, hs.Index(i).Interface()) == true {
				return true, i
			}
		}
	}

	return false, 0
}

// Get environment variable providing default if not exist
func GetEnvWithDefault(envName, def string) string {
	ev := os.Getenv(envName)
	if ev == "" {
		return def
	}

	return ev
}

// Turning comma delimited string to slic
func CommaStrToSlice(str string) []string {
	tmp := strings.Split(str, ",")
	// Remove spaces
	var result []string
	for _, s := range tmp {
		result = append(result, strings.TrimSpace(s))
	}

	return result
}

// Secret CRUD wrapper
func GetSecret(client kubernetes.Interface, namespace, secretName string, options metav1.GetOptions) (*v1.Secret, error) {
	return client.CoreV1().Secrets(namespace).Get(secretName, options)
}

// Secret CRUD wrapper
func CreateSecret(client kubernetes.Interface, namespace string, secret *v1.Secret) (*v1.Secret, error) {
	return client.CoreV1().Secrets(namespace).Create(secret)
}

// Secret CRUD wrapper
func UpdateSecret(client kubernetes.Interface, namespace string, secret *v1.Secret) (*v1.Secret, error) {
	return client.CoreV1().Secrets(namespace).Update(secret)
}

// Secret CRUD wrapper
func DeleteSecret(client kubernetes.Interface, namespace, secretName string, options *metav1.DeleteOptions) error {
	return client.CoreV1().Secrets(namespace).Delete(secretName, options)
}
