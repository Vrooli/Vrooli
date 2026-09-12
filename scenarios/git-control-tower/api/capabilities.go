package main

import (
	"time"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

// These aliases preserve Git Control Tower's existing package-local API while
// making capability-registry-go the sole lifecycle-state implementation.
type DependencyKind = capabilityregistry.DependencyKind
type CapabilityStatus = capabilityregistry.Status
type CapabilityDef = capabilityregistry.Def
type CapabilityState = capabilityregistry.State
type StatusChecker = capabilityregistry.Checker
type CapabilityRegistry = capabilityregistry.Registry

const (
	DependencyScenario = capabilityregistry.DependencyScenario
	DependencyResource = capabilityregistry.DependencyResource
	StatusAvailable    = capabilityregistry.StatusAvailable
	StatusUnavailable  = capabilityregistry.StatusUnavailable
	StatusUnknown      = capabilityregistry.StatusUnknown
)

func NewCapabilityRegistry(defs []CapabilityDef, checkers map[string]StatusChecker, cacheTTL time.Duration) *CapabilityRegistry {
	return capabilityregistry.New(defs, checkers, cacheTTL)
}
