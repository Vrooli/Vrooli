package config

import "os"

type Config struct {
	Name, DisplayName, Category, Description, DataDir string
	DefaultVUs, DefaultIterations                     int
	DefaultDuration                                   string
}

func Defaults() Config {
	return Config{Name: "k6", DisplayName: "K6 Load Testing", Category: "execution", Description: "Modern load testing tool with JavaScript scripting", DataDir: os.ExpandEnv("$HOME/.k6"), DefaultVUs: 10, DefaultIterations: 100, DefaultDuration: "30s"}
}
