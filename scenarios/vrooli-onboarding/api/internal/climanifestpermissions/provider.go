// Package climanifestpermissions exposes the CLI owners' versioned effect and
// process-spawn declarations as read-only onboarding metadata. These
// declarations describe command contracts; they are not authorization grants.
package climanifestpermissions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

const (
	ProviderID        = "onboarding.cli-manifest-declarations"
	Owner             = "cli-core"
	PermissionID      = "onboarding/cli-manifest-declarations"
	PermissionScope   = "cli-core:read"
	PermissionSubject = "cli-manifest-catalog"
	SchemaID          = "cli-manifest/v1"
	SchemaPath        = ".vrooli/schemas/cli-manifest.schema.json"
)

type Provider struct{ root string }

func New(root string) *Provider { return &Provider{root: strings.TrimSpace(root)} }

var _ permissions.Provider = (*Provider)(nil)
var _ permissions.PermissionIDMatcher = (*Provider)(nil)

func (*Provider) ProviderID() string                 { return ProviderID }
func (*Provider) Owner() string                      { return Owner }
func (*Provider) MatchesPermissionID(id string) bool { return id == PermissionID }

type Snapshot struct {
	Source            string        `json:"source"`
	SourceRevision    string        `json:"source_revision"`
	SourceDigest      string        `json:"source_digest"`
	SchemaID          string        `json:"schema_id"`
	Declarations      []Declaration `json:"declarations"`
	AuthorizationNote string        `json:"authorization_note"`
}

// Declaration is a manifest statement paired with the scope catalog's
// derived dispatch scope. The permission strings remain explicitly declared
// metadata and never become grants by appearing here.
type Declaration struct {
	Manifest             string   `json:"manifest"`
	Scenario             string   `json:"scenario"`
	DeclaringOwner       string   `json:"declaring_owner"`
	Group                string   `json:"group"`
	Command              string   `json:"command"`
	Effect               string   `json:"declared_effect"`
	RunEligible          bool     `json:"declared_run_eligible"`
	Permissions          []string `json:"declared_permissions,omitempty"`
	ProcessSpawnDeclared bool     `json:"process_spawn_declared"`
	DerivedScope         string   `json:"derived_scope"`
	DispatchEligible     bool     `json:"dispatch_eligible"`
	AuthorizationSource  string   `json:"authorization_source"`
}

type sourceManifest struct {
	Name   string        `json:"name"`
	Groups []sourceGroup `json:"groups"`
}
type sourceGroup struct {
	Name     string          `json:"name"`
	Flat     bool            `json:"flat"`
	Groups   []sourceGroup   `json:"groups"`
	Commands []sourceCommand `json:"commands"`
}
type sourceCommand struct {
	Name       string `json:"name"`
	Governance struct {
		Effect      string   `json:"effect"`
		RunEligible *bool    `json:"run_eligible"`
		Permissions []string `json:"permissions"`
	} `json:"governance"`
}

