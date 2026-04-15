// Package testkitgo provides the dependency-bottom shared Go test-support layer
// for Vrooli.
//
// The root package is intentionally narrow. It contains only repo fixtures,
// file/JSON/malformed writers, and small path-safe utilities that any Go test
// consumer in the repository can import without pulling in root-module
// internal/* dependencies.
package testkitgo
