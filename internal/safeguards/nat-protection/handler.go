package natprotection

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// redirectRe matches iptables -S lines like:
//
//	-A OUTPUT -p tcp -m tcp --dport 443 -j REDIRECT --to-ports 8085
var redirectRe = regexp.MustCompile(`--dport\s+(\d+)\s.*--to-ports\s+(\d+)`)

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "NAT protection is only supported on Linux")
		return status
	}

	if _, err := hostreqkit.LookPathFn("iptables"); err != nil {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "iptables not available; NAT protection not applicable")
		return status
	}

	dead := findDeadRedirects()
	if len(dead) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "no dead NAT redirect rules found")
		return status
	}

	for _, r := range dead {
		status.Notes = append(status.Notes,
			fmt.Sprintf("dead redirect: port %s -> %s (nothing listening on %s)", r.srcPort, r.dstPort, r.dstPort))
	}
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual safeguard action required by manifest declaration")
		return status, nil
	}

	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would remove dead NAT redirect rules")
		return status, nil
	}

	dead := findDeadRedirects()
	if len(dead) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "no dead NAT redirect rules to remove")
		return status, nil
	}

	removed := 0
	for _, r := range dead {
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "iptables",
			[]string{"-t", "nat", "-D", "OUTPUT", "-p", "tcp", "--dport", r.srcPort, "-j", "REDIRECT", "--to-ports", r.dstPort},
			opts,
		); err != nil {
			status.Notes = append(status.Notes,
				fmt.Sprintf("failed to remove redirect %s->%s: %s", r.srcPort, r.dstPort, err))
			continue
		}
		removed++
		status.Notes = append(status.Notes,
			fmt.Sprintf("removed dead redirect: port %s -> %s", r.srcPort, r.dstPort))
	}

	// Verify no dead redirects remain.
	remaining := findDeadRedirects()
	if len(remaining) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionApplied
		status.Notes = append(status.Notes, fmt.Sprintf("removed %d dead NAT redirect rules", removed))
		return status, nil
	}

	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes,
		fmt.Sprintf("removed %d rules but %d dead redirects remain", removed, len(remaining)))
	return status, nil
}

type redirect struct {
	srcPort string
	dstPort string
}

func findDeadRedirects() []redirect {
	out, err := hostreqkit.CombinedOutputFn("iptables", "-t", "nat", "-S", "OUTPUT")
	if err != nil {
		return nil
	}

	ssOut, ssErr := hostreqkit.CombinedOutputFn("ss", "-tln")
	if ssErr != nil {
		return nil
	}
	ssLines := string(ssOut)

	var dead []redirect
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "REDIRECT") {
			continue
		}
		m := redirectRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		srcPort, dstPort := m[1], m[2]
		if !portListening(ssLines, dstPort) {
			dead = append(dead, redirect{srcPort: srcPort, dstPort: dstPort})
		}
	}

	// Sort by source port descending so removal by exact match works cleanly.
	sort.Slice(dead, func(i, j int) bool {
		pi, _ := strconv.Atoi(dead[i].srcPort)
		pj, _ := strconv.Atoi(dead[j].srcPort)
		return pi > pj
	})
	return dead
}

// portListening checks whether any line in ss -tln output shows a listener
// on the given port. It avoids false positives from partial matches (e.g.
// port 80 matching port 8080) by requiring a non-digit after the port number.
func portListening(ssOutput, port string) bool {
	marker := ":" + port
	for _, line := range strings.Split(ssOutput, "\n") {
		idx := strings.Index(line, marker)
		if idx < 0 {
			continue
		}
		after := idx + len(marker)
		if after >= len(line) || !isDigit(line[after]) {
			return true
		}
	}
	return false
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
