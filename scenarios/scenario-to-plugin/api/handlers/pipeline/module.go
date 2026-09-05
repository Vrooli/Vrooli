package pipeline

// Package pipeline owns the six delivery-ramp seams. The implementation is
// deliberately fail-closed: declarations are read from the governed
// manifest, package records contain paths/digests rather than opaque bytes,
// and distribution requires an external release decision.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"scenario-to-plugin/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	approvalsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/approvals/approvalsv1connect"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	evidencev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
	att "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/attestation"
	attconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/attestation/attestation_v1connect"
	comp "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/composition"
	compconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/composition/composition_v1connect"
	conf "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/conformance"
	confconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/conformance/conformance_v1connect"
	decl "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/declaration"
	declconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/declaration/declaration_v1connect"
	dist "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/distribution"
	distconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/distribution/distribution_v1connect"
	reh "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/rehearsal"
	rehconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/rehearsal/rehearsal_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type serviceManifest struct {
	Service struct{ Name, Description, Version string } `json:"service"`
	Plugin  *struct {
		Slug            string `json:"slug"`
		EntitlementTier string `json:"entitlement_tier"`
		Skills          []struct {
			Name, Source  string
			CommandGroups []string `json:"command_groups"`
		} `json:"skills"`
		MCP *struct {
			Name           string   `json:"name"`
			Command        string   `json:"command"`
			Args           []string `json:"args"`
			Authentication string   `json:"authentication"`
		} `json:"mcp"`
		Standalone struct {
			InstallScript   string   `json:"install_script"`
			RuntimeBinaries []string `json:"runtime_binaries"`
			Resources       []string `json:"resources"`
		} `json:"standalone"`
	} `json:"plugin"`
}

type packageRecord struct {
	Package      *comp.Package
	Root         string
	ScenarioRoot string
}
type handler struct {
	root     string
	db       *sql.DB
	mu       sync.RWMutex
	packages map[string]packageRecord
}

func New(repoRoot string) module.Module {
	return NewWithDB(repoRoot, nil)
}

func NewWithDB(repoRoot string, db *sql.DB) module.Module {
	h := &handler{root: repoRoot, db: db, packages: map[string]packageRecord{}}
	return module.Module{Name: "pipeline", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/readiness", func(w http.ResponseWriter, req *http.Request) {
			items, _ := os.ReadDir(filepath.Join(h.root, "scenarios"))
			out := make([]*decl.Readiness, 0, len(items))
			for _, item := range items {
				if !item.IsDir() {
					continue
				}
				_, readiness, _, err := h.readiness(item.Name())
				if err == nil {
					out = append(out, readiness)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/package/compose", h.composeREST).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/package/{id}", h.packageREST).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/distributability", h.distributabilityREST).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/publish", h.publishREST).Methods(http.MethodPost)
		path, endpoint := declconnect.NewDeclarationServiceHandler(h)
		mount(r, path, endpoint)
		path, endpoint = compconnect.NewCompositionServiceHandler(h)
		mount(r, path, endpoint)
		path, endpoint = confconnect.NewConformanceServiceHandler(h)
		mount(r, path, endpoint)
		path, endpoint = attconnect.NewAttestationServiceHandler(h)
		mount(r, path, endpoint)
		path, endpoint = rehconnect.NewRehearsalServiceHandler(h)
		mount(r, path, endpoint)
		path, endpoint = distconnect.NewDistributionServiceHandler(h)
		mount(r, path, endpoint)
	}, Endpoints: Endpoints}
}

func (h *handler) distributabilityREST(w http.ResponseWriter, req *http.Request) {
	var in struct {
		Scenario          string `json:"scenario"`
		TargetCLIManifest string `json:"targetCliManifest"`
	}
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil || strings.TrimSpace(in.Scenario) == "" || strings.TrimSpace(in.TargetCLIManifest) == "" {
		writeJSON(w, map[string]string{"error": "scenario and targetCliManifest are required"}, http.StatusBadRequest)
		return
	}
	report, err := distributabilityForScenario(h.root, in.Scenario, in.TargetCLIManifest)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusPreconditionFailed)
		return
	}
	writeJSON(w, report, http.StatusOK)
}

func writeJSON(w http.ResponseWriter, value any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func (h *handler) composeREST(w http.ResponseWriter, req *http.Request) {
	var in comp.ComposeRequest
	if json.NewDecoder(req.Body).Decode(&in) != nil {
		writeJSON(w, map[string]string{"error": "invalid JSON"}, http.StatusBadRequest)
		return
	}
	resp, err := h.Compose(req.Context(), connect.NewRequest(&in))
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusPreconditionFailed)
		return
	}
	writeJSON(w, resp.Msg, http.StatusOK)
}
func (h *handler) packageREST(w http.ResponseWriter, req *http.Request) {
	resp, err := h.GetPackage(req.Context(), connect.NewRequest(&comp.GetPackageRequest{PackageId: mux.Vars(req)["id"]}))
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusNotFound)
		return
	}
	writeJSON(w, resp.Msg, http.StatusOK)
}
func (h *handler) publishREST(w http.ResponseWriter, req *http.Request) {
	var in dist.PublishRequest
	if json.NewDecoder(req.Body).Decode(&in) != nil {
		writeJSON(w, map[string]string{"error": "invalid JSON"}, http.StatusBadRequest)
		return
	}
	resp, err := h.Publish(req.Context(), connect.NewRequest(&in))
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusPreconditionFailed)
		return
	}
	writeJSON(w, resp.Msg, http.StatusOK)
}

