// Package pkgb is the other half of the cyclic-import fixture.
package pkgb

import (
	"github.com/vrooli/fixtures/go-cycles/pkga"
)

// TypeB is a small type declared in pkgb.
type TypeB struct {
	Ref *pkga.TypeA
}

// Greet returns a constant string; the import above completes the cycle.
func Greet() string {
	return "hello from pkgb"
}
