package main

import (
    "testing"
)

func TestBasicFunctionality(t *testing.T) {
    t.Log("Basic Go test infrastructure working")
    // Add actual tests here
}

func TestAPIVersion(t *testing.T) {
    expected := "3.0.0"
    if apiVersion != expected {
        t.Errorf("Expected API version %s, got %s", expected, apiVersion)
    }
}

func TestServiceName(t *testing.T) {
    expected := "visited-tracker"
    if serviceName != expected {
        t.Errorf("Expected service name %s, got %s", expected, serviceName)
    }
}