func mount(r *mux.Router, resultPath string, result http.Handler) {
	r.PathPrefix(resultPath).Handler(result)
}

func (h *handler) scenarioRoot(name string) string { return filepath.Join(h.root, "scenarios", name) }

func (h *handler) read(name string) (serviceManifest, string, error) {
	root := h.scenarioRoot(name)
	b, err := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	if err != nil {
		return serviceManifest{}, root, connect.NewError(connect.CodeNotFound, fmt.Errorf("scenario %q has no governed manifest", name))
	}
	var m serviceManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return m, root, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid governed manifest: %w", err))
	}
	return m, root, nil
}

func prerequisite(code, description string, satisfied bool) *decl.Prerequisite {
	return &decl.Prerequisite{Code: code, Description: description, Satisfied: satisfied}
}

func (h *handler) readiness(name string) (*decl.Declaration, *decl.Readiness, string, error) {
	m, root, err := h.read(name)
	if err != nil {
		return nil, nil, root, err
	}
	r := &decl.Readiness{Scenario: name}
	if m.Plugin == nil {
		r.Prerequisites = []*decl.Prerequisite{prerequisite("PLG-DECL-SOURCE", "scenario has no plugin declaration", false)}
		r.BlockingPrerequisite = "PLG-DECL-SOURCE"
		return nil, r, root, nil
	}
	p := m.Plugin
	tier := p.EntitlementTier
	if tier == "" {
		tier = "free"
	}
	d := &decl.Declaration{Scenario: name, Slug: p.Slug, EntitlementTier: tier, Standalone: &decl.Standalone{InstallScript: p.Standalone.InstallScript, RuntimeBinaries: p.Standalone.RuntimeBinaries, Resources: p.Standalone.Resources}}
	if p.MCP != nil {
		d.Mcp = &decl.MCP{Name: p.MCP.Name, Command: p.MCP.Command, Args: p.MCP.Args, Authentication: p.MCP.Authentication}
	}
	for _, s := range p.Skills {
		d.Skills = append(d.Skills, &decl.Skill{Name: s.Name, Source: s.Source, CommandGroups: s.CommandGroups})
	}
	checks := []struct {
		code, text string
		ok         bool
	}{
		{"PLG-DECL-SOURCE", "plugin declaration is present", true},
		{"PLG-DECL-STANDALONE", "standalone install script, runtime binaries, and resource set are declared", p.Standalone.InstallScript != "" && len(p.Standalone.RuntimeBinaries) > 0},
	}
	for _, c := range checks {
		r.Prerequisites = append(r.Prerequisites, prerequisite(c.code, c.text, c.ok))
		if !c.ok && r.BlockingPrerequisite == "" {
			r.BlockingPrerequisite = c.code
		}
	}
	r.Eligible = r.BlockingPrerequisite == ""
	return d, r, root, nil
}

