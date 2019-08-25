package conf

import (
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"sync"
)

type Config []Node

type Node struct {
	Id     int
	Name   string
	Data   SshConfig
	Detail string
}

type SshConfig struct {
	IP         string
	Port       int
	Username   string
	Password   string
	PrivateKey string
}

func Read(Filepath string) (c Config) {
	var once sync.Once
	c = make(Config,5)
	once.Do(func() {
		file, _ := filepath.Abs(Filepath)
		f, err := os.Open(file)
		defer f.Close()
		if err != nil {
			panic(err)
		}
		decode := yaml.NewDecoder(f)
		if err := decode.Decode(&c); err != nil {
			panic(err)
		}
	})
	return c
}

func Write(c *Config, Filepath string) error {
	file, _ := filepath.Abs(Filepath)
	f, err := os.Create(file)
	defer f.Close()
	if err != nil {
		return err
	}
	encode := yaml.NewEncoder(f)
	if err := encode.Encode(&c); err != nil {
		return err
	}
	return nil
}
