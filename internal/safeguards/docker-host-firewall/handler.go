package dockerhostfirewall

import (
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	handlerParameterF = 2
)

const (
	chainName       = "VROOLI-DOCKER"
	dockerUserChain = "DOCKER-USER"
)

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

	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "Docker host firewall management is only supported on Linux")
		return status
	}

	if _, err := hostreqkit.LookPathFn("iptables"); err != nil {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "iptables not available")
		return status
	}

	if _, err := hostreqkit.LookPathFn("docker"); err != nil {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "Docker not installed; firewall chain not needed")
		return status
	}

	chainOK := chainExists(chainName)
	wiredOK := chainWired(dockerUserChain, chainName)

	if chainOK && wiredOK {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, chainName+" chain exists and is wired into "+dockerUserChain)
		return status
	}

	pending := make([]string, 0, handlerParameterF)
	if !chainOK {
		pending = append(pending, "create "+chainName+" chain")
	}
	if !wiredOK {
		pending = append(pending, "wire "+chainName+" into "+dockerUserChain)
	}
	status.Notes = append(status.Notes, "pending: "+strings.Join(pending, ", "))
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
		status.Notes = append(status.Notes, "dry-run: would create "+chainName+" chain and wire into "+dockerUserChain)
		return status, nil
	}

	if !chainExists(chainName) {
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "iptables", []string{"-N", chainName}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "failed to create chain: "+err.Error())
			return status, nil
		}
	}

	if !chainWired(dockerUserChain, chainName) {
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "iptables", []string{"-I", dockerUserChain, "1", "-j", chainName}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "failed to wire chain: "+err.Error())
			return status, nil
		}
	}

	if chainExists(chainName) && chainWired(dockerUserChain, chainName) {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionApplied
		status.Notes = append(status.Notes, chainName+" chain created and wired into "+dockerUserChain)
		return status, nil
	}

	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, "chain commands completed but verification failed")
	return status, nil
}

func chainExists(chain string) bool {
	_, err := hostreqkit.CombinedOutputFn("iptables", "-n", "-L", chain)
	return err == nil
}

func chainWired(parent, child string) bool {
	_, err := hostreqkit.CombinedOutputFn("iptables", "-C", parent, "-j", child)
	return err == nil
}
