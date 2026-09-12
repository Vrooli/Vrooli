// Package logging defines the minimal structured-output seam used across
// composition and integration adapters.
package logging

type Logger interface {
	Printf(string, ...any)
}
