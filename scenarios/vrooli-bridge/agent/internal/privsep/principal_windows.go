//go:build windows

package privsep

import "os/exec"

// principal is unused on Windows: the provisioning service already runs as
// the account that owns the checkout.
type principal struct {
	home string
}

func checkoutPrincipal(string, int) (*principal, error) { return nil, nil }

func clientPrincipal(int) (*principal, error) { return nil, nil }

func (p *principal) apply(*exec.Cmd) {}

func chownTree(string, *principal) error { return nil }
