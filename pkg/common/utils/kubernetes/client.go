package kubernetes

import (
	"agones.dev/agones/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"matchmaker/pkg/common/utils"
)

var (
	kubeConfig = createKubernetesConfig()
	KubeClient = kubernetes.NewForConfigOrDie(kubeConfig)

	// AgonesClient contains the Agones client for creating GameServerAllocation objects
	AgonesClient = versioned.NewForConfigOrDie(kubeConfig)
)

func createKubernetesConfig() *rest.Config {
	kConfig, err := utils.CreateKubernetesConfig()
	if err != nil {
		panic(err)
	}
	return kConfig
}
