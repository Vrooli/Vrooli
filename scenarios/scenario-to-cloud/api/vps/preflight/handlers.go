// DOC: docs/reference/api-endpoints.md#preflight — preflight and fix endpoint documentation
package preflight

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/reach"
)

// HandlerDeps contains dependencies for preflight handlers.
type HandlerDeps struct {
	// Reach is the transport every host observation goes through. Preflight
	// runs before a deployment record exists, so the target is derived from
	// the manifest locator and reached with the operator's ambient identity
	// (or the enrolled Bridge node).
	Reach      reach.Reach
	DNSService dns.Service
	// ValidateManifest is a function that validates and normalizes a manifest.
	// Returns normalized manifest and validation issues.
	ValidateManifest func(manifest domain.CloudManifest) (domain.CloudManifest, []domain.ValidationIssue)
	// HasBlockingIssues checks if issues contain blocking errors.
	HasBlockingIssues func(issues []domain.ValidationIssue) bool
}

// HandleRequirements returns canonical VPS requirement metadata.
// GET /api/v1/preflight/requirements
func HandleRequirements() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, BuildRequirementsResponse())
	}
}

// HandlePreflight creates a handler for running VPS preflight checks.
// POST /api/v1/preflight
func HandlePreflight(deps HandlerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		manifest, err := httputil.DecodeJSON[domain.CloudManifest](r.Body, 1<<20)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_json",
				Message: "Request body must be valid JSON",
				Hint:    err.Error(),
			})
			return
		}

		normalized, issues := deps.ValidateManifest(manifest)
		if deps.HasBlockingIssues(issues) {
			httputil.WriteJSON(w, http.StatusUnprocessableEntity, domain.ManifestValidateResponse{
				Valid:     false,
				Issues:    issues,
				Manifest:  normalized,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()

		resp := Run(ctx, normalized, deps.DNSService, deps.Reach, domain.TargetRefFromManifest(normalized), RunOptions{})
		resp.Issues = issues
		httputil.WriteJSON(w, http.StatusOK, resp)
	}
}

// HostRequest names a target that has no deployment record yet: the locator
// only. There is no key reference on the wire; the transport authenticates
// with the credential binding of a bound deployment or the operator's
// ambient identity.
type HostRequest struct {
	Host string `json:"host"`
	Port int    `json:"port,omitempty"`
	User string `json:"user,omitempty"`
}

// FirewallFixRequest is the request body for opening firewall ports via UFW.
type FirewallFixRequest struct {
	HostRequest
	Ports []int `json:"ports,omitempty"`
}

// FirewallFixResponse is the response from opening firewall ports.
type FirewallFixResponse struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	Ports     []int  `json:"ports"`
	Status    string `json:"status,omitempty"`
	Timestamp string `json:"timestamp"`
}

// DiskUsageRequest is the request body for getting disk usage details.
type DiskUsageRequest struct {
	HostRequest
}

// DiskUsageEntry represents a single disk usage entry.
type DiskUsageEntry struct {
	Path  string `json:"path"`
	Size  string `json:"size"`
	Bytes int64  `json:"bytes"`
}

// DiskUsageResponse is the response from disk usage check.
type DiskUsageResponse struct {
	OK          bool             `json:"ok"`
	FreeSpace   string           `json:"free_space"`
	FreeBytes   int64            `json:"free_bytes"`
	TotalSpace  string           `json:"total_space"`
	TotalBytes  int64            `json:"total_bytes"`
	UsedPercent int              `json:"used_percent"`
	LargestDirs []DiskUsageEntry `json:"largest_dirs"`
	Error       string           `json:"error,omitempty"`
	Timestamp   string           `json:"timestamp"`
}

// diskUsageRoots are the directories whose immediate children are sized.
// The set is fixed: a caller cannot name a path.
var diskUsageRoots = []string{"/var", "/home", "/root", "/opt", "/tmp"}

