// Package bootstrap provides initialization helpers that wire up the application
// components during startup. This separates the "what gets registered" concern
// from the entry point.
//
// Responsibility: Orchestration layer - decides what checks get registered and
// with what configuration. Domain logic lives in the checks themselves.
package bootstrap

import (
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/checks/infra"
	"vrooli-autoheal/internal/checks/vrooli"
	"vrooli-autoheal/internal/platform"
)

// Default targets for infrastructure checks - defined here in bootstrap
// so check implementations stay pure and don't embed operational defaults.
const (
	DefaultNetworkTarget = "8.8.8.8:53" // Google DNS for connectivity check
	DefaultDNSDomain     = "google.com" // Reliable domain for DNS resolution check
)

// RegisterDefaultChecks adds all standard health checks to the registry.
// This centralizes check registration, keeping main.go focused on server setup.
// Platform capabilities are passed to checks that need them for runtime decisions.
func RegisterDefaultChecks(registry *checks.Registry, caps *platform.Capabilities) {
	// Infrastructure checks - explicit targets make dependencies clear
	registry.Register(infra.NewNetworkCheck(DefaultNetworkTarget))
	registry.Register(infra.NewDNSCheck(DefaultDNSDomain))
	registry.Register(infra.NewDockerCheck())
	registry.Register(infra.NewCloudflaredCheck(caps))
	registry.Register(infra.NewRDPCheck(caps))

	// Vrooli resource checks
	registry.Register(vrooli.NewResourceCheck("postgres"))
	registry.Register(vrooli.NewResourceCheck("redis"))
}
