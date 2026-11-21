// Responsibility: refuse verified one-shot relay when live process identity cannot be proven.
//go:build !linux

package programs

import "github.com/vrooli/api-core/localprincipal"

func openKernelProcess(int) (int, error) { return -1, localprincipal.ErrUnsupported }
func kernelProcessAlive(int) bool        { return false }
func closeKernelProcess(int)             {}