func (h *handler) GetDeclaration(_ context.Context, req *connect.Request[decl.GetDeclarationRequest]) (*connect.Response[decl.GetDeclarationResponse], error) {
	d, r, _, err := h.readiness(req.Msg.Scenario)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&decl.GetDeclarationResponse{Declaration: d, Readiness: r}), nil
}
func (h *handler) ListReadiness(_ context.Context, req *connect.Request[decl.ListReadinessRequest]) (*connect.Response[decl.ListReadinessResponse], error) {
	names := req.Msg.Scenarios
	if len(names) == 0 {
		entries, _ := os.ReadDir(filepath.Join(h.root, "scenarios"))
		for _, e := range entries {
			if e.IsDir() {
				names = append(names, e.Name())
			}
		}
	}
	resp := &decl.ListReadinessResponse{}
	for _, name := range names {
		_, r, _, err := h.readiness(name)
		if err == nil {
			resp.Items = append(resp.Items, r)
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) Compose(_ context.Context, req *connect.Request[comp.ComposeRequest]) (*connect.Response[comp.ComposeResponse], error) {
	d, r, scenarioRoot, err := h.readiness(req.Msg.Scenario)
	if err != nil {
		return nil, err
	}
	if !r.Eligible {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(r.BlockingPrerequisite))
	}
	m, _, _ := h.read(req.Msg.Scenario)
	selectedSkills := d.Skills
	if strings.EqualFold(strings.TrimSpace(req.Msg.EntitlementTier), "tier-2") && strings.TrimSpace(req.Msg.TargetCliManifest) == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("PLG-DIST-TARGET: tier-2 composition requires target_cli_manifest"))
	}
	if strings.TrimSpace(req.Msg.TargetCliManifest) != "" {
		report, reportErr := distributabilityForScenario(h.root, req.Msg.Scenario, req.Msg.TargetCliManifest)
		if reportErr != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("PLG-DIST-MANIFEST: %w", reportErr))
		}
		for _, item := range report.Skills {
			if !item.Distributable {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("PLG-DIST-DRIFT: skill %s requires missing command(s): %s", item.Skill, strings.Join(item.MissingCommands, ", ")))
			}
		}
	}
	revision := req.Msg.SourceRevision
	if revision == "" {
		revision = "working-tree"
	}
	seed := []byte(req.Msg.Scenario + "\x00" + revision + "\x00" + time.Now().UTC().Format(time.RFC3339Nano))
	sum := sha256.Sum256(seed)
	id := hex.EncodeToString(sum[:8])
	root, err := os.MkdirTemp("", "vrooli-plugin-"+id+"-")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	manifest := map[string]any{"$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json", "name": d.Slug, "version": m.Service.Version, "description": m.Service.Description}
	for _, s := range selectedSkills {
		src := filepath.Join(scenarioRoot, s.Source)
		body, e := os.ReadFile(src)
		if e != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%s: %w", s.Source, e))
		}
		dst := filepath.Join(root, s.Source)
		if e = os.MkdirAll(filepath.Dir(dst), 0755); e != nil {
			return nil, e
		}
		if e = os.WriteFile(dst, body, 0644); e != nil {
			return nil, e
		}
	}
	if m.Plugin.MCP != nil {
		server := map[string]any{"type": "stdio", "command": m.Plugin.MCP.Command}
		if len(m.Plugin.MCP.Args) > 0 {
			server["args"] = m.Plugin.MCP.Args
		}
		if !strings.HasPrefix(m.Plugin.MCP.Command, "./") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("PLG-COMPOSE-MCP: bundled command must be plugin-relative"))
		}
		mcp := map[string]any{"$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json", "mcpServers": map[string]any{m.Plugin.MCP.Name: server}}
		if err := os.WriteFile(filepath.Join(root, "mcp.json"), mustJSON(mcp), 0644); err != nil {
			return nil, err
		}
	}
	if err := validatePluginManifest(manifest); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err := os.WriteFile(filepath.Join(root, "plugin.json"), mustJSON(manifest), 0644); err != nil {
		return nil, err
	}
	for _, rel := range append([]string{m.Plugin.Standalone.InstallScript}, m.Plugin.Standalone.RuntimeBinaries...) {
		src := filepath.Join(scenarioRoot, rel)
		dst := filepath.Join(root, rel)
		b, e := os.ReadFile(src)
		if e != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("standalone artifact %s: %w", rel, e))
		}
		if e = os.MkdirAll(filepath.Dir(dst), 0755); e != nil {
			return nil, e
		}
		if e = os.WriteFile(dst, b, 0755); e != nil {
			return nil, e
		}
	}
	artifact, artifactDigest, err := packageArchive(root)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("archive package: %w", err))
	}
	if err := os.WriteFile(filepath.Join(root, ".agent-plugin.tar.gz"), artifact, 0644); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	posture := ""
	if m.Plugin.MCP != nil {
		posture = m.Plugin.MCP.Authentication
	}
	p := &comp.Package{Id: id, Scenario: req.Msg.Scenario, SourceRevision: revision, Digest: artifactDigest, ArtifactRoot: root, State: "composed", McpAuthentication: posture}
	h.mu.Lock()
	h.packages[id] = packageRecord{Package: p, Root: root, ScenarioRoot: scenarioRoot}
	h.mu.Unlock()
	if h.db != nil {
		if _, err := h.db.Exec(`INSERT OR REPLACE INTO plugin_packages (id, scenario, source_revision, digest, artifact_root, state, mcp_authentication) VALUES (?, ?, ?, ?, ?, ?, ?)`, p.Id, p.Scenario, p.SourceRevision, p.Digest, p.ArtifactRoot, p.State, p.McpAuthentication); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("persist package record: %w", err))
		}
	}
	return connect.NewResponse(&comp.ComposeResponse{Package: p}), nil
}

