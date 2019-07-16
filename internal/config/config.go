package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type Config struct {
	ProjectID string `yaml:"project_id"`

	GitLab struct {
		URL   string `yaml:"url"`
		Token string `yaml:"token"`
	} `yaml:"gitlab"`

	Files []string `yaml:"files"`
}

func Load(path string) *Config {
	// TODO читать из перменных CI
	var c Config

	d, err := ioutil.ReadFile(path)

	if err != nil {
		log.Fatal(err, "Failed to read config file")
	}

	err = yaml.Unmarshal(d, &c)

	if err != nil {
		log.Fatal(err, "Failed to parse config file")
	}

	return &c
}
