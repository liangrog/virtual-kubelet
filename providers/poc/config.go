package poc

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	localKubeConfig  string `yaml:"LocalKubeConfig"`
	remoteKubeConfig string `yaml:"RemoteKubeConfig"`
}

func NewConfig(cfg string) (*Config, error) {
	data, err := ioutil.ReadFile(cfg)
	if err != nil {
		return nil, err
	}

	c := new(Config)
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}

	return c, nil
}