// packageArchive produces the consumer artifact with stable tar metadata. The
// archive excludes generated evidence and OCI projections so the package
// digest remains the identity of the composed tree, not of a later attestation.
func packageArchive(root string) ([]byte, string, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || path == filepath.Join(root, ".agent-plugin.tar.gz") || strings.HasPrefix(filepath.ToSlash(path), filepath.ToSlash(filepath.Join(root, "oci"))) || strings.HasPrefix(filepath.Base(path), "cosign.") || filepath.Base(path) == "provenance.intoto.json" || filepath.Base(path) == "bom.json" {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported artifact entry %s", path)
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	sort.Strings(paths)
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	gz.Header.ModTime = time.Unix(0, 0)
	tw := tar.NewWriter(gz)
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, "", err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, "", err
		}
		header := &tar.Header{Name: filepath.ToSlash(rel), Mode: int64(info.Mode().Perm()), Size: info.Size(), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(header); err != nil {
			return nil, "", err
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, "", err
		}
		_, copyErr := io.Copy(tw, file)
		closeErr := file.Close()
		if copyErr != nil {
			return nil, "", copyErr
		}
		if closeErr != nil {
			return nil, "", closeErr
		}
	}
	if err := tw.Close(); err != nil {
		return nil, "", err
	}
	if err := gz.Close(); err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(out.Bytes())
	return out.Bytes(), "sha256:" + hex.EncodeToString(digest[:]), nil
}
func (h *handler) GetPackage(_ context.Context, req *connect.Request[comp.GetPackageRequest]) (*connect.Response[comp.GetPackageResponse], error) {
	h.mu.RLock()
	p, ok := h.packages[req.Msg.PackageId]
	h.mu.RUnlock()
	if !ok {
		if h.db != nil {
			var p comp.Package
			if err := h.db.QueryRow(`SELECT id, scenario, source_revision, digest, artifact_root, state, mcp_authentication FROM plugin_packages WHERE id = ?`, req.Msg.PackageId).Scan(&p.Id, &p.Scenario, &p.SourceRevision, &p.Digest, &p.ArtifactRoot, &p.State, &p.McpAuthentication); err == nil {
				root := p.ArtifactRoot
				h.mu.Lock()
				h.packages[p.Id] = packageRecord{Package: &p, Root: root, ScenarioRoot: h.scenarioRoot(p.Scenario)}
				h.mu.Unlock()
				return connect.NewResponse(&comp.GetPackageResponse{Package: &p}), nil
			}
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.New("package not found"))
	}
	return connect.NewResponse(&comp.GetPackageResponse{Package: p.Package}), nil
}

func (h *handler) Check(_ context.Context, req *connect.Request[conf.CheckRequest]) (*connect.Response[conf.CheckResponse], error) {
	h.mu.RLock()
	p, ok := h.packages[req.Msg.PackageId]
	h.mu.RUnlock()
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("package not found"))
	}
	out := &conf.CheckResponse{Passed: true, ManifestRevision: cliManifestRevision(filepath.Join(p.ScenarioRoot, "cli", "manifest.json"))}
	b, _ := os.ReadFile(filepath.Join(p.Root, "plugin.json"))
	if len(b) == 0 {
		out.Passed = false
		out.Findings = append(out.Findings, &conf.Finding{Code: "PLG-CONF-SPEC", Message: "plugin.json is missing"})
	}
	out.Findings = append(out.Findings, h.conformance(p)...)
	if len(out.Findings) > 0 {
		out.Passed = false
	}
	return connect.NewResponse(out), nil
}
func (h *handler) Attest(_ context.Context, req *connect.Request[att.AttestRequest]) (*connect.Response[att.AttestResponse], error) {
	h.mu.RLock()
	p, ok := h.packages[req.Msg.PackageId]
	h.mu.RUnlock()
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("package not found"))
	}
	check, err := h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: req.Msg.PackageId}))
	if err != nil {
		return nil, err
	}
	if !check.Msg.Passed {
		return connect.NewResponse(&att.AttestResponse{Passed: false, ArtifactDigest: p.Package.Digest, Findings: []*att.Finding{{Code: "PLG-ATTEST-ORDER", Message: "conformance must pass before attestation"}}}), nil
	}
	if scanner := os.Getenv("SCENARIO_TO_PLUGIN_SCANNER_FINDINGS"); scanner != "" {
		var findings []struct {
			Severity, Message string `json:"severity"`
		}
		if json.Unmarshal([]byte(scanner), &findings) == nil {
			for _, finding := range findings {
				if strings.EqualFold(finding.Severity, "high") || strings.EqualFold(finding.Severity, "critical") {
					return connect.NewResponse(&att.AttestResponse{Passed: false, ArtifactDigest: p.Package.Digest, Findings: []*att.Finding{{Code: "PLG-ATTEST-SCAN", Message: finding.Severity + ": " + finding.Message}}}), nil
				}
			}
		}
	}
	if secret := findSecret(p.Root); secret != "" {
		return connect.NewResponse(&att.AttestResponse{Passed: false, ArtifactDigest: p.Package.Digest, Findings: []*att.Finding{{Code: "PLG-ATTEST-NO-SECRETS", Message: "credential-like literal found at " + secret}}}), nil
	}
	if !req.Msg.DryRun {
		managedDir := strings.TrimSpace(os.Getenv("SCENARIO_TO_PLUGIN_ATTESTATION_DIR"))
		if managedDir == "" {
			return connect.NewResponse(&att.AttestResponse{Passed: false, ArtifactDigest: p.Package.Digest, Findings: []*att.Finding{{Code: "PLG-ATTEST-TOOLING", Message: "managed attestation requires SCENARIO_TO_PLUGIN_ATTESTATION_DIR containing cosign.signature.json, provenance.intoto.json, and bom.json"}}}), nil
		}
		for _, name := range []string{"cosign.signature.json", "provenance.intoto.json", "bom.json"} {
			b, readErr := os.ReadFile(filepath.Join(managedDir, name))
			if readErr != nil || len(bytes.TrimSpace(b)) == 0 {
				return connect.NewResponse(&att.AttestResponse{Passed: false, ArtifactDigest: p.Package.Digest, Findings: []*att.Finding{{Code: "PLG-ATTEST-TOOLING", Message: "managed attestation is missing " + name}}}), nil
			}
			if secretInBytes(b) {
				return connect.NewResponse(&att.AttestResponse{Passed: false, ArtifactDigest: p.Package.Digest, Findings: []*att.Finding{{Code: "PLG-ATTEST-NO-SECRETS", Message: "credential-like literal found in managed attestation " + name}}}), nil
			}
			if validationErr := validateManagedEvidence(name, b, p.Package.Digest); validationErr != nil {
				return connect.NewResponse(&att.AttestResponse{Passed: false, ArtifactDigest: p.Package.Digest, Findings: []*att.Finding{{Code: "PLG-ATTEST-TOOLING", Message: validationErr.Error()}}}), nil
			}
			if writeErr := os.WriteFile(filepath.Join(p.Root, name), b, 0644); writeErr != nil {
				return nil, connect.NewError(connect.CodeInternal, writeErr)
			}
		}
	}
	refs := []struct {
		kind, name string
		value      any
	}{
		{"cosign-signature", "cosign.signature.json", map[string]string{"digest": p.Package.Digest, "mode": map[bool]string{true: "dry-run", false: "managed-authority"}[req.Msg.DryRun]}},
		{"slsa-provenance", "provenance.intoto.json", map[string]any{"subject": p.Package.Digest, "source_revision": p.Package.SourceRevision, "build_environment": "vrooli-scenario-to-plugin"}},
		{"cyclonedx-sbom", "bom.json", map[string]any{"bomFormat": "CycloneDX", "specVersion": "1.5", "version": 1, "metadata": map[string]string{"component": p.Package.Scenario, "digest": p.Package.Digest}}},
	}
	evidence := make([]*att.Evidence, 0, len(refs))
	for _, ref := range refs {
		if !req.Msg.DryRun {
			// Managed files were produced by the external signing/provenance/SBOM
			// toolchain above; never replace them with locally invented JSON.
			b, readErr := os.ReadFile(filepath.Join(p.Root, ref.name))
			if readErr != nil || len(bytes.TrimSpace(b)) == 0 {
				return connect.NewResponse(&att.AttestResponse{Passed: false, ArtifactDigest: p.Package.Digest, Findings: []*att.Finding{{Code: "PLG-ATTEST-TOOLING", Message: "managed attestation file is empty: " + ref.name}}}), nil
			}
			evidence = append(evidence, &att.Evidence{Kind: ref.kind, Digest: p.Package.Digest, Reference: "artifact://" + ref.name})
			continue
		}
		b, _ := json.MarshalIndent(ref.value, "", "  ")
		if err := os.WriteFile(filepath.Join(p.Root, ref.name), b, 0644); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		evidence = append(evidence, &att.Evidence{Kind: ref.kind, Digest: p.Package.Digest, Reference: "artifact://" + ref.name})
	}
	return connect.NewResponse(&att.AttestResponse{Passed: true, ArtifactDigest: p.Package.Digest, Evidence: evidence}), nil
}

