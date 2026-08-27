package sample

import "testing"

type fake struct{}

func (fake) LookPath(string) (string, error) { return "", nil }

func TestNameAndKind(t *testing.T) {
	t.Helper()
}

// #!/bin/sh
func TestShell(t *testing.T) { t.Helper() }

