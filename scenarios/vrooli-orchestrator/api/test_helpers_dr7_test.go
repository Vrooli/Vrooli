package main

import (
	"os"
	"path/filepath"

	resourceport "github.com/vrooli/vrooli/packages/resource-port"
)

// getResourcePort queries the port registry for a resource's port
func getResourcePort(resourceName string) string {
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		root = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}
	portName := ""
	if resourceName == "postgres" {
		portName = "sql"
	}
	if port, err := resourceport.Resolve(root, resourceName, portName); err == nil {
		return port
	}
	return "8080"
}