func findSecret(root string) string {
	var found string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || found != "" || info.IsDir() {
			return nil
		}
		b, e := os.ReadFile(path)
		if e == nil && secretInBytes(b) {
			found, _ = filepath.Rel(root, path)
		}
		return nil
	})
	return found
}

func secretInBytes(b []byte) bool {
	text := string(b)
	return strings.Contains(text, "BEGIN PRIVATE KEY") || strings.Contains(text, "ghp_") || strings.Contains(text, "sk-")
}

func validateManagedEvidence(name string, body []byte, artifactDigest string) error {
	var document map[string]any
	if err := json.Unmarshal(body, &document); err != nil {
		return fmt.Errorf("managed attestation %s is not valid JSON: %w", name, err)
	}
	switch name {
	case "cosign.signature.json":
		if _, ok := document["verificationMaterial"]; !ok {
			return errors.New("managed Cosign bundle has no verificationMaterial")
		}
		if _, ok := document["messageSignature"]; !ok {
			return errors.New("managed Cosign bundle has no messageSignature")
		}
	case "provenance.intoto.json":
		subjects, ok := document["subject"].([]any)
		if !ok || len(subjects) == 0 {
			return errors.New("managed provenance has no subject")
		}
		want := strings.TrimPrefix(artifactDigest, "sha256:")
		bound := false
		for _, raw := range subjects {
			entry, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			digests, ok := entry["digest"].(map[string]any)
			if !ok {
				continue
			}
			if value, ok := digests["sha256"].(string); ok && value == want {
				bound = true
			}
		}
		if !bound {
			return errors.New("managed provenance subject is not bound to the composed artifact digest")
		}
	case "bom.json":
		if document["bomFormat"] != "CycloneDX" {
			return errors.New("managed SBOM is not CycloneDX")
		}
		if _, ok := document["components"].([]any); !ok {
			return errors.New("managed CycloneDX SBOM has no components array")
		}
	}
	return nil
}
func (h *handler) Run(_ context.Context, req *connect.Request[reh.RunRequest]) (*connect.Response[reh.RunResponse], error) {
	h.mu.RLock()
	p, ok := h.packages[req.Msg.PackageId]
	h.mu.RUnlock()
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("package not found"))
	}
	if req.Msg.Sandbox != "" && req.Msg.Sandbox != "workspace-sandbox" {
		return connect.NewResponse(&reh.RunResponse{Passed: false, Findings: []*reh.Finding{{Code: "PLG-REHEARSE-ISOLATE", Message: "rehearsal requires workspace-sandbox isolation"}}}), nil
	}
	script := filepath.Join(p.Root, "cli/install.sh")
	if _, err := os.Stat(script); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("PLG-REHEARSE-CLEAN: install script missing"))
	}
	prefix, _ := os.MkdirTemp("", "hello-plugin-rehearsal-")
	defer os.RemoveAll(prefix)
	defer cleanupRehearsalProcess(prefix)
	rehearsalPort := ""
	if listener, listenErr := net.Listen("tcp", "127.0.0.1:0"); listenErr == nil {
		rehearsalPort = strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
		_ = listener.Close()
	}
	if rehearsalPort == "" {
		return connect.NewResponse(&reh.RunResponse{Passed: false, Findings: []*reh.Finding{{Code: "PLG-REHEARSE-ISOLATE", Message: "could not allocate an isolated loopback port"}}}), nil
	}
	workspaceStarted := false
	cleanupSandbox := func() error {
		if !workspaceStarted {
			return nil
		}
		return cleanupRehearsalSandboxes(rehearsalPort)
	}
	defer func() { _ = cleanupSandbox() }()
	results := make([]*reh.CommandResult, 0)
	for i := 0; i < 2; i++ {
		cmd := exec.Command("sh", script)
		cmd.Env = []string{"HELLO_PLUGIN_PREFIX=" + prefix, "WORKSPACE_SANDBOX_PREFIX=" + prefix, "XDG_BIN_HOME=" + prefix, "HOME=" + filepath.Join(prefix, "home"), "PATH=" + prefix + ":/usr/bin:/bin", "WORKSPACE_SANDBOX_STANDALONE_PORT=" + rehearsalPort}
		_ = os.MkdirAll(filepath.Join(prefix, "home"), 0700)
		if out, err := cmd.CombinedOutput(); err != nil {
			return connect.NewResponse(&reh.RunResponse{Passed: false, Findings: []*reh.Finding{{Code: "PLG-REHEARSE-IDEMPOTENT", Message: string(out)}}}), nil
		}
	}
	for _, documented := range h.documentedCommands(p) {
		fields := strings.Fields(documented)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "workspace-sandbox" {
			workspaceStarted = true
		}
		if installed := filepath.Join(prefix, fields[0]); fileExists(installed) {
			fields[0] = installed
		}
		cmdCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		cmd := exec.CommandContext(cmdCtx, fields[0], fields[1:]...)
		cmd.Env = []string{"HOME=" + filepath.Join(prefix, "home"), "PATH=" + prefix + ":/usr/bin:/bin", "PLUGIN_ROOT=" + p.Root, "PLUGIN_DATA=" + filepath.Join(prefix, "data"), "WORKSPACE_SANDBOX_STANDALONE_PORT=" + rehearsalPort}
		out, err := cmd.CombinedOutput()
		cancel()
		code := 0
		if err != nil {
			if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
				err = fmt.Errorf("command timed out after 20s")
			}
			code = 1
		}
		results = append(results, &reh.CommandResult{Command: documented, ExitCode: int32(code), RedactedOutput: redact(string(out))})
		if err != nil {
			return connect.NewResponse(&reh.RunResponse{Passed: false, Commands: results, Findings: []*reh.Finding{{Code: "PLG-REHEARSE-COMMANDS", Message: "documented command failed: " + documented}}}), nil
		}
	}
	// The documented journey intentionally leaves the sandbox alive after
	// promotion. Delete it before terminating the standalone API so its driver
	// can unmount cleanly and no FUSE daemon is orphaned by the rehearsal.
	if cleanupErr := cleanupSandbox(); cleanupErr != nil {
		return connect.NewResponse(&reh.RunResponse{Passed: false, Commands: results, Findings: []*reh.Finding{{Code: "PLG-REHEARSE-CLEAN", Message: cleanupErr.Error()}}}), nil
	}
	journey := "capture-store://rehearsal/" + p.Package.Id + "?profile=protocol&redacted=true"
	if err := h.reportRehearsalVerdict(context.Background(), p, journey); err != nil {
		return connect.NewResponse(&reh.RunResponse{Passed: false, Commands: results, JourneyManifest: journey, Findings: []*reh.Finding{{Code: "PLG-REHEARSE-EVIDENCE", Message: err.Error()}}}), nil
	}
	return connect.NewResponse(&reh.RunResponse{Passed: true, Commands: results, JourneyManifest: journey}), nil
}

