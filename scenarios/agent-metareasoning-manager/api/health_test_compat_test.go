//go:build testing

package main

import (
	"github.com/vrooli/api-core/health"
)

var Health = health.New(serviceName).
	Version(apiVersion).
	Handler()
