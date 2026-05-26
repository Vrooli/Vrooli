//go:build windows

package config

// Windows has no sudo/uid model, so the owned-write seam performs no ownership
// changes — file ownership follows the creating process as usual.

func chownCreatedToInvokingUser(_ []string) error { return nil }

func chownPathToInvokingUser(_ string) error { return nil }

func reconcileHomeOwnership(_ string, _, _ int) (int, error) { return 0, nil }