func cleanupRehearsalSandboxes(port string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	list, err := client.Get("http://127.0.0.1:" + port + "/api/v1/sandboxes")
	if err != nil {
		return err
	}
	defer list.Body.Close()
	if list.StatusCode/100 != 2 {
		return fmt.Errorf("sandbox list returned HTTP %d", list.StatusCode)
	}
	var payload struct {
		Sandboxes []struct {
			ID string `json:"id"`
		} `json:"sandboxes"`
	}
	if err := json.NewDecoder(list.Body).Decode(&payload); err != nil {
		return err
	}
	for _, sandbox := range payload.Sandboxes {
		if sandbox.ID == "" {
			continue
		}
		req, err := http.NewRequest(http.MethodDelete, "http://127.0.0.1:"+port+"/api/v1/sandboxes/"+sandbox.ID, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("sandbox delete returned HTTP %d", resp.StatusCode)
		}
	}
	return nil
}

func cleanupRehearsalProcess(prefix string) {
	pidPath := filepath.Join(prefix, "home", ".local", "state", "workspace-sandbox", "api.pid")
	b, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return
	}
	if process, err := os.FindProcess(pid); err == nil {
		_ = process.Kill()
	}
}

func (h *handler) reportRehearsalVerdict(ctx context.Context, p packageRecord, journey string) error {
	base := strings.TrimRight(os.Getenv("DEPLOYMENT_MANAGER_API_BASE"), "/")
	if base == "" {
		// Local rehearsal is useful without the governance plane; publication
		// remains closed until the managed evidence path is configured.
		return nil
	}
	refs := make([]*commonv1.EvidenceRef, 0, 4)
	for _, item := range []struct {
		name string
		kind string
	}{{".agent-plugin.tar.gz", "agent-plugin-artifact"}, {"cosign.signature.json", "cosign-signature"}, {"provenance.intoto.json", "slsa-provenance"}, {"bom.json", "cyclonedx-sbom"}} {
		path := filepath.Join(p.Root, item.name)
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(b)
		refs = append(refs, &commonv1.EvidenceRef{Producer: "scenario-to-plugin", ArtifactId: journey + "/" + item.name, Kind: item.kind, Checksum: "sha256:" + hex.EncodeToString(sum[:]), SizeBytes: int64(len(b)), CreatedAt: timestamppb.Now()})
	}
	if len(refs) == 0 {
		return errors.New("no evidence references available for managed target verdict")
	}
	client := evidencev1connect.NewEvidenceServiceClient(http.DefaultClient, base)
	_, err := client.ReportTargetVerdict(ctx, connect.NewRequest(&evidencev1.ReportTargetVerdictRequest{
		ProfileId:     "scenario-to-plugin",
		GitCommitHash: p.Package.SourceRevision,
		Verdict: &commonv1.TargetVerdict{
			Target:        &commonv1.EvidenceTarget{Ramp: "scenario-to-plugin", Platform: runtime.GOOS, Os: runtime.GOOS, DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST},
			Disposition:   commonv1.Disposition_DISPOSITION_PASSED,
			Refs:          refs,
			RunId:         p.Package.Id,
			Detail:        "protocol-profile rehearsal passed",
			EvidenceClass: "protocol",
		},
	}))
	return err
}

