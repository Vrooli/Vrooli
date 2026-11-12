package runtimeapp

import "github.com/vrooli/vrooli/internal/lightingruntime"

// Service owns typed runtime supervisor and recovery operations. It carries no
// CLI parsing.
type Service struct {
	Version       string
	ResolveRootFn func() (string, error)
	// Lighting is set by the control-plane composition that provides both
	// concrete lighting owners. Ordinary runtime setup needs neither owner.
	Lighting *lightingruntime.Owners
}
