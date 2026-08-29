// Package onboardingapplyprivileges provisions the one host grant used by
// onboarding's non-interactive apply run.  The grant is deliberately a list
// of literal argv entries; it never contains a sudoers wildcard.
package onboardingapplyprivileges

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const grantPath = "/etc/sudoers.d/vrooli-onboarding-apply"

var safeToken = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	if host.OS != "linux" && host.OS != "darwin" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "onboarding apply grants are supported only on Linux and macOS")
		return status
	}
	user := hostreqkit.InvokingUser()
	content, err := policyContent(user, requirement.Config)
	if err != nil {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, err.Error())
		return status
	}
	switch hostreqkit.CompareFileContent(grantPath, content) {
	case hostreqkit.FileComparisonMatch:
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status
	case hostreqkit.FileComparisonUnreadable:
		// A sudoers drop-in is mode 440 root:root by necessity — sudo refuses to
		// read one that is more permissive — so every unprivileged inspection
		// lands here on a correctly installed grant. Reporting that as missing
		// made a required item permanently unsatisfiable for `setup status`,
		// `setup --dry-run`, and the unprivileged onboarding API, which is a
		// false blocker on a healthy host. The privileged apply path still
		// reads the file and rewrites it when it has genuinely drifted, so the
		// stale case is detected exactly where it can also be repaired.
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes,
			"onboarding apply grant present at "+grantPath+"; its contents cannot be read without privilege, so this run did not re-verify them against the current selection")
		return status
	default:
		status.Notes = append(status.Notes, "onboarding apply grant missing or stale at "+grantPath+"; run `vrooli setup --sudo-mode=ask`")
		return status
	}
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.SupportClass == hostreqkit.SupportUnsupported {
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	content, err := policyContent(hostreqkit.InvokingUser(), status.Config)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would install "+grantPath)
		return status, nil
	}
	if err := hostreqkit.EnsureManagedDir("/etc/sudoers.d", opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "prepare sudoers directory: "+err.Error())
		return status, nil
	}
	tmp, err := hostreqkit.WriteTempFileFn(content)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "prepare onboarding apply grant: "+err.Error())
		return status, nil
	}
	defer os.Remove(tmp)
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "visudo", []string{"-c", "-f", tmp}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "sudoers validation failed; grant was not written: "+err.Error())
		return status, nil
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "install", []string{"-m", "440", tmp, grantPath}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "install onboarding apply grant: "+err.Error())
		return status, nil
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	return status, nil
}

// ConfigForRequirements is the setup-to-handler boundary. The values are
// plain JSON-compatible slices because ResolvedRequirement.Config is persisted
// and transported through the normal host requirement path.
func ConfigForRequirements(executable string, tools, safeguards []string) map[string]any {
	return map[string]any{
		"executable": executable,
		"tools":      append([]string(nil), tools...),
		"safeguards": append([]string(nil), safeguards...),
	}
}

func policyContent(user string, config map[string]any) (string, error) {
	if !safeToken.MatchString(strings.TrimSpace(user)) {
		return "", fmt.Errorf("onboarding apply grant cannot safely identify the invoking user")
	}
	executable, ok := config["executable"].(string)
	if !ok || strings.TrimSpace(executable) == "" || !strings.HasPrefix(executable, "/") || strings.ContainsAny(executable, "\t\r\n*?[]") {
		return "", fmt.Errorf("onboarding apply grant requires an absolute executable path")
	}
	// sudoers uses backslash escaping for whitespace in a command path. The
	// subprocess still receives the original absolute path, so paths containing
	// spaces remain portable without turning the policy into a shell fragment.
	sudoersExecutable := strings.NewReplacer("\\", "\\\\", " ", "\\ ").Replace(executable)
	tools, err := configTokens(config["tools"])
	if err != nil {
		return "", fmt.Errorf("onboarding apply grant tools: %w", err)
	}
	safeguards, err := configTokens(config["safeguards"])
	if err != nil {
		return "", fmt.Errorf("onboarding apply grant safeguards: %w", err)
	}
	entries := make([]string, 0, len(tools)+len(safeguards))
	for _, name := range tools {
		entries = append(entries, fmt.Sprintf("%s host install %s --json --sudo-mode error", sudoersExecutable, name))
	}
	for _, name := range safeguards {
		entries = append(entries, fmt.Sprintf("%s host safeguard %s --json --sudo-mode error", sudoersExecutable, name))
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("onboarding apply grant has no elevated commands")
	}
	slices.Sort(entries)
	return fmt.Sprintf("# Managed by Vrooli -- do not edit manually\n%s ALL=(root) NOPASSWD: %s\n", user, strings.Join(entries, ", ")), nil
}

func configTokens(value any) ([]string, error) {
	values, ok := value.([]string)
	if !ok {
		if raw, okRaw := value.([]any); okRaw {
			values = make([]string, 0, len(raw))
			for _, item := range raw {
				name, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("command names must be strings")
				}
				values = append(values, name)
			}
		} else {
			return nil, fmt.Errorf("command names must be an array")
		}
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !safeToken.MatchString(value) {
			return nil, fmt.Errorf("command name %q is not a safe literal token", value)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	slices.Sort(result)
	return result, nil
}
