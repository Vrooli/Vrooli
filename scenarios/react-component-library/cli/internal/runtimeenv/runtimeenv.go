// Package runtimeenv defines the small ambient dependencies used by CLI
// commands so tests can provide a stable clock and environment.
package runtimeenv

import (
	"os"
	"time"
)

type Clock interface {
	Now() time.Time
}

type Environment interface {
	LookupEnv(string) (string, bool)
}

type Dependencies struct {
	Clock       Clock
	Environment Environment
}

// System returns the production process clock and environment.
func System() Dependencies {
	return Dependencies{Clock: systemClock{}, Environment: processEnvironment{}}
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

type processEnvironment struct{}

func (processEnvironment) LookupEnv(key string) (string, bool) { return os.LookupEnv(key) }