func (h *handler) documentedCommands(p packageRecord) []string {
	commands := make([]string, 0)
	for _, skill := range p.PackageSkills() {
		body, err := os.ReadFile(filepath.Join(p.Root, skill))
		if err != nil {
			continue
		}
		inCode := false
		for _, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				inCode = !inCode
				continue
			}
			if !inCode {
				continue
			}
			if match := commandLine.FindStringSubmatch(line); len(match) > 2 {
				commands = append(commands, strings.TrimSpace(match[0]))
			}
		}
	}
	return commands
}

func fileExists(path string) bool { info, err := os.Stat(path); return err == nil && !info.IsDir() }
func redact(value string) string {
	for _, token := range []string{"sk-", "ghp_", "BEGIN PRIVATE KEY"} {
		value = strings.ReplaceAll(value, token, "[REDACTED]")
	}
	if len(value) > 2048 {
		return value[:2048]
	}
	return value
}
func (h *handler) Publish(ctx context.Context, req *connect.Request[dist.PublishRequest]) (*connect.Response[dist.PublishResponse], error) {
	h.mu.RLock()
	p, ok := h.packages[req.Msg.PackageId]
	h.mu.RUnlock()
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("package not found"))
	}
	if req.Msg.SourceRevision != "" && req.Msg.SourceRevision != p.Package.SourceRevision {
		return connect.NewResponse(&dist.PublishResponse{Refusal: "PLG-DIST-GATE: request source revision does not match the composed package"}), nil
	}
	gate, gateErr := h.deploymentManagerGate(ctx, p.Package.SourceRevision)
	if gateErr != nil {
		return connect.NewResponse(&dist.PublishResponse{Refusal: "PLG-DIST-GATE: deployment-manager release decision unavailable: " + gateErr.Error()}), nil
	}
	if !gate {
		return connect.NewResponse(&dist.PublishResponse{Refusal: "PLG-DIST-GATE: no passing deployment-manager release decision for this source revision"}), nil
	}
	if strings.TrimSpace(req.Msg.Channel) == "" {
		return connect.NewResponse(&dist.PublishResponse{Refusal: "PLG-DIST-GATE: publication channel is required"}), nil
	}
	attested, err := h.Attest(ctx, connect.NewRequest(&att.AttestRequest{PackageId: req.Msg.PackageId, DryRun: false}))
	if err != nil || !attested.Msg.Passed {
		return connect.NewResponse(&dist.PublishResponse{Refusal: "PLG-DIST-ATTEST: package lacks managed passing attestation"}), nil
	}
	registry, err := newOCIRegistry(os.Getenv("PLUGIN_REGISTRY"))
	if err != nil {
		return connect.NewResponse(&dist.PublishResponse{Refusal: "PLG-DIST-REGISTRY: " + err.Error()}), nil
	}
	artifact, err := os.ReadFile(filepath.Join(p.Root, ".agent-plugin.tar.gz"))
	if err != nil {
		return connect.NewResponse(&dist.PublishResponse{Refusal: "PLG-DIST-REGISTRY: composed artifact archive is missing"}), nil
	}
	readEvidence := func(name string) ([]byte, error) { return os.ReadFile(filepath.Join(p.Root, name)) }
	signature, sigErr := readEvidence("cosign.signature.json")
	provenance, provErr := readEvidence("provenance.intoto.json")
	sbom, sbomErr := readEvidence("bom.json")
	if sigErr != nil || provErr != nil || sbomErr != nil {
		return connect.NewResponse(&dist.PublishResponse{Refusal: "PLG-DIST-ATTEST: managed evidence files are incomplete"}), nil
	}
	coordinate, err := registry.pushPackage(ctx, req.Msg.Channel, artifact, signature, provenance, sbom)
	if err != nil {
		return connect.NewResponse(&dist.PublishResponse{Refusal: "PLG-DIST-CONFIRM: " + err.Error()}), nil
	}
	if h.db != nil {
		if _, err := h.db.ExecContext(ctx, `INSERT OR REPLACE INTO plugin_publications (package_id, channel, digest, coordinate, withdrawn) VALUES (?, ?, ?, ?, 0)`, p.Package.Id, req.Msg.Channel, p.Package.Digest, coordinate); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&dist.PublishResponse{Published: true, Coordinate: coordinate, Digest: p.Package.Digest, ConfirmationReference: coordinate}), nil
}

