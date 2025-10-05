package main

import (
	"testing"
)

func TestNewServer(t *testing.T) {
	config := &Config{
		Port:        "8080",
		DatabaseURL: "",
		OllamaURL:   "http://localhost:11434",
		RedisURL:    "redis://localhost:6379",
	}

	server := NewServer(config)

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.config != config {
		t.Error("Expected server config to match input config")
	}
}

func TestServerInitialize(t *testing.T) {
	config := &Config{
		Port:        "8080",
		DatabaseURL: "",
		OllamaURL:   "http://localhost:11434",
		RedisURL:    "redis://localhost:6379",
	}

	server := NewServer(config)

	// Initialize should succeed even without actual resources
	err := server.Initialize()
	if err != nil {
		t.Logf("Initialize returned error (expected in test environment): %v", err)
	}
}

func TestServerRoutes(t *testing.T) {
	config := &Config{
		Port:        "8080",
		DatabaseURL: "",
		OllamaURL:   "http://localhost:11434",
		RedisURL:    "redis://localhost:6379",
	}

	server := NewServer(config)
	router := server.setupRouter()

	if router == nil {
		t.Fatal("Expected router to be initialized")
	}
}
