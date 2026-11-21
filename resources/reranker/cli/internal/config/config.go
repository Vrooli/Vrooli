package config

import "os"

type Config struct {
	Endpoint, Model             string
	Port, StartupTimeoutSeconds int
	GPUEnabled                  bool
}

func Defaults() Config {
	return Config{Endpoint: "http://127.0.0.1:11453", Model: os.Getenv("RERANKER_MODEL"), Port: 11453, StartupTimeoutSeconds: 120}
}