// HandleDiskUsage reports root filesystem usage and the largest directories
// under the fixed roots, through df and du observations.
func HandleDiskUsage(reachFor ReachFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DiskUsageRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		target := preflightTarget(req.Host, req.Port, req.User, domain.DefaultVPSWorkdir)
		obs := observer{reach: reachFor(target), target: target}
		now := time.Now().UTC().Format(time.RFC3339)

		dfRes, err := obs.observe(ctx, "df", "-Pk", "/")
		totalKB, _, availKB, usedPct, found := dfRoot(dfRes)
		if !ok(dfRes, err) || !found {
			msg := "df did not answer"
			if err != nil {
				msg = err.Error()
			}
			httputil.WriteJSON(w, http.StatusOK, DiskUsageResponse{OK: false, Error: msg, Timestamp: now})
			return
		}

		httputil.WriteJSON(w, http.StatusOK, DiskUsageResponse{
			OK:          true,
			FreeSpace:   formatBytes(availKB),
			FreeBytes:   availKB * 1024,
			TotalSpace:  formatBytes(totalKB),
			TotalBytes:  totalKB * 1024,
			UsedPercent: usedPct,
			LargestDirs: largestDirectories(ctx, obs),
			Timestamp:   now,
		})
	}
}

// largestDirectories sizes the children of the fixed roots (du -sk on each
// root's entries, resolved through ls) and returns the ten largest.
func largestDirectories(ctx context.Context, obs observer) []DiskUsageEntry {
	var entries []DiskUsageEntry
	for _, root := range diskUsageRoots {
		list, err := obs.observe(ctx, "ls", "-A1", "--", root)
		if !ok(list, err) {
			continue
		}
		var paths []string
		for _, name := range strings.Split(list.Stdout, "\n") {
			name = strings.TrimSpace(name)
			if name == "" || reach.ValidateArgs([]string{name}) != nil {
				continue
			}
			paths = append(paths, root+"/"+name)
		}
		if len(paths) == 0 {
			continue
		}
		du, err := obs.observe(ctx, "du", append([]string{"-sk", "--"}, paths...)...)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(du.Stdout, "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			sizeKB, perr := strconv.ParseInt(fields[0], 10, 64)
			if perr != nil {
				continue
			}
			entries = append(entries, DiskUsageEntry{Path: fields[1], Size: formatBytes(sizeKB), Bytes: sizeKB * 1024})
		}
	}
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].Bytes > entries[j-1].Bytes; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
	if len(entries) > 10 {
		entries = entries[:10]
	}
	return entries
}

// DiskCleanupRequest is the request body for running disk cleanup.
type DiskCleanupRequest struct {
	HostRequest
	Actions []string `json:"actions"`
}

// DiskCleanupResponse is the response from disk cleanup.
type DiskCleanupResponse struct {
	OK            bool                      `json:"ok"`
	SpaceFreed    string                    `json:"space_freed"`
	SpaceFreedKB  int64                     `json:"space_freed_kb"`
	Message       string                    `json:"message"`
	ActionsRun    []string                  `json:"actions_run"`
	ActionsFailed []string                  `json:"actions_failed,omitempty"`
	ActionResults []DiskCleanupActionResult `json:"action_results,omitempty"`
	Timestamp     string                    `json:"timestamp"`
}