func (h *handler) deploymentManagerGate(ctx context.Context, sourceRevision string) (bool, error) {
	base := strings.TrimRight(os.Getenv("DEPLOYMENT_MANAGER_API_BASE"), "/")
	if base == "" {
		return false, errors.New("DEPLOYMENT_MANAGER_API_BASE is not configured")
	}
	client := approvalsv1connect.NewApprovalsServiceClient(http.DefaultClient, base)
	value, err := structpb.NewStruct(map[string]any{"profile_id": "scenario-to-plugin", "git_commit_hash": sourceRevision})
	if err != nil {
		return false, err
	}
	response, err := client.CheckReleaseGate(ctx, connect.NewRequest(structpb.NewStructValue(value)))
	if err != nil {
		return false, err
	}
	if response.Msg.GetStructValue() == nil {
		return false, errors.New("deployment-manager returned a non-object gate response")
	}
	ready := response.Msg.GetStructValue().GetFields()["ready"]
	return ready != nil && ready.GetBoolValue(), nil
}
func (h *handler) Revoke(_ context.Context, req *connect.Request[dist.RevokeRequest]) (*connect.Response[dist.RevokeResponse], error) {
	h.mu.RLock()
	_, ok := h.packages[req.Msg.PackageId]
	h.mu.RUnlock()
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("package not found"))
	}
	out := &dist.RevokeResponse{Complete: true}
	if h.db != nil {
		rows, err := h.db.Query(`SELECT channel FROM plugin_publications WHERE package_id = ? AND withdrawn = 0`, req.Msg.PackageId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		defer rows.Close()
		for rows.Next() {
			var channel string
			if err := rows.Scan(&channel); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			outcome := &dist.ChannelOutcome{Channel: channel, Withdrawn: true, Detail: "local publication withdrawn"}
			if _, err := h.db.Exec(`UPDATE plugin_publications SET withdrawn = 1 WHERE package_id = ? AND channel = ?`, req.Msg.PackageId, channel); err != nil {
				outcome.Withdrawn = false
				outcome.Detail = err.Error()
				out.Complete = false
			}
			out.Outcomes = append(out.Outcomes, outcome)
		}
		if err := rows.Err(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(out), nil
}

func mustJSON(v any) []byte { b, _ := json.MarshalIndent(v, "", "  "); return b }

func validatePluginManifest(manifest map[string]any) error {
	allowed := map[string]bool{"$schema": true, "name": true, "version": true, "description": true, "author": true, "homepage": true, "repository": true, "license": true, "keywords": true, "extensions": true}
	for key := range manifest {
		if !allowed[key] {
			return fmt.Errorf("PLG-COMPOSE-SPEC: plugin.json field %q is not permitted by Agent Plugins 1.0.0", key)
		}
	}
	if manifest["$schema"] != "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json" {
		return errors.New("PLG-COMPOSE-SPEC: plugin.json has an invalid Agent Plugins schema identifier")
	}
	name, ok := manifest["name"].(string)
	if !ok || !validPluginName(name) {
		return errors.New("PLG-COMPOSE-SPEC: plugin.json name is invalid")
	}
	return nil
}

func validPluginName(name string) bool {
	if name == "" || len(name) > 64 || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") || strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") || strings.Contains(name, "--") || strings.Contains(name, "..") {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' && r != '.' {
			return false
		}
	}
	return true
}
