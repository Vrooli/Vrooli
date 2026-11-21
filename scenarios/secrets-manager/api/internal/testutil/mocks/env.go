// Package mocks contains deterministic test implementations of API seams.
package mocks

import "secrets-manager-api/internal/envx"

// FakeEnv is a map-backed environment reader for deterministic tests.
type FakeEnv map[string]string

func (f FakeEnv) Getenv(key string) string { return f[key] }

var _ envx.Reader = FakeEnv{}
