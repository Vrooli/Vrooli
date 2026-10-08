package cloudtarget

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/shell"
)

// Caddy layout the edge verbs own. The main Caddyfile belongs to the host
// (operator or another product); this owner only appends one import line to
// it, once, and otherwise writes exclusively beneath ConfDir.
const (
	DefaultCaddyMainConfig = privilegebroker.CaddyMainConfigPath
	DefaultCaddyConfDir    = "/etc/caddy/conf.d"
	DefaultCaddyDataDir    = "/var/lib/caddy/.local/share/caddy"
	caddyImportLine        = "import conf.d/*.caddy"
	snippetPrefix          = "vrooli-"
	snippetSuffix          = ".caddy"
	edgeDirName            = "edge"
	edgeActiveFile         = "active.caddy"
	edgePreviousFile       = "previous.caddy"
	edgeSpecFile           = "active-spec.json"
	confMode               = 0o644
)

// CaddyPaths locates the proxy configuration; zero values select the
// defaults above.
type CaddyPaths struct {
	MainConfig string
	ConfDir    string
	DataDir    string
}

func (p CaddyPaths) withDefaults() CaddyPaths {
	if p.MainConfig == "" {
		p.MainConfig = DefaultCaddyMainConfig
	}
	if p.ConfDir == "" {
		p.ConfDir = DefaultCaddyConfDir
	}
	if p.DataDir == "" {
		p.DataDir = DefaultCaddyDataDir
	}
	return p
}

// SnippetPath is the per-deployment file beneath ConfDir.
func (p CaddyPaths) SnippetPath(deploymentID string) string {
	return filepath.Join(p.withDefaults().ConfDir, snippetPrefix+deploymentID+snippetSuffix)
}

// CaddyController validates and reloads the proxy. Root is required for the
// reload, so production selects the privilege broker; the argv controller is
// for a target already running as the proxy owner and for tests.
type CaddyController interface {
	Validate(ctx context.Context) error
	Reload(ctx context.Context) error
}

// BrokerCaddyController delegates to privilegebroker edge.caddy.* actions.
type BrokerCaddyController struct {
	Client       BrokerClient
	DeploymentID string
}

func (c BrokerCaddyController) do(ctx context.Context, action, code string) error {
	req := privilegebroker.Request{Version: privilegebroker.ProtocolVersion, RequestID: "edge-" + randomSuffix(), Action: action, Caddy: &privilegebroker.CaddySubject{DeploymentID: c.DeploymentID}}
	if err := privilegebroker.Validate(req); err != nil {
		return refuse(CodeActionNotAllowed, "%s", err.Error())
	}
	if c.Client == nil || !c.Client.Available() {
		return fail(CodeBrokerUnavailable, "privilege broker is not available for %s", action)
	}
	result, err := c.Client.Do(ctx, req)
	if err != nil {
		return fail(code, "%s: %v", action, err)
	}
	if result.Status != "completed" {
		return fail(code, "%s: %s", action, strings.TrimSpace(result.Code+" "+result.Evidence.Detail))
	}
	return nil
}

func (c BrokerCaddyController) Validate(ctx context.Context) error {
	return c.do(ctx, privilegebroker.ActionEdgeCaddyValidate, CodeEdgeConfigInvalid)
}

func (c BrokerCaddyController) Reload(ctx context.Context) error {
	return c.do(ctx, privilegebroker.ActionEdgeCaddyReload, CodeEdgeReloadFailed)
}

// ArgvCaddyController runs the same fixed argv the broker policy builds,
// directly, for a process that already owns the proxy (root on a
// single-tenant target). It never composes a shell string.
type ArgvCaddyController struct {
	Runner       shell.Runner
	DeploymentID string
}

func (c ArgvCaddyController) run(ctx context.Context, action, code string) error {
	req := privilegebroker.Request{Version: privilegebroker.ProtocolVersion, RequestID: "edge-argv", Action: action, Caddy: &privilegebroker.CaddySubject{DeploymentID: c.DeploymentID}}
	name, args, err := privilegebroker.CaddyArgs(req)
	if err != nil {
		return refuse(CodeActionNotAllowed, "%s", err.Error())
	}
	runner := c.Runner
	if runner == nil {
		runner = shell.OSRunner{}
	}
	out, runErr := runner.Run(ctx, name, args...)
	if runErr != nil {
		return fail(code, "%s %s: %v: %s", name, strings.Join(args, " "), runErr, lastLines(string(out), 8))
	}
	return nil
}

func (c ArgvCaddyController) Validate(ctx context.Context) error {
	return c.run(ctx, privilegebroker.ActionEdgeCaddyValidate, CodeEdgeConfigInvalid)
}

func (c ArgvCaddyController) Reload(ctx context.Context) error {
	return c.run(ctx, privilegebroker.ActionEdgeCaddyReload, CodeEdgeReloadFailed)
}

