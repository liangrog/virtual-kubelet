// +build aws_provider

package register

import (
	"github.com/virtual-kubelet/virtual-kubelet/providers"
	"github.com/virtual-kubelet/virtual-kubelet/providers/poc"
)

func init() {
	register("poc", initPoc)
}

func initPoc(cfg InitConfig) (providers.Provider, error) {
	return poc.NewPocProvider(cfg.ConfigPath, cfg.ResourceManager, cfg.NodeName, cfg.OperatingSystem, cfg.InternalIP, cfg.DaemonPort)
}