// DiskCleanupActionResult captures execution details for one cleanup action.
type DiskCleanupActionResult struct {
	Action   string `json:"action"`
	OK       bool   `json:"ok"`
	ExitCode int    `json:"exit_code"`
	Summary  string `json:"summary,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// cleanupAction is one disk cleanup the target owner can perform: the
// privilege-broker action and its closed subject.
type cleanupAction struct {
	broker  string
	subject any
	hint    string
}

// journalVacuumBytes is the journal size the vacuum keeps (100 MiB).
const journalVacuumBytes = 100 * 1024 * 1024

// CleanupActions is the closed set of disk cleanup actions and their owner
// operations. Every entry runs through `cloud-target host repair`.
var CleanupActions = map[string]cleanupAction{
	"journal_vacuum":       {broker: "journald.vacuum", subject: map[string]any{"journal": map[string]any{"max_use_bytes": journalVacuumBytes}}, hint: "Check journald permissions and the journal size settings, then retry."},
	"docker_prune":         {broker: "docker.prune.unused-images", subject: map[string]any{"docker": map[string]any{}}, hint: "Ensure Docker is installed and running, or drop docker_prune from actions."},
	"docker_prune_volumes": {broker: "docker.prune.unused-volumes", subject: map[string]any{"docker": map[string]any{}}, hint: "Ensure Docker is installed and running, or drop docker_prune_volumes from actions."},
}

// UnownedCleanupActions are the cleanup actions callers used to request that
// have no privilege-broker owner. They are refused by name so the missing
// owner is explicit rather than silently shelled.
var UnownedCleanupActions = map[string]string{
	"apt_clean": "no privilege-broker action owns apt cache cleaning (apt.packages.ensure only installs)",
	"tmp_clean": "no privilege-broker action owns /tmp expiry (a path-taking delete is refused by design)",
}

// DefaultCleanupActions run when the request names none.
var DefaultCleanupActions = []string{"journal_vacuum"}

// HandleDiskCleanup frees disk space through the target owner's host repair
// verb. An action without an owner is a typed unsupported_capability refusal
// before anything runs.
func HandleDiskCleanup(reachFor ReachFactory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DiskCleanupRequest
		if !httputil.DecodeRequestBody(w, r, &req) {
			return
		}
		if len(req.Actions) == 0 {
			req.Actions = append([]string(nil), DefaultCleanupActions...)
		}
		for _, action := range req.Actions {
			if _, owned := CleanupActions[action]; owned {
				continue
			}
			reason, known := UnownedCleanupActions[action]
			if !known {
				reason = "unknown cleanup action"
			}
			apierrors.Write(w, apierrors.New(apierrors.CodeUnsupportedCapability, "disk cleanup action has no owner operation: "+action).
				WithDetail("action", action).WithDetail("reason", reason).WithDetail("supported_actions", cleanupActionNames()).
				WithNextAction(apierrors.NextAction{Owner: "internal/privilegebroker", Kind: "capability", Reference: action, Label: "Add a privilege-broker action for " + action}))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()
		target := preflightTarget(req.Host, req.Port, req.User, domain.DefaultVPSWorkdir)
		rr := reachFor(target)
		obs := observer{reach: rr, target: target}

		before, _ := obs.observe(ctx, "df", "-Pk", "/")
		_, _, beforeKB, _, _ := dfRoot(before)

		var (
			actionsRun    []string
			actionsFailed []string
			actionResults []DiskCleanupActionResult
		)
		for _, action := range req.Actions {
			spec := CleanupActions[action]
			res, err := hostRepair(ctx, rr, target, spec.broker, spec.subject)
			failure := verbFailure(res, err)
			result := DiskCleanupActionResult{Action: action, OK: failure == "", ExitCode: res.ExitCode, Summary: summarizeDiskCleanupAction(res.Stdout, res.Stderr), Stderr: strings.TrimSpace(res.Stderr)}
			if failure != "" {
				result.Summary = failure
				result.Hint = spec.hint
				if err != nil && res.ExitCode == 0 {
					result.ExitCode = 1
				}
				actionsFailed = append(actionsFailed, action)
			} else {
				actionsRun = append(actionsRun, action)
			}
			actionResults = append(actionResults, result)
		}

		after, _ := obs.observe(ctx, "df", "-Pk", "/")
		_, _, afterKB, _, _ := dfRoot(after)
		freedKB := afterKB - beforeKB
		if freedKB < 0 {
			freedKB = 0
		}
		msg := fmt.Sprintf("Freed %s of disk space", formatBytes(freedKB))
		if freedKB == 0 {
			msg = "No significant space was freed"
		}
		httputil.WriteJSON(w, http.StatusOK, DiskCleanupResponse{
			OK:            len(actionsFailed) == 0,
			SpaceFreed:    formatBytes(freedKB),
			SpaceFreedKB:  freedKB,
			Message:       msg,
			ActionsRun:    actionsRun,
			ActionsFailed: actionsFailed,
			ActionResults: actionResults,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func cleanupActionNames() []string {
	names := make([]string, 0, len(CleanupActions))
	for name := range CleanupActions {
		names = append(names, name)
	}
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	return names
}

func summarizeDiskCleanupAction(stdout, stderr string) string {
	for _, source := range []string{stderr, stdout} {
		for _, line := range strings.Split(source, "\n") {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

// StopScenarioProcessesRequest is the request body for stopping stale scenario processes.
type StopScenarioProcessesRequest struct {
	HostRequest
	Workdir    string `json:"workdir"`
	ScenarioID string `json:"scenario_id,omitempty"` // If empty, stops all vrooli processes
}

// StopScenarioProcessesResponse is the response from stopping scenario processes.
type StopScenarioProcessesResponse struct {
	OK        bool   `json:"ok"`
	Action    string `json:"action"` // "stop_scenario" or "stop_all"
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}

// writeStopScenarioResponse is a helper to reduce response construction boilerplate.
func writeStopScenarioResponse(w http.ResponseWriter, ok bool, action, message, output string) {
	httputil.WriteJSON(w, http.StatusOK, StopScenarioProcessesResponse{
		OK:        ok,
		Action:    action,
		Message:   message,
		Output:    output,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