// SelectCaddyController prefers the broker whenever it is installed; a
// process running as root without a broker uses argv; anything else fails
// closed at the first validate.
func SelectCaddyController(client BrokerClient, runner shell.Runner, deploymentID string) CaddyController {
	if client != nil && client.Available() {
		return BrokerCaddyController{Client: client, DeploymentID: deploymentID}
	}
	return ArgvCaddyController{Runner: runner, DeploymentID: deploymentID}
}

func lastLines(text string, n int) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// EdgeRouteApplyRequest is the input of `edge route apply`.
type EdgeRouteApplyRequest struct {
	Effect     EffectRequest
	Spec       EdgeRouteSpec
	Paths      CaddyPaths
	Controller CaddyController
}

// EdgeRouteReport is the typed outcome of apply and rollback.
type EdgeRouteReport struct {
	SnippetPath     string   `json:"snippet_path"`
	SpecDigest      string   `json:"spec_digest"`
	PreviousDigest  string   `json:"previous_digest,omitempty"`
	ImportLineAdded bool     `json:"import_line_added"`
	RolledBack      bool     `json:"rolled_back"`
	UnrelatedHash   string   `json:"unrelated_config_hash"`
	Hosts           []string `json:"hosts"`
}

// EdgeRouteApply writes the deployment snippet transactionally: the previous
// snippet is retained, the whole proxy configuration is validated, and a
// failed validate or reload restores the previous snippet so every other
// route keeps serving. Unrelated configuration (the main Caddyfile body and
// other deployments' snippets) is hashed before and after; a change is a
// failure of this verb, never something it does.
func (s *Store) EdgeRouteApply(ctx context.Context, req EdgeRouteApplyRequest) (EdgeRouteReport, EffectResult, error) {
	if err := req.Spec.Validate(); err != nil {
		return EdgeRouteReport{}, EffectResult{}, err
	}
	if req.Spec.DeploymentID != req.Effect.DeploymentID {
		return EdgeRouteReport{}, EffectResult{}, refuse(CodeEdgeSpecInvalid, "spec deployment %q does not match --deployment %q", req.Spec.DeploymentID, req.Effect.DeploymentID)
	}
	if req.Controller == nil {
		return EdgeRouteReport{}, EffectResult{}, refuse(CodeInvalidArgument, "caddy controller is required")
	}
	digest, err := req.Spec.SpecDigest()
	if err != nil {
		return EdgeRouteReport{}, EffectResult{}, fail(CodeInvalidArgument, "digest edge spec: %v", err)
	}
	effect := req.Effect
	effect.Verb = "edge.route.apply"
	effect.Input = map[string]any{"spec_digest": digest}
	var report EdgeRouteReport
	result, err := s.RunEffect(ctx, effect, func(ctx context.Context) (map[string]any, Outcome, error) {
		var outcome Outcome
		report, outcome, err = s.applySnippet(ctx, req, digest)
		details := map[string]any{"snippet_path": report.SnippetPath, "spec_digest": report.SpecDigest, "previous_digest": report.PreviousDigest, "import_line_added": report.ImportLineAdded, "rolled_back": report.RolledBack, "unrelated_config_hash": report.UnrelatedHash, "hosts": report.Hosts, "acme_environment": req.Spec.ACMEEnvironment}
		return details, outcome, err
	})
	return report, result, err
}

