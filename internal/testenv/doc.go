// Package testenv owns cross-domain test fixtures: process identity, runtime
// homes, and repository trees on disk. Domain-specific fixture packages remain
// owners of shell, process-record, scenario, resource, package-governance, and
// host-requirement fixtures; they may build on testenv but do not move into it.
package testenv