func (p *Provider) snapshot(ctx context.Context) (Snapshot, error) {
	if p == nil || p.root == "" {
		return Snapshot{}, connect.NewError(connect.CodeUnavailable, errors.New("CLI manifest catalog root is unavailable"))
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	// Build performs the canonical manifest path walk and validates every
	// document against the checked-in schema. It fails closed rather than
	// silently omitting a malformed or stale declaration.
	catalog, err := scopecatalog.Build(p.root)
	if err != nil {
		return Snapshot{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("validate CLI manifest catalog: %w", err))
	}
	schemaBytes, err := os.ReadFile(filepath.Join(p.root, filepath.FromSlash(SchemaPath)))
	if err != nil {
		return Snapshot{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("read CLI manifest schema: %w", err))
	}
	var schema struct {
		ID string `json:"$id"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil || schema.ID != SchemaID {
		return Snapshot{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("CLI manifest schema ID is missing or unsupported"))
	}
	paths := []string{filepath.Join(p.root, "cli", "manifest.json")}
	entries, err := os.ReadDir(filepath.Join(p.root, "scenarios"))
	if err != nil {
		return Snapshot{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("read scenario catalog: %w", err))
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(p.root, "scenarios", entry.Name(), "cli", "manifest.json")
		info, statErr := os.Stat(path)
		if statErr == nil && !info.IsDir() {
			paths = append(paths, path)
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return Snapshot{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("stat CLI manifest %s: %w", path, statErr))
		}
	}
	sort.Strings(paths)
	byCommand := make(map[string][]scopecatalog.Scope, len(catalog.Scopes))
	for _, scope := range catalog.Scopes {
		key := scope.Scenario + "\x00" + scope.Command
		byCommand[key] = append(byCommand[key], scope)
	}
	hash := sha256.New()
	hash.Write(schemaBytes)
	result := Snapshot{Source: "cli/manifest.json + scenarios/*/cli/manifest.json", SchemaID: schema.ID,
		AuthorizationNote: "Manifest effect, run eligibility, and process:spawn are declarations. For remote dispatch, Bridge checks caller authorization and the node's derived command scope plus Bridge transport scope; non-run-eligible commands are excluded. The audited dispatch path does not separately enforce process:spawn. Local commands use their owner handler and local OS identity. This projection grants nothing."}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return Snapshot{}, err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return Snapshot{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("read CLI manifest %s: %w", path, err))
		}
		hash.Write([]byte(filepath.ToSlash(mustRelative(p.root, path))))
		hash.Write(raw)
		var manifest sourceManifest
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		if err := dec.Decode(&manifest); err != nil {
			return Snapshot{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("decode CLI manifest %s: %w", path, err))
		}
		var trailing any
		if err := dec.Decode(&trailing); err != io.EOF {
			return Snapshot{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("CLI manifest %s has trailing JSON", path))
		}
		if strings.TrimSpace(manifest.Name) == "" {
			return Snapshot{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("CLI manifest %s has no owner name", path))
		}
		scenario := manifest.Name
		if filepath.Clean(path) == filepath.Clean(paths[0]) {
			scenario = scopecatalog.ProjectManifestIdentity
		} else if filepath.Base(filepath.Dir(filepath.Dir(path))) != manifest.Name {
			return Snapshot{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("CLI manifest path owner does not match manifest name: %s declares %s", path, manifest.Name))
		}
		if err := collect(&result.Declarations, byCommand, mustRelative(p.root, path), scenario, nil, manifest.Groups); err != nil {
			return Snapshot{}, connect.NewError(connect.CodeFailedPrecondition, err)
		}
	}
	result.SourceDigest = "sha256:" + hex.EncodeToString(hash.Sum(nil))
	cmd := exec.CommandContext(ctx, "git", "-C", p.root, "rev-parse", "HEAD")
	if revision, err := cmd.Output(); err == nil {
		result.SourceRevision = strings.TrimSpace(string(revision))
	}
	if result.SourceRevision == "" {
		result.SourceRevision = "working-tree:" + result.SourceDigest
	}
	return result, nil
}

func collect(dst *[]Declaration, scopes map[string][]scopecatalog.Scope, manifest, scenario string, parent []string, groups []sourceGroup) error {
	for _, group := range groups {
		path := append([]string(nil), parent...)
		if !group.Flat {
			path = append(path, group.Name)
		}
		for _, command := range group.Commands {
			g := command.Governance
			if command.Name == "" || g.Effect != "read" && g.Effect != "write" && g.Effect != "destructive" || g.RunEligible == nil {
				return fmt.Errorf("manifest %s group %s contains malformed command governance", manifest, strings.Join(path, "/"))
			}
			for _, permission := range g.Permissions {
				switch permission {
				case "filesystem:read", "filesystem:write", "process:spawn", "network:internal", "network:external", "admin":
				default:
					return fmt.Errorf("manifest %s command %s declares unsupported permission %q", manifest, command.Name, permission)
				}
			}
			commandPath := strings.Join(append(append([]string(nil), path...), command.Name), "/")
			key := scenario + "\x00" + commandPath
			permissions := append([]string(nil), g.Permissions...)
			sort.Strings(permissions)
			candidates := scopes[key]
			matched := -1
			for i, candidate := range candidates {
				candidatePermissions := append([]string(nil), candidate.Permissions...)
				sort.Strings(candidatePermissions)
				if candidate.Effect == scopecatalog.Effect(g.Effect) && candidate.RunEligible == *g.RunEligible && equalStrings(candidatePermissions, permissions) {
					matched = i
					break
				}
			}
			if matched < 0 {
				return fmt.Errorf("manifest %s command %s has no derived authorization scope", manifest, commandPath)
			}
			scope := candidates[matched]
			scopes[key] = append(candidates[:matched], candidates[matched+1:]...)
			spawn := false
			for _, permission := range permissions {
				if permission == "process:spawn" {
					spawn = true
				}
			}
			*dst = append(*dst, Declaration{Manifest: filepath.ToSlash(manifest), Scenario: scenario, DeclaringOwner: scenario, Group: strings.Join(path, "/"), Command: commandPath, Effect: g.Effect, RunEligible: *g.RunEligible, Permissions: permissions, ProcessSpawnDeclared: spawn, DerivedScope: scope.Value, DispatchEligible: scope.RunEligible, AuthorizationSource: "local owner handler/OS identity; remote Bridge caller authorization and node command/transport scope"})
		}
		if err := collect(dst, scopes, manifest, scenario, path, group.Groups); err != nil {
			return err
		}
	}
	return nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mustRelative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func (p *Provider) List(ctx context.Context) ([]permissions.Descriptor, error) {
	principal, err := principalFrom(ctx)
	if err != nil {
		return nil, err
	}
	snapshot, err := p.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return []permissions.Descriptor{{Version: permissions.ContractVersion, ID: PermissionID, Owner: Owner, PrincipalKind: string(principal.Kind), PrincipalID: principal.Subject, Scope: PermissionScope, SubjectKind: PermissionSubject, SubjectID: snapshot.Source, Source: snapshot.Source, SchemaVersion: revision(snapshot), CanRead: true, CanWrite: false, Sensitivity: "operator"}}, nil
}

func (p *Provider) AuthorizeRead(ctx context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	principal, err := principalFrom(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	if req.Target != (permissions.TargetBinding{}) || req.Owner != Owner || req.ActorID != principal.Subject || d.ID != PermissionID || d.Owner != Owner || d.PrincipalID != principal.Subject || d.PrincipalKind != string(principal.Kind) || d.Scope != PermissionScope || d.SubjectKind != PermissionSubject || !d.CanRead || d.CanWrite {
		return permissions.Decision{}, connect.NewError(connect.CodePermissionDenied, errors.New("CLI manifest declaration binding mismatch"))
	}
	snapshot, err := p.snapshot(ctx)
	if err != nil {
		return permissions.Decision{}, err
	}
	if d.SchemaVersion != revision(snapshot) {
		return permissions.Decision{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("CLI manifest source changed; refresh permission inventory"))
	}
	return permissions.Decision{Allowed: true, DecisionRef: "cli-manifest:" + snapshot.SourceDigest}, nil
}

func (p *Provider) Read(ctx context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, connect.NewError(connect.CodePermissionDenied, errors.New("CLI manifest read permit mismatch"))
	}
	if _, err := principalFrom(ctx); err != nil {
		return permissions.ReadResult{}, err
	}
	snapshot, err := p.snapshot(ctx)
	if err != nil {
		return permissions.ReadResult{}, err
	}
	rev := revision(snapshot)
	if d.SchemaVersion != rev {
		return permissions.ReadResult{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("CLI manifest source changed; refresh permission inventory"))
	}
	value, err := json.Marshal(snapshot)
	if err != nil {
		return permissions.ReadResult{}, connect.NewError(connect.CodeInternal, errors.New("encode CLI manifest declarations"))
	}
	digest := sha256.Sum256(value)
	return permissions.ReadResult{PermissionID: d.ID, Owner: Owner, Revision: rev, Value: value, ValueDigest: hex.EncodeToString(digest[:]), ObservedAt: time.Now().UTC()}, nil
}

func (*Provider) Authorize(context.Context, permissions.Descriptor, permissions.WriteRequest) (permissions.Decision, error) {
	return permissions.Decision{}, connect.NewError(connect.CodeUnimplemented, errors.New("CLI manifest declarations are source-owned and read-only"))
}
func (*Provider) Write(context.Context, permissions.Descriptor, permissions.Permit, permissions.WriteRequest) (permissions.Receipt, error) {
	return permissions.Receipt{}, connect.NewError(connect.CodeUnimplemented, errors.New("CLI manifest declarations are source-owned and read-only"))
}

func revision(s Snapshot) string { return SchemaID + "#" + s.SourceDigest + "#" + s.SourceRevision }
func principalFrom(ctx context.Context) (identity.Principal, error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok || !p.IsVerified() || strings.TrimSpace(p.Subject) == "" {
		return identity.Principal{}, connect.NewError(connect.CodeUnauthenticated, errors.New("verified onboarding principal required"))
	}
	return p, nil
}
