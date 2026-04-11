package runtime

import (
	"errors"
	"fmt"
	"strings"
)

var ErrUnsupportedPlatform = errors.New("unsupported platform")

type Host struct {
	OS              string   `json:"os"`
	PackageManager  string   `json:"package_manager,omitempty"`
	SupportsSetup   bool     `json:"supports_setup"`
	SupportsDevelop bool     `json:"supports_develop"`
	SupportsSysctl  bool     `json:"supports_sysctl"`
	SupportsSystemd bool     `json:"supports_systemd"`
	Notes           []string `json:"notes,omitempty"`
}

func Current() Host {
	return currentHost()
}

func (h Host) ValidateSetup() error {
	if h.SupportsSetup {
		return nil
	}
	return h.unsupportedError("setup")
}

func (h Host) ValidateDevelop() error {
	if h.SupportsDevelop {
		return nil
	}
	return h.unsupportedError("develop")
}

func (h Host) unsupportedError(command string) error {
	if len(h.Notes) == 0 {
		return fmt.Errorf("%w: vrooli %s is not supported on %s", ErrUnsupportedPlatform, command, defaultOS(h.OS))
	}
	return fmt.Errorf("%w: vrooli %s is not supported on %s (%s)", ErrUnsupportedPlatform, command, defaultOS(h.OS), strings.Join(h.Notes, "; "))
}

func defaultOS(value string) string {
	if strings.TrimSpace(value) == "" {
		return "this platform"
	}
	return value
}
