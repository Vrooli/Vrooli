// Package testutil provides shared helpers for swarm-manager CLI tests.
package testutil

import (
	"reflect"

	"swarm-manager/cli/internal/support"
)

// StubDeps returns a support.Dependencies with every support.CommandFunc field
// populated with a non-nil no-op stub. This lets register wiring tests detect an
// unwired dependency: if a Register call references a field this helper did not
// set (because it is not a CommandFunc), or leaves a Run handler nil, the test
// can assert on it.
func StubDeps() support.Dependencies {
	var d support.Dependencies
	v := reflect.ValueOf(&d).Elem()
	cf := reflect.TypeOf(support.CommandFunc(nil))
	stub := support.CommandFunc(func([]string) error { return nil })
	for i := 0; i < v.NumField(); i++ {
		if f := v.Field(i); f.Type() == cf && f.CanSet() {
			f.Set(reflect.ValueOf(stub))
		}
	}
	return d
}
