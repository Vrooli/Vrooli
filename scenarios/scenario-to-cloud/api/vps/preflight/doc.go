// Package preflight runs pre-deployment validation checks against a VPS target.
// It verifies SSH connectivity, OS version, disk space, RAM, network access,
// firewall rules, and credential validity before deployment begins.
//
// Main entry point is [Run] which executes all checks and returns a [domain.PreflightResponse].
package preflight
