// Package app contains application boundaries for root CLI command families.
//
// The app layer exists when domain logic is shared by more than one consumer
// or when a command family needs a transport-free orchestration boundary.
// CLI-only pass-through packages are not promoted into this layer; resource,
// scenario, and project commands retain their direct domain bindings where
// that is their existing ownership boundary.
package app