func (s *Store) applySnippet(ctx context.Context, req EdgeRouteApplyRequest, digest string) (EdgeRouteReport, Outcome, error) {
	paths := req.Paths.withDefaults()
	snippetPath := paths.SnippetPath(req.Spec.DeploymentID)
	report := EdgeRouteReport{SnippetPath: snippetPath, SpecDigest: digest, Hosts: hostsOf(req.Spec)}
	added, err := ensureImportLine(paths)
	if err != nil {
		return report, OutcomeFailed, err
	}
	report.ImportLineAdded = added
	before, err := unrelatedConfigHash(paths, snippetPath)
	if err != nil {
		return report, OutcomeFailed, err
	}
	report.UnrelatedHash = before
	previous, hadPrevious, err := readOptional(snippetPath)
	if err != nil {
		return report, OutcomeFailed, fail(CodeStoreIO, "read current snippet: %v", err)
	}
	if hadPrevious {
		report.PreviousDigest = sha256Hex(previous)
	}
	if hadPrevious && previous == req.Spec.Snippet {
		if err := s.recordEdgeState(req.Spec, previous, hadPrevious, false); err != nil {
			return report, OutcomeFailed, err
		}
		return report, OutcomeUnchanged, nil
	}
	if err := writeFileAtomic(snippetPath, req.Spec.Snippet); err != nil {
		return report, OutcomeFailed, fail(CodeStoreIO, "write snippet: %v", err)
	}
	if err := s.fault("edge:after_write"); err != nil {
		return report, OutcomeFailed, err
	}
	restore := func() error {
		if hadPrevious {
			return writeFileAtomic(snippetPath, previous)
		}
		if err := os.Remove(snippetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := req.Controller.Validate(ctx); err != nil {
		typed := AsError(err)
		if restoreErr := restore(); restoreErr != nil {
			return report, OutcomeFailed, fail(CodeEdgeConfigInvalid, "%s; restore of previous snippet also failed: %v", typed.Message, restoreErr)
		}
		report.RolledBack = true
		return report, OutcomeFailed, (&Error{Code: CodeEdgeConfigInvalid, Message: typed.Message, Exit: ExitFailed}).withDetails(map[string]any{"rolled_back": true, "previous_digest": report.PreviousDigest})
	}
	if err := req.Controller.Reload(ctx); err != nil {
		typed := AsError(err)
		if restoreErr := restore(); restoreErr != nil {
			return report, OutcomeFailed, fail(CodeEdgeReloadFailed, "%s; restore of previous snippet also failed: %v", typed.Message, restoreErr)
		}
		report.RolledBack = true
		// The previous configuration was the running one; a reload after
		// restore returns the proxy to a known-validated file.
		_ = req.Controller.Reload(ctx)
		return report, OutcomeFailed, (&Error{Code: CodeEdgeReloadFailed, Message: typed.Message, Exit: ExitFailed}).withDetails(map[string]any{"rolled_back": true, "previous_digest": report.PreviousDigest})
	}
	after, err := unrelatedConfigHash(paths, snippetPath)
	if err != nil {
		return report, OutcomeFailed, err
	}
	if after != before {
		return report, OutcomeFailed, fail(CodeEdgeUnrelatedChanged, "unrelated proxy configuration changed during apply (%s -> %s)", before, after).withDetails(map[string]any{"before": before, "after": after})
	}
	if err := s.recordEdgeState(req.Spec, previous, hadPrevious, true); err != nil {
		return report, OutcomeFailed, err
	}
	return report, OutcomeSucceeded, nil
}

// recordEdgeState keeps the active snippet, the displaced one and the spec
// beneath the deployment directory so rollback and status do not depend on
// the proxy directory alone.
func (s *Store) recordEdgeState(spec EdgeRouteSpec, previous string, hadPrevious, displaced bool) error {
	dir, err := s.DeploymentDir(spec.DeploymentID)
	if err != nil {
		return err
	}
	edgeDir := filepath.Join(dir, edgeDirName)
	if err := os.MkdirAll(edgeDir, dirMode); err != nil {
		return fail(CodeStoreIO, "create edge state dir: %v", err)
	}
	if err := writeFileAtomic(filepath.Join(edgeDir, edgeActiveFile), spec.Snippet); err != nil {
		return fail(CodeStoreIO, "record active snippet: %v", err)
	}
	if err := writeJSONAtomic(filepath.Join(edgeDir, edgeSpecFile), spec); err != nil {
		return fail(CodeStoreIO, "record active spec: %v", err)
	}
	if displaced && hadPrevious {
		if err := writeFileAtomic(filepath.Join(edgeDir, edgePreviousFile), previous); err != nil {
			return fail(CodeStoreIO, "record previous snippet: %v", err)
		}
	}
	return nil
}

// EdgeRouteRollbackRequest is the input of `edge route rollback`.
type EdgeRouteRollbackRequest struct {
	Effect     EffectRequest
	Paths      CaddyPaths
	Controller CaddyController
}

// EdgeRouteRollback restores the snippet displaced by the last successful
// apply. It is eligible only while a previous snippet is retained.
func (s *Store) EdgeRouteRollback(ctx context.Context, req EdgeRouteRollbackRequest) (EdgeRouteReport, EffectResult, error) {
	if req.Controller == nil {
		return EdgeRouteReport{}, EffectResult{}, refuse(CodeInvalidArgument, "caddy controller is required")
	}
	dir, err := s.DeploymentDir(req.Effect.DeploymentID)
	if err != nil {
		return EdgeRouteReport{}, EffectResult{}, err
	}
	previous, hasPrevious, err := readOptional(filepath.Join(dir, edgeDirName, edgePreviousFile))
	if err != nil {
		return EdgeRouteReport{}, EffectResult{}, fail(CodeStoreIO, "read previous snippet: %v", err)
	}
	if !hasPrevious {
		return EdgeRouteReport{}, EffectResult{}, refuse(CodeEdgeRollbackNotEligible, "deployment %s retains no previous edge snippet", req.Effect.DeploymentID)
	}
	paths := req.Paths.withDefaults()
	snippetPath := paths.SnippetPath(req.Effect.DeploymentID)
	effect := req.Effect
	effect.Verb = "edge.route.rollback"
	effect.Input = map[string]any{"previous_digest": sha256Hex(previous)}
	report := EdgeRouteReport{SnippetPath: snippetPath, SpecDigest: sha256Hex(previous)}
	result, err := s.RunEffect(ctx, effect, func(ctx context.Context) (map[string]any, Outcome, error) {
		details := map[string]any{"snippet_path": snippetPath, "restored_digest": report.SpecDigest}
		before, err := unrelatedConfigHash(paths, snippetPath)
		if err != nil {
			return details, OutcomeFailed, err
		}
		report.UnrelatedHash = before
		details["unrelated_config_hash"] = before
		current, _, err := readOptional(snippetPath)
		if err != nil {
			return details, OutcomeFailed, fail(CodeStoreIO, "read current snippet: %v", err)
		}
		report.PreviousDigest = sha256Hex(current)
		if err := writeFileAtomic(snippetPath, previous); err != nil {
			return details, OutcomeFailed, fail(CodeStoreIO, "restore snippet: %v", err)
		}
		if err := req.Controller.Validate(ctx); err != nil {
			_ = writeFileAtomic(snippetPath, current)
			return details, OutcomeFailed, err
		}
		if err := req.Controller.Reload(ctx); err != nil {
			_ = writeFileAtomic(snippetPath, current)
			_ = req.Controller.Reload(ctx)
			return details, OutcomeFailed, err
		}
		edgeDir := filepath.Join(dir, edgeDirName)
		_ = writeFileAtomic(filepath.Join(edgeDir, edgeActiveFile), previous)
		_ = writeFileAtomic(filepath.Join(edgeDir, edgePreviousFile), current)
		report.RolledBack = true
		details["rolled_back"] = true
		return details, OutcomeSucceeded, nil
	})
	return report, result, err
}

func hostsOf(spec EdgeRouteSpec) []string {
	hosts := make([]string, 0, len(spec.Routes))
	for _, route := range spec.Routes {
		hosts = append(hosts, strings.ToLower(strings.TrimSpace(route.Host)))
	}
	sort.Strings(hosts)
	return hosts
}

// ensureImportLine appends the conf.d import to the main Caddyfile exactly
// once. It never rewrites, reorders or removes anything already there; a
// missing main Caddyfile is created with only the import line.
func ensureImportLine(paths CaddyPaths) (bool, error) {
	if err := os.MkdirAll(paths.ConfDir, dirMode); err != nil {
		return false, fail(CodeStoreIO, "create %s: %v", paths.ConfDir, err)
	}
	current, exists, err := readOptional(paths.MainConfig)
	if err != nil {
		return false, fail(CodeStoreIO, "read %s: %v", paths.MainConfig, err)
	}
	for _, line := range strings.Split(current, "\n") {
		if strings.TrimSpace(line) == caddyImportLine {
			return false, nil
		}
	}
	var updated string
	switch {
	case !exists || strings.TrimSpace(current) == "":
		updated = caddyImportLine + "\n"
	case strings.HasSuffix(current, "\n"):
		updated = current + caddyImportLine + "\n"
	default:
		updated = current + "\n" + caddyImportLine + "\n"
	}
	if err := writeFileAtomic(paths.MainConfig, updated); err != nil {
		return false, fail(CodeStoreIO, "append import line to %s: %v", paths.MainConfig, err)
	}
	return true, nil
}

// unrelatedConfigHash digests everything in the proxy configuration this
// deployment does not own: the main Caddyfile and every other snippet.
func unrelatedConfigHash(paths CaddyPaths, ownSnippet string) (string, error) {
	h := sha256.New()
	main, _, err := readOptional(paths.MainConfig)
	if err != nil {
		return "", fail(CodeStoreIO, "read %s: %v", paths.MainConfig, err)
	}
	fmt.Fprintf(h, "main\x00%s\x00", main)
	entries, err := os.ReadDir(paths.ConfDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fail(CodeStoreIO, "read %s: %v", paths.ConfDir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), snippetSuffix) {
			continue
		}
		path := filepath.Join(paths.ConfDir, entry.Name())
		if path == ownSnippet {
			continue
		}
		names = append(names, path)
	}
	sort.Strings(names)
	for _, path := range names {
		body, _, err := readOptional(path)
		if err != nil {
			return "", fail(CodeStoreIO, "read %s: %v", path, err)
		}
		fmt.Fprintf(h, "%s\x00%s\x00", filepath.Base(path), body)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readOptional(path string) (string, bool, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // paths are owner-built beneath fixed roots
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(raw), true, nil
}

func writeFileAtomic(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return err
	}
	tmp := path + ".tmp-" + randomSuffix()
	if err := os.WriteFile(tmp, []byte(body), confMode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func sha256Hex(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}
