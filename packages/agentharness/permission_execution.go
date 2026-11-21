package agentharness

import (
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"regexp"
	"strings"
)

// ExecutionPermissions describes intent, not provider-specific configuration.
// All boundaries are explicit; omission leaves native execution settings alone.
type ExecutionPermissions struct {
	Filesystem ExecutionFilesystem `json:"filesystem" toml:"filesystem"`
	Network    ExecutionNetwork    `json:"network" toml:"network"`
	Approval   ExecutionApproval   `json:"approval" toml:"approval"`
}
type ExecutionFilesystem struct {
	Workspace     string   `json:"workspace" toml:"workspace"`
	WritableRoots []string `json:"writable_roots,omitempty" toml:"writable_roots,omitempty"`
}
type ExecutionNetwork struct {
	Enabled *bool             `json:"enabled" toml:"enabled"`
	Domains map[string]string `json:"domains,omitempty" toml:"domains,omitempty"`
}
type ExecutionApproval struct {
	Policy   string `json:"policy" toml:"policy"`
	Reviewer string `json:"reviewer" toml:"reviewer"`
}

// ExecutionCapability is configuration support, independent of runtime proof.
type ExecutionCapability struct {
	Runner string `json:"runner"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}
type ExecutionPlan struct {
	Desired              *ExecutionPermissions `json:"desired"`
	NativeNetworkEnabled bool                  `json:"native_network_enabled"`
	Capability           ExecutionCapability   `json:"capability"`
	Profile              string                `json:"profile"`
	Activate             bool                  `json:"activate"`
	RuntimeVersion       string                `json:"runtime_version,omitempty"`
	VersionCommand       []string              `json:"version_command,omitempty"`
	RuntimeEvidence      string                `json:"runtime_evidence"`
}

func PermissionExecutionCapability(runner string) ExecutionCapability {
	c := ExecutionCapability{Runner: runner, Status: "unsupported", Reason: "native execution projection has not been qualified for this adapter"}
	if runner == "codex" {
		c.Status = "supported"
		c.Reason = "user-scope profiles require Codex >= 0.138.0; filtered networking requires >= 0.156.1; runtime enforcement needs independent verification"
	}
	return c
}

func RequireExecutionCapability(document PermissionDocument, runner string) error {
	if document.Execution == nil {
		return nil
	}
	c := PermissionExecutionCapability(runner)
	if c.Status != "supported" {
		return fmt.Errorf("%s: execution permissions %s: %s", runner, c.Status, c.Reason)
	}
	return nil
}

func LoadPermissionDocumentForRunner(path string, stdin io.Reader, runner string) (PermissionDocument, []byte, error) {
	d, data, err := LoadPermissionDocument(path, stdin)
	if err == nil {
		err = RequireExecutionCapability(d, runner)
	}
	return d, data, err
}

var executionHost = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*$`)

func ValidateExecutionPermissions(e *ExecutionPermissions) error {
	if e == nil {
		return nil
	}
	var problems []error
	if e.Filesystem.Workspace != "read" && e.Filesystem.Workspace != "write" {
		problems = append(problems, errors.New("execution.filesystem.workspace must be read or write"))
	}
	seen := map[string]bool{}
	for _, root := range e.Filesystem.WritableRoots {
		if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == filepath.VolumeName(root)+string(filepath.Separator) {
			problems = append(problems, fmt.Errorf("writable root %q must be a clean absolute directory other than the filesystem root", root))
		}
		if seen[root] {
			problems = append(problems, fmt.Errorf("duplicate writable root %q", root))
		}
		seen[root] = true
	}
	if e.Network.Enabled == nil {
		problems = append(problems, errors.New("execution.network.enabled must be explicit"))
	}
	if e.Network.Enabled != nil && !*e.Network.Enabled && len(e.Network.Domains) > 0 {
		problems = append(problems, errors.New("disabled network cannot declare domains"))
	}
	for host, action := range e.Network.Domains {
		if len(host) > 253 || (net.ParseIP(host) == nil && !executionHost.MatchString(host)) || host != strings.ToLower(host) {
			problems = append(problems, fmt.Errorf("domain %q must be an exact lowercase hostname or IP literal; ports and wildcards are unsupported", host))
		}
		if action != "allow" && action != "deny" {
			problems = append(problems, fmt.Errorf("domain %q action must be allow or deny", host))
		}
	}
	if e.Approval.Policy != "on-request" && e.Approval.Policy != "never" {
		problems = append(problems, errors.New("execution.approval.policy must be on-request or never"))
	}
	if e.Approval.Reviewer != "user" && e.Approval.Reviewer != "auto_review" {
		problems = append(problems, errors.New("execution.approval.reviewer must be user or auto_review"))
	}
	if e.Approval.Policy == "never" && e.Approval.Reviewer == "auto_review" {
		problems = append(problems, errors.New("automatic review requires on-request approvals"))
	}
	return errors.Join(problems...)
}
