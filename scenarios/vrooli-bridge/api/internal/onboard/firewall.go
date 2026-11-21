package onboard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

// UFWStatus is the conservative observation exposed to callers. Unsupported
// hosts must remain unknown rather than being presented as open.
type UFWStatus struct {
	Available   bool
	Active      bool
	RuleFound   bool
	Inspectable bool
	// Privileged is false unless a separately audited privileged boundary is
	// supplied. Bridge currently never runs UFW mutations itself.
	Privileged bool
}

// FirewallInspector is the read-only host-policy seam consumed by readiness.
// It deliberately does not expose rule mutation: collecting sudo credentials
// or executing a broad shell command would violate the admission contract.
type FirewallInspector interface {
	InspectUFW(context.Context, string, int) UFWStatus
}

// UFWObserver performs the safe, unprivileged `ufw status` observation on the
// Bridge host. Missing UFW and permission/command failures are uninspectable,
// never evidence that the port is open.
type UFWObserver struct{}

func NewUFWObserver() UFWObserver { return UFWObserver{} }

func (UFWObserver) InspectUFW(ctx context.Context, candidateIP string, port int) UFWStatus {
	cmd := exec.CommandContext(ctx, "ufw", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return UFWStatus{}
		}
		return UFWStatus{Available: true}
	}
	return ParseUFWStatus(string(output), candidateIP, port)
}

// ParseUFWStatus parses `ufw status` output without claiming that an absent
// rule means an open firewall. It accepts the normal English output emitted by
// UFW; callers retain Unknown when the command itself is unavailable.
func ParseUFWStatus(output, candidateIP string, port int) UFWStatus {
	text := strings.ToLower(output)
	status := UFWStatus{Available: true, Inspectable: true, Active: strings.Contains(text, "status: active")}
	if !status.Active {
		return status
	}
	needleIP, err := canonicalIP(candidateIP)
	if err != nil {
		return status
	}
	needlePort := strconv.Itoa(port)
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, needleIP) && strings.Contains(line, needlePort) && strings.Contains(line, "allow") {
			status.RuleFound = true
			break
		}
	}
	return status
}

func canonicalIP(raw string) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil {
		return "", fmt.Errorf("invalid IP")
	}
	return strings.ToLower(ip.String()), nil
}
