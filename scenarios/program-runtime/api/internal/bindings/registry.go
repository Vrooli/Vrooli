// Package bindings projects the repository's manifest-bound Connect surface
// into the governed callable namespace used by program-runtime.
package bindings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

var errNoBinding = errors.New("binding does not exist")

// reachabilityTTL bounds how long a scenario URL probe may be reused. It is
// short enough to observe lifecycle changes during a session while keeping a
// fleet-wide binding census from probing the same scenario on every call.
const reachabilityTTL = 30 * time.Second

type methodInfo struct {
	input   protoreflect.MessageDescriptor
	output  protoreflect.MessageDescriptor
	service protoreflect.FullName
	source  string
}

type responseProjection struct {
	rowsField       string
	metaFields      []string
	candidateFields []string
}

// responseProjectionFor resolves the response shape from protobuf descriptors
// and the owning command declaration. JSON object ordering is deliberately not
// part of this contract: an ambiguous response is refused until its owner
// declares which repeated field represents the operation's primary rows.
func responseProjectionFor(output protoreflect.MessageDescriptor, declaredPrimary string) responseProjection {
	repeated := make([]string, 0)
	all := make([]string, 0, output.Fields().Len())
	for i := 0; i < output.Fields().Len(); i++ {
		field := output.Fields().Get(i)
		name := string(field.JSONName())
		all = append(all, name)
		if field.IsList() && !field.IsMap() {
			repeated = append(repeated, name)
		}
	}
	sort.Strings(repeated)
	sort.Strings(all)
	projection := responseProjection{metaFields: append([]string(nil), all...)}
	switch {
	case len(repeated) == 1:
		projection.rowsField = repeated[0]
	case len(repeated) > 1:
		primary := normalizeField(declaredPrimary)
		for _, candidate := range repeated {
			if normalizeField(candidate) == primary {
				projection.rowsField = candidate
				break
			}
		}
		if projection.rowsField == "" {
			projection.candidateFields = repeated
		}
	}
	filtered := projection.metaFields[:0]
	for _, field := range projection.metaFields {
		if field != projection.rowsField {
			filtered = append(filtered, field)
		}
	}
	projection.metaFields = filtered
	return projection
}

// ReachabilityResolver resolves a scenario's advertised API base URL. Keeping
// this as a narrow function seam makes the doctor census deterministic in
// tests while production uses api-core/discovery's cross-platform resolver.
type ReachabilityResolver func(context.Context, string) (string, error)

// Registry is a pinned callable fleet surface. A registry returned by Load is
// backed by a Source and refreshes between requests; each method observes one
// immutable generation. Registries built by LoadFiles remain deterministic
// static fixtures for tests.
type Registry struct {
	dynamic           *registryDynamic
	bindings          []*bindingsv1.Binding
	unbound           []*bindingsv1.UnboundCapability
	skipped           []*bindingsv1.SkippedManifest
	methods           map[string]methodInfo
	required          map[string]map[string]bool
	schemas           map[string]cliapp.ArgSchema
	byID              map[string]*bindingsv1.Binding
	operation         map[string][]*bindingsv1.Binding
	shared            []string
	semantic          map[string]semanticCounts
	recorder          InvocationRecorder
	artifactMtime     time.Time
	generationMtime   time.Time
	manifestMtimes    map[string]time.Time
	manifestPaths     map[string]string
	resolver          ReachabilityResolver
	manifestCount     int
	totalScenarios    int
	reachabilityMu    sync.Mutex
	reachabilityCache map[string]reachabilityStatus
	reachabilityAt    map[string]time.Time
	now               func() time.Time
}

type reachabilityStatus struct {
	reachable bool
	reason    string
}

type registryDynamic struct {
	source         *descriptorimage.Source
	descriptorPath string
	manifestPaths  []string
	mu             sync.Mutex
	current        *Registry
	generation     uint64
	recorder       InvocationRecorder
	resolver       ReachabilityResolver
}

// Load resolves the canonical repository artifacts and builds a registry.
func Load(repoRoot string) (*Registry, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return nil, errors.New("repository root is required")
	}
	descriptorPath := filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb")
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("load repository contract: %w", err)
	}
	targets, err := contract.EnumerateTargets(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("enumerate repository targets: %w", err)
	}
	manifestPaths := make([]string, 0)
	for _, target := range targets {
		if target.Kind != repocontract.TargetKindScenario && target.Kind != repocontract.TargetKindProject {
			continue
		}
		var path string
		if target.Kind == repocontract.TargetKindProject {
			rel, relErr := repocontract.ScenarioCLIManifestRel(repoRoot)
			if relErr != nil {
				return nil, fmt.Errorf("resolve project CLI manifest: %w", relErr)
			}
			path = filepath.Join(repoRoot, filepath.FromSlash(rel))
		} else {
			path, err = repocontract.ScenarioCLIManifestPath(repoRoot, target.ID)
			if err != nil {
				return nil, fmt.Errorf("resolve CLI manifest for %s: %w", target.ID, err)
			}
		}
		manifestPaths = append(manifestPaths, path)
	}
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: descriptorPath, ManifestPaths: manifestPaths})
	if err != nil {
		return nil, err
	}
	return LoadFromSource(source)
}

// LoadFromSource builds a registry over a shared descriptor source. The source
// must watch the descriptor first and any CLI manifests after it.
func LoadFromSource(source *descriptorimage.Source) (*Registry, error) {
	if source == nil {
		return nil, errors.New("descriptor source is required")
	}
	paths := source.WatchedPaths()
	if len(paths) == 0 {
		return nil, errors.New("descriptor source has no watched paths")
	}
	descriptorPath := source.DescriptorPath()
	manifestPaths := make([]string, 0, len(paths)-1)
	for _, path := range paths {
		if path != descriptorPath {
			manifestPaths = append(manifestPaths, path)
		}
	}
	initial, err := source.Snapshot()
	if err != nil {
		return nil, err
	}
	registry, err := buildRegistry(descriptorPath, manifestPaths, initial)
	if err != nil {
		return nil, err
	}
	registry.dynamic = &registryDynamic{source: source, descriptorPath: descriptorPath, manifestPaths: append([]string(nil), manifestPaths...), current: registry, generation: initial.Generation}
	return registry, nil
}

// LoadWithRetry bounds startup retries around descriptor publication. Once a
// registry exists, refresh failures are fail-safe and keep the prior surface.
func LoadWithRetry(repoRoot string, attempts int, delay time.Duration) (*Registry, error) {
	if attempts < 1 {
		attempts = 1
	}
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		var registry *Registry
		registry, err = Load(repoRoot)
		if err == nil {
			return registry, nil
		}
		if attempt+1 < attempts && delay > 0 {
			time.Sleep(delay)
		}
	}
	return nil, err
}

// LoadFiles builds a registry from an explicit descriptor and manifest set.
// It is the deterministic seam used by unit tests and fixture-based checks.
func LoadFiles(descriptorPath string, manifestPaths []string) (*Registry, error) {
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: descriptorPath, ManifestPaths: manifestPaths})
	if err != nil {
		return nil, err
	}
	snapshot, err := source.Snapshot()
	if err != nil {
		return nil, err
	}
	return buildRegistry(descriptorPath, manifestPaths, snapshot)
}

func buildRegistry(descriptorPath string, manifestPaths []string, snapshot *descriptorimage.Snapshot) (*Registry, error) {
	if snapshot == nil || snapshot.Files == nil {
		return nil, errors.New("descriptor snapshot is unavailable")
	}
	files := snapshot.Files
	methods := make(map[string]methodInfo)
	serviceMethods := make(map[string][]methodInfo)
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			for j := 0; j < svc.Methods().Len(); j++ {
				m := svc.Methods().Get(j)
				info := methodInfo{input: m.Input(), output: m.Output(), service: svc.FullName(), source: fd.Path()}
				full := string(svc.FullName()) + "." + string(m.Name())
				short := string(svc.Name()) + "." + string(m.Name())
				methods[full] = info
				serviceMethods[full] = append(serviceMethods[full], info)
				serviceMethods[short] = append(serviceMethods[short], info)
			}
		}
		return true
	})

	r := &Registry{
		methods:           methods,
		required:          make(map[string]map[string]bool),
		schemas:           make(map[string]cliapp.ArgSchema),
		byID:              make(map[string]*bindingsv1.Binding),
		operation:         make(map[string][]*bindingsv1.Binding),
		semantic:          make(map[string]semanticCounts),
		resolver:          ResolveTargetURLDefault,
		totalScenarios:    len(manifestPaths),
		reachabilityCache: make(map[string]reachabilityStatus),
		reachabilityAt:    make(map[string]time.Time),
		now:               time.Now,
		manifestMtimes:    make(map[string]time.Time),
		manifestPaths:     make(map[string]string),
	}
	if info, statErr := os.Stat(descriptorPath); statErr == nil {
		r.generationMtime = info.ModTime()
	}
	for _, path := range manifestPaths {
		if info, statErr := os.Stat(path); statErr == nil && !info.IsDir() {
			r.manifestCount++
			scenario := manifestScenarioPath(descriptorPath, path)
			r.manifestMtimes[scenario] = info.ModTime()
			repoRoot := filepath.Clean(filepath.Join(filepath.Dir(descriptorPath), "../../../.."))
			if relative, relErr := filepath.Rel(repoRoot, path); relErr == nil {
				r.manifestPaths[scenario] = filepath.ToSlash(relative)
			} else {
				r.manifestPaths[scenario] = filepath.ToSlash(path)
			}
		}
	}
	r.artifactMtime = snapshot.ArtifactMTime
	r.shared = sharedContractPrefixes(filepath.Clean(filepath.Join(filepath.Dir(descriptorPath), "../../../..")))
	for _, path := range manifestPaths {
		skipped, err := r.addManifest(path, serviceMethods, r.shared)
		if err != nil {
			return nil, err
		}
		if skipped != nil {
			r.skipped = append(r.skipped, skipped)
			r.addUnbound(&bindingsv1.UnboundCapability{
				Scenario: skipped.GetScenario(),
				Reason:   bindingsv1.UnboundReason_UNBOUND_REASON_MALFORMED_MANIFEST,
				Detail:   skipped.GetParseError(),
			})
		}
	}
	for _, path := range manifestPaths {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			name := manifestScenarioPath(descriptorPath, path)
			r.addUnbound(&bindingsv1.UnboundCapability{Scenario: name, Reason: bindingsv1.UnboundReason_UNBOUND_REASON_NO_MANIFEST, Detail: "scenario has no CLI manifest"})
		}
	}
	sort.Slice(r.bindings, func(i, j int) bool { return r.bindings[i].GetId() < r.bindings[j].GetId() })
	sort.Slice(r.unbound, func(i, j int) bool {
		if r.unbound[i].GetScenario() != r.unbound[j].GetScenario() {
			return r.unbound[i].GetScenario() < r.unbound[j].GetScenario()
		}
		return r.unbound[i].GetCommand() < r.unbound[j].GetCommand()
	})
	return r, nil
}

// ResolveTargetURLDefault resolves scenario targets through the normal
// discovery contract and resolves the repository project through its stable
// VROOLI_API_PORT contract. The project is not a scenario runtime, so asking
// `vrooli scenario port vrooli API_PORT` would be both misleading and
// platform-dependent. The environment override is injected by project
// lifecycle setup; 8092 is the service manifest's documented fallback.
func ResolveTargetURLDefault(ctx context.Context, target string) (string, error) {
	if target != "vrooli" {
		return discovery.ResolveScenarioURLDefault(ctx, target)
	}
	port := strings.TrimSpace(os.Getenv("VROOLI_API_PORT"))
	if port == "" {
		port = "8092"
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return "", fmt.Errorf("invalid VROOLI_API_PORT %q", port)
	}
	host := strings.TrimSpace(os.Getenv("VROOLI_API_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, number), nil
}

func (r *Registry) active() *Registry {
	if r == nil || r.dynamic == nil {
		return r
	}
	return r.dynamic.refresh()
}

func (d *registryDynamic) refresh() *Registry {
	d.mu.Lock()
	defer d.mu.Unlock()
	snapshot, err := d.source.Snapshot()
	if err != nil {
		return d.current
	}
	if d.current != nil && d.generation == snapshot.Generation {
		return d.current
	}
	next, err := buildRegistry(d.descriptorPath, d.manifestPaths, snapshot)
	if err != nil {
		return d.current
	}
	next.recorder = d.recorder
	if d.resolver != nil {
		next.resolver = d.resolver
	}
	d.current = next
	d.generation = snapshot.Generation
	return next
}

// SnapshotMetadata exposes the source state for health surfaces without
// forcing callers to know how the registry is refreshed.
func (r *Registry) SnapshotMetadata() (digest string, generation uint64, loadedAt, artifactMTime time.Time, reloadErr error) {
	if r == nil || r.dynamic == nil {
		return "", 0, time.Time{}, r.artifactMtime, nil
	}
	snapshot, err := r.dynamic.source.Snapshot()
	if snapshot != nil {
		digest, generation, loadedAt, artifactMTime = snapshot.Digest, snapshot.Generation, snapshot.LoadedAt, snapshot.ArtifactMTime
	}
	if err != nil {
		reloadErr = err
	}
	if sourceErr := r.dynamic.source.LastReloadError(); sourceErr != nil {
		reloadErr = sourceErr
	}
	return
}

// Execute performs one governed outbound Connect JSON call. The bridge is
// intentionally behind this method so Python cannot bypass the manifest-bound
// descriptor and governance checks. Process isolation is enforced by the
// kernel supervisor; this registry is the typed network policy boundary.
type InvocationMetadata struct {
	SessionID  string
	ProgramID  string
	Provenance string
}

func (r *Registry) SetInvocationRecorder(recorder InvocationRecorder) {
	if r.dynamic != nil {
		r.dynamic.mu.Lock()
		r.dynamic.recorder = recorder
		if r.dynamic.current != nil {
			r.dynamic.current.recorder = recorder
		}
		r.dynamic.mu.Unlock()
		return
	}
	r.recorder = recorder
}

func (r *Registry) RecordInvocation(ctx context.Context, invocation Invocation) {
	r = r.active()
	if r.recorder != nil {
		if invocation.TargetScenario == "" {
			if binding := r.byID[invocation.BindingID]; binding != nil {
				invocation.TargetScenario = binding.GetScenario()
			}
		}
		_ = r.recorder.RecordInvocation(ctx, invocation)
	}
}

func (r *Registry) Execute(ctx context.Context, id string, args map[string]any, grants []string, confirmed bool, metadata InvocationMetadata, client *http.Client) (result map[string]any, err error) {
	r = r.active()
	started := time.Now()
	defer func() {
		if r.recorder == nil {
			return
		}
		binding := r.byID[id]
		target := ""
		if binding != nil {
			target = binding.GetScenario()
		}
		outcome := "success"
		reason := ""
		if err != nil {
			outcome = "failed"
			reason = err.Error()
			if strings.Contains(reason, "requires an explicit grant") || strings.Contains(reason, "requires explicit confirmation") || strings.Contains(reason, "invalid arguments") || strings.Contains(reason, "missing required field") {
				outcome = "refused"
			}
		}
		usageInput, usageOutput, usageCost := invocationUsage(result)
		invocation := Invocation{BindingID: id, TargetScenario: target, SessionID: metadata.SessionID, ProgramID: metadata.ProgramID, Provenance: metadata.Provenance, Outcome: outcome, Reason: reason, LatencyMS: time.Since(started).Milliseconds(), UsageInputTokens: usageInput, UsageOutputTokens: usageOutput, UsageCostMicros: usageCost, OccurredAt: time.Now().UTC()}
		invocation.Origin = invocationOrigin(invocation)
		invocation.InvocationClass = classifyInvocation(invocation)
		r.RecordInvocation(ctx, invocation)
	}()
	binding, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("binding %q is not governed", id)
	}
	if err := r.Authorize(id, grants, confirmed); err != nil {
		return nil, err
	}
	if err := r.ValidateArguments(id, args); err != nil {
		return nil, err
	}
	statuses, _ := r.reachability(ctx, binding.GetScenario())
	if status := statuses[binding.GetScenario()]; !status.reachable {
		return nil, fmt.Errorf("binding %s is unreachable: %s", id, status.reason)
	}
	info, ok := r.methods[id]
	if !ok {
		return nil, fmt.Errorf("binding %q has no descriptor method", id)
	}
	resolver := r.resolver
	if resolver == nil {
		resolver = discovery.ResolveScenarioURLDefault
	}
	base, err := resolver(ctx, binding.GetScenario())
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", binding.GetScenario(), err)
	}
	canonical, err := r.canonicalArguments(id, args)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("encode binding arguments: %w", err)
	}
	request := dynamicpb.NewMessage(info.input)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, request); err != nil {
		return nil, fmt.Errorf("decode binding arguments: %w", err)
	}
	body, err := protojson.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal binding request: %w", err)
	}
	url := strings.TrimRight(base, "/") + "/" + string(info.service) + "/" + binding.GetMethod()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		r.invalidateReachability(binding.GetScenario())
		return nil, fmt.Errorf("invoke %s: %w", id, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", id, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("invoke %s: remote status %s: %s", id, resp.Status, strings.TrimSpace(string(responseBody)))
	}
	response := dynamicpb.NewMessage(info.output)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(responseBody, response); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", id, err)
	}
	jsonResponse, err := protojson.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("encode %s response: %w", id, err)
	}
	var jsonResult map[string]any
	if err := json.Unmarshal(jsonResponse, &jsonResult); err != nil {
		return nil, fmt.Errorf("decode %s response object: %w", id, err)
	}
	return jsonResult, nil
}

func invocationUsage(result map[string]any) (input, output, cost int64) {
	usage, ok := result["usage"].(map[string]any)
	if !ok {
		return 0, 0, 0
	}
	return numberToInt(usage["input_tokens"]), numberToInt(usage["output_tokens"]), numberToInt(usage["cost_micros"])
}

func numberToInt(value any) int64 {
	switch n := value.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	case json.Number:
		value, _ := n.Int64()
		return value
	case string:
		value, _ := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		return value
	}
	return 0
}

func (r *Registry) addManifest(path string, serviceMethods map[string][]methodInfo, sharedPrefixes []string) (*bindingsv1.SkippedManifest, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return &bindingsv1.SkippedManifest{Path: path, Scenario: manifestScenario(path, nil), ParseError: fmt.Sprintf("read CLI manifest: %v", err)}, nil
	}
	m, err := cliapp.ParseManifest(data)
	if err != nil {
		return &bindingsv1.SkippedManifest{Path: path, Scenario: manifestScenario(path, data), ParseError: fmt.Sprintf("parse CLI manifest: %v", err)}, nil
	}
	scenario := m.Name
	for _, group := range m.Groups {
		for _, command := range group.Commands {
			b := command.Binding
			if b.Kind == "local" {
				r.addUnbound(&bindingsv1.UnboundCapability{Scenario: scenario, Group: group.Name, Command: command.Name, Reason: bindingsv1.UnboundReason_UNBOUND_REASON_LOCAL_BINDING, Detail: "manifest declares a local binding"})
				continue
			}
			if !command.Governance.RunEligible {
				r.addUnbound(&bindingsv1.UnboundCapability{Scenario: scenario, Group: group.Name, Command: command.Name, Service: b.Service, Method: b.Method, Reason: bindingsv1.UnboundReason_UNBOUND_REASON_EXTERNAL_TOOL_ONLY, Detail: "manifest governance declares run_eligible=false"})
				continue
			}
			info, err := resolveMethod(scenario, b.Service+"."+b.Method, serviceMethods, sharedPrefixes)
			if err != nil {
				r.addUnbound(&bindingsv1.UnboundCapability{Scenario: scenario, Group: group.Name, Command: command.Name, Service: b.Service, Method: b.Method, Reason: bindingsv1.UnboundReason_UNBOUND_REASON_OMITTED_RPC, Detail: "manifest binding does not resolve in the descriptor image"})
				continue
			}
			id := scenario + "/" + group.Name + "/" + command.Name
			requiresConfirmation := command.Governance.RequiresConfirmation != nil && *command.Governance.RequiresConfirmation
			if command.Governance.RequiresConfirmation == nil && command.Governance.Effect == "destructive" {
				requiresConfirmation = true
			}
			binding := &bindingsv1.Binding{
				Id: id, Scenario: scenario, Group: group.Name, Command: command.Name,
				Service: b.Service, Method: b.Method, RequestType: string(info.input.FullName()), ResponseType: string(info.output.FullName()),
				Effect: command.Governance.Effect, RunEligible: true, RequiresConfirmation: requiresConfirmation,
				Permissions: append([]string(nil), command.Governance.Permissions...), Description: command.Description,
				Signature: fmt.Sprintf("%s.%s(%s) -> %s", b.Service, b.Method, info.input.FullName(), info.output.FullName()),
			}
			projection := responseProjectionFor(info.output, "")
			if command.Architecture != nil {
				projection = responseProjectionFor(info.output, command.Architecture.PrimaryField)
			}
			binding.RowsField = projection.rowsField
			binding.MetaFields = append([]string(nil), projection.metaFields...)
			binding.RowFieldCandidates = append([]string(nil), projection.candidateFields...)
			r.bindings = append(r.bindings, binding)
			r.byID[id] = binding
			r.methods[id] = info
			req := make(map[string]bool)
			for _, p := range command.Positionals {
				if p.Required && !p.LocalOnly {
					req[p.Name] = true
				}
			}
			for _, f := range command.Flags {
				if f.Required && !f.LocalOnly {
					req[f.Name] = true
				}
			}
			r.required[id] = req
			schema, err := cliapp.ManifestArgs(command)
			if err != nil {
				return nil, fmt.Errorf("manifest %s command %s/%s: %w", path, group.Name, command.Name, err)
			}
			r.schemas[id] = schema
			r.addSemanticCounts(scenario, command, info.input, schema)
			for _, key := range []string{normalizeField(command.Name), normalizeField(group.Name + "/" + command.Name), normalizeField(b.Service + "." + b.Method), normalizeField(id)} {
				r.operation[key] = append(r.operation[key], binding)
			}
		}
	}
	for _, omitted := range m.Omitted {
		r.addUnbound(&bindingsv1.UnboundCapability{Scenario: scenario, Service: omitted.Service, Method: omitted.Method, Reason: bindingsv1.UnboundReason_UNBOUND_REASON_OMITTED_RPC, Detail: omitted.Reason})
	}
	for _, exception := range m.Exceptions {
		r.addUnbound(&bindingsv1.UnboundCapability{Scenario: scenario, Command: exception.Command, Reason: bindingsv1.UnboundReason_UNBOUND_REASON_EXTERNAL_TOOL_ONLY, Detail: exception.Reason})
	}
	return nil, nil
}

// manifestScenarioPath provides the stable owner name for freshness metadata
// and missing-manifest diagnostics. Scenario manifests follow the conventional
// scenarios/<name>/cli path; the project manifest is deliberately different
// and is governed as the control-plane target "vrooli".
func manifestScenarioPath(descriptorPath, manifestPath string) string {
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(descriptorPath), "../../../.."))
	if rel, err := repocontract.ScenarioCLIManifestRel(repoRoot); err == nil {
		if filepath.Clean(manifestPath) == filepath.Join(repoRoot, filepath.FromSlash(rel)) {
			return "vrooli"
		}
	}
	clean := filepath.Clean(manifestPath)
	return filepath.Base(filepath.Dir(filepath.Dir(clean)))
}

func manifestScenario(path string, data []byte) string {
	var header struct {
		Name string `json:"name"`
	}
	if len(data) > 0 && json.Unmarshal(data, &header) == nil && strings.TrimSpace(header.Name) != "" {
		return header.Name
	}
	clean := filepath.Clean(path)
	return filepath.Base(filepath.Dir(filepath.Dir(clean)))
}

type sharedContractDeclaration struct {
	Contracts []struct {
		Prefix string `json:"prefix"`
	} `json:"contracts"`
}

func sharedContractPrefixes(repoRoot string) []string {
	data, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "schemas", "shared-proto-contracts.json"))
	if err != nil {
		return nil
	}
	var declaration sharedContractDeclaration
	if json.Unmarshal(data, &declaration) != nil {
		return nil
	}
	out := make([]string, 0, len(declaration.Contracts))
	for _, contract := range declaration.Contracts {
		if strings.TrimSpace(contract.Prefix) != "" {
			prefix := strings.TrimSuffix(filepath.ToSlash(contract.Prefix), "/") + "/"
			out = append(out, prefix, "schemas/"+prefix, "packages/proto/schemas/"+prefix)
		}
	}
	return out
}

func resolveMethod(scenario, key string, methods map[string][]methodInfo, sharedPrefixes []string) (methodInfo, error) {
	candidates := methods[key]
	var own, shared []methodInfo
	for _, candidate := range candidates {
		// The repository project owns the root control-plane manifest, while
		// its cli/v1 contracts intentionally live in the shared proto package.
		// Keep this exception narrow to the project target and this contract;
		// scenario manifests still require an explicit shared-proto declaration.
		if scenario == "vrooli" && strings.HasPrefix(filepath.ToSlash(candidate.source), "cli/v1/") {
			shared = append(shared, candidate)
			continue
		}
		if strings.HasPrefix(filepath.ToSlash(candidate.source), scenario+"/") {
			own = append(own, candidate)
			continue
		}
		for _, prefix := range sharedPrefixes {
			if strings.HasPrefix(filepath.ToSlash(candidate.source), prefix) {
				shared = append(shared, candidate)
				break
			}
		}
	}
	if len(own) == 1 {
		return own[0], nil
	}
	if len(own) > 1 {
		return methodInfo{}, fmt.Errorf("%s: ambiguous own-package service %q", scenario, key)
	}
	if len(shared) == 1 {
		return shared[0], nil
	}
	if len(shared) > 1 {
		return methodInfo{}, fmt.Errorf("%s: ambiguous shared service %q", scenario, key)
	}
	return methodInfo{}, fmt.Errorf("%s: service %q is not declared in its own package or shared contracts", scenario, key)
}

func (r *Registry) addUnbound(c *bindingsv1.UnboundCapability) { r.unbound = append(r.unbound, c) }

func normalizeField(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "-", "_"))
}

func (r *Registry) List(scenario, group string) []*bindingsv1.Binding {
	return r.ListContext(context.Background(), scenario, group, false)
}

// ListContext returns governed bindings annotated with the cached reachability
// result from the same discovery probe used by DoctorContext.
func (r *Registry) ListContext(ctx context.Context, scenario, group string, reachableOnly bool) []*bindingsv1.Binding {
	r = r.active()
	statuses, _ := r.reachability(ctx, scenario)
	out := make([]*bindingsv1.Binding, 0)
	for _, b := range r.bindings {
		if scenario != "" && b.GetScenario() != scenario {
			continue
		}
		if group != "" && b.GetGroup() != group {
			continue
		}
		status := statuses[b.GetScenario()]
		if reachableOnly && !status.reachable {
			continue
		}
		clone := proto.Clone(b).(*bindingsv1.Binding)
		clone.Reachable = status.reachable
		clone.ReachabilityReason = status.reason
		out = append(out, clone)
	}
	return out
}

func (r *Registry) ReachabilityCheckedAt(ctx context.Context, scenario string) time.Time {
	r = r.active()
	_, checkedAt := r.reachability(ctx, scenario)
	return checkedAt
}

// ReachabilitySnapshot is the live status used by the kernel's reachability
// surface. The cache is shared by every binding owned by the scenario.
type ReachabilitySnapshot struct {
	Reachable bool   `json:"reachable"`
	Reason    string `json:"reason"`
	CheckedAt string `json:"checked_at"`
}

func (r *Registry) ReachabilitySnapshot(ctx context.Context) map[string]ReachabilitySnapshot {
	r = r.active()
	statuses, _ := r.reachability(ctx, "")
	r.reachabilityMu.Lock()
	defer r.reachabilityMu.Unlock()
	out := make(map[string]ReachabilitySnapshot, len(statuses))
	for name, status := range statuses {
		out[name] = ReachabilitySnapshot{Reachable: status.reachable, Reason: status.reason, CheckedAt: r.reachabilityAt[name].UTC().Format(time.RFC3339Nano)}
	}
	return out
}

// SetClock is a deterministic test seam for TTL behavior. Production always
// uses time.Now; callers should set it only before concurrent use.
func (r *Registry) SetClock(clock func() time.Time) {
	r.reachabilityMu.Lock()
	defer r.reachabilityMu.Unlock()
	if clock == nil {
		r.now = time.Now
		return
	}
	r.now = clock
}

func (r *Registry) Unbound(scenario string) []*bindingsv1.UnboundCapability {
	r = r.active()
	out := make([]*bindingsv1.UnboundCapability, 0)
	for _, c := range r.unbound {
		if scenario == "" || c.GetScenario() == scenario {
			out = append(out, proto.Clone(c).(*bindingsv1.UnboundCapability))
		}
	}
	return out
}

func (r *Registry) Binding(id string) (*bindingsv1.Binding, bool) {
	r = r.active()
	b, ok := r.byID[id]
	if !ok {
		return nil, false
	}
	return proto.Clone(b).(*bindingsv1.Binding), true
}

func (r *Registry) IsInferenceBinding(id string) bool {
	r = r.active()
	binding, ok := r.byID[id]
	if !ok {
		return false
	}
	return strings.EqualFold(binding.GetScenario(), "ai-gateway") || strings.Contains(strings.ToLower(binding.GetService()), "inference")
}

// InferenceUsage extracts the canonical ai-gateway Usage projection from a
// protojson result without coupling the bridge to a provider implementation.
func InferenceUsage(result map[string]any) (input, output, cost int64, present bool) {
	usage, ok := result["usage"].(map[string]any)
	if !ok || usage == nil {
		return 0, 0, 0, false
	}
	input = numberToInt(usage["input_tokens"])
	if input == 0 {
		input = numberToInt(usage["inputTokens"])
	}
	output = numberToInt(usage["output_tokens"])
	if output == 0 {
		output = numberToInt(usage["outputTokens"])
	}
	cost = numberToInt(usage["cost_micros"])
	if cost == 0 {
		cost = numberToInt(usage["costMicros"])
	}
	return input, output, cost, true
}

// SetReachabilityResolver replaces the discovery seam used by Doctor. It is
// intended for deterministic tests and controlled embedding; nil restores
// the production api-core/discovery resolver.
func (r *Registry) SetReachabilityResolver(resolver ReachabilityResolver) {
	if r.dynamic != nil {
		r.dynamic.mu.Lock()
		r.dynamic.resolver = resolver
		if r.dynamic.current != nil {
			r.dynamic.current.resolver = resolver
		}
		r.dynamic.mu.Unlock()
		return
	}
	if resolver == nil {
		r.resolver = ResolveTargetURLDefault
		return
	}
	r.resolver = resolver
}

// Doctor returns the fleet callability census and every argument that still
// cannot be projected onto its request descriptor. It intentionally derives
// its results from the same resolver used by Execute.
func (r *Registry) Doctor(scenario string) *bindingsv1.DoctorBindingsResponse {
	r = r.active()
	return r.DoctorContext(context.Background(), scenario)
}

// DoctorContext returns the fleet callability census and distinguishes
// governed scenarios that can currently resolve an API URL from scenarios
// whose bindings exist but whose target is unreachable. It also reports the
// manifest-bearing scenario count, which is the measured ceiling for this
// binding-backed surface.
func (r *Registry) DoctorContext(ctx context.Context, scenario string) *bindingsv1.DoctorBindingsResponse {
	r = r.active()
	response := &bindingsv1.DoctorBindingsResponse{}
	response.ManifestScenarios = int32(r.manifestCount)
	response.TotalScenarios = int32(r.totalScenarios)
	for _, binding := range r.bindings {
		if scenario != "" && binding.GetScenario() != scenario {
			continue
		}
		response.Bindings++
		analysis := r.analyze(binding.GetId())
		if analysis.zeroArg {
			response.ZeroArg++
		}
		if analysis.callable {
			response.Callable++
		} else if analysis.requiredBroken {
			response.Uncallable++
		} else {
			response.Partial++
		}
		if !analysis.owned {
			response.Misroutes++
		}
		response.Issues = append(response.Issues, analysis.issues...)
	}
	for name, counts := range r.semantic {
		if scenario != "" && scenario != name {
			continue
		}
		response.FieldCollisions += int32(counts.fieldCollisions)
		response.ControlFlagsBound += int32(counts.controlFlagsBound)
		response.RequiredFieldsUnpopulated += int32(counts.requiredFieldsUnpopulated)
		response.BindsWhereRenameSuffices += int32(counts.bindsWhereRenameSuffices)
		response.ScalarBoundToMessage += int32(counts.scalarBoundToMessage)
	}
	for _, skipped := range r.skipped {
		if scenario == "" || skipped.GetScenario() == scenario {
			response.SkippedManifests = append(response.SkippedManifests, proto.Clone(skipped).(*bindingsv1.SkippedManifest))
		}
	}
	response.SkippedManifestCount = int32(len(response.SkippedManifests))
	r.populateReachability(ctx, scenario, response)
	return response
}

func (r *Registry) populateReachability(ctx context.Context, scenario string, response *bindingsv1.DoctorBindingsResponse) {
	names := make(map[string]struct{})
	for _, binding := range r.bindings {
		if scenario == "" || binding.GetScenario() == scenario {
			names[binding.GetScenario()] = struct{}{}
		}
	}
	if len(names) == 0 {
		return
	}
	statuses, checkedAt := r.reachability(ctx, scenario)
	response.ReachabilityCheckedAt = checkedAt.UTC().Format(time.RFC3339Nano)
	response.ReachabilityCheckedAtByScenario = make(map[string]string, len(statuses))
	r.reachabilityMu.Lock()
	for name := range statuses {
		if at := r.reachabilityAt[name]; !at.IsZero() {
			response.ReachabilityCheckedAtByScenario[name] = at.UTC().Format(time.RFC3339Nano)
		}
	}
	r.reachabilityMu.Unlock()
	for name := range names {
		item := statuses[name]
		if item.reachable {
			response.ReachableScenarios = append(response.ReachableScenarios, name)
		} else {
			response.UnreachableScenarios = append(response.UnreachableScenarios, name)
		}
	}
	sort.Strings(response.ReachableScenarios)
	sort.Strings(response.UnreachableScenarios)
}

// reachability performs one concurrent discovery probe per scenario and
// reuses each result only for reachabilityTTL. A caller asking for one
// scenario still shares the same cache with fleet-wide doctor calls.
func (r *Registry) reachability(ctx context.Context, scenario string) (map[string]reachabilityStatus, time.Time) {
	if r.resolver == nil {
		r.resolver = discovery.ResolveScenarioURLDefault
	}
	names := make(map[string]struct{})
	for _, binding := range r.bindings {
		if scenario == "" || binding.GetScenario() == scenario {
			names[binding.GetScenario()] = struct{}{}
		}
	}
	r.reachabilityMu.Lock()
	missing := make([]string, 0, len(names))
	now := r.now()
	for name := range names {
		checkedAt := r.reachabilityAt[name]
		if _, ok := r.reachabilityCache[name]; !ok || checkedAt.IsZero() || now.Sub(checkedAt) >= reachabilityTTL {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		cached := cloneReachability(r.reachabilityCache)
		checkedAt := latestReachabilityAt(r.reachabilityAt, names)
		r.reachabilityMu.Unlock()
		return cached, checkedAt
	}
	r.reachabilityMu.Unlock()

	type result struct {
		name   string
		status reachabilityStatus
	}
	results := make(chan result, len(missing))
	for _, name := range missing {
		name := name
		go func() {
			probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			_, err := r.resolver(probeCtx, name)
			status := reachabilityStatus{reachable: err == nil, reason: "scenario API resolved"}
			if err != nil {
				status.reason = err.Error()
			}
			results <- result{name: name, status: status}
		}()
	}
	r.reachabilityMu.Lock()
	for range missing {
		item := <-results
		r.reachabilityCache[item.name] = item.status
		r.reachabilityAt[item.name] = now
	}
	cached := cloneReachability(r.reachabilityCache)
	checkedAt := latestReachabilityAt(r.reachabilityAt, names)
	r.reachabilityMu.Unlock()
	return cached, checkedAt
}

func latestReachabilityAt(times map[string]time.Time, names map[string]struct{}) time.Time {
	var latest time.Time
	for name := range names {
		if times[name].After(latest) {
			latest = times[name]
		}
	}
	return latest
}

func (r *Registry) invalidateReachability(scenario string) {
	r.reachabilityMu.Lock()
	delete(r.reachabilityCache, scenario)
	delete(r.reachabilityAt, scenario)
	r.reachabilityMu.Unlock()
}

func cloneReachability(source map[string]reachabilityStatus) map[string]reachabilityStatus {
	clone := make(map[string]reachabilityStatus, len(source))
	for name, status := range source {
		clone[name] = status
	}
	return clone
}

// Describe returns the resolved request path for every manifest argument.
func (r *Registry) Describe(id string) (*bindingsv1.DescribeBindingResponse, error) {
	r = r.active()
	binding, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errNoBinding, id)
	}
	info, ok := r.methods[id]
	if !ok {
		return nil, fmt.Errorf("binding %q has no descriptor method", id)
	}
	schema := r.schemas[id]
	response := &bindingsv1.DescribeBindingResponse{
		Binding:        proto.Clone(binding).(*bindingsv1.Binding),
		ResolvedSource: info.source,
		Callable:       true,
	}
	appendArgument := func(name string, required bool) {
		argument := &bindingsv1.BindingArgument{Name: name, Required: required}
		resolved, err := cliapp.ResolveArgField(info.input, name, schema)
		if err != nil {
			argument.Reason = err.Error()
			response.Callable = false
		} else {
			argument.ProtoPath = resolvedPath(name, resolved.Path)
			argument.Kind = resolved.Kind
		}
		response.Arguments = append(response.Arguments, argument)
	}
	appendLocalArgument := func(name string, required bool) {
		response.Arguments = append(response.Arguments, &bindingsv1.BindingArgument{
			Name: name, Required: required, Reason: "local-only CLI control",
		})
	}
	for _, positional := range schema.Positionals {
		if positional.LocalOnly {
			appendLocalArgument(positional.Name, positional.Required)
			continue
		}
		appendArgument(positional.Name, positional.Required)
	}
	for _, flag := range schema.Flags {
		if flag.LocalOnly {
			appendLocalArgument(flag.Name, flag.Required)
			continue
		}
		appendArgument(flag.Name, flag.Required)
	}
	return response, nil
}

type bindingAnalysis struct {
	callable       bool
	requiredBroken bool
	zeroArg        bool
	owned          bool
	issues         []*bindingsv1.BindingIssue
}

func (r *Registry) analyze(id string) bindingAnalysis {
	binding := r.byID[id]
	info := r.methods[id]
	schema := r.schemas[id]
	nonLocalArgs := 0
	for _, positional := range schema.Positionals {
		if !positional.LocalOnly {
			nonLocalArgs++
		}
	}
	for _, flag := range schema.Flags {
		if !flag.LocalOnly {
			nonLocalArgs++
		}
	}
	analysis := bindingAnalysis{callable: true, zeroArg: nonLocalArgs == 0, owned: r.sourceBelongsTo(binding.GetScenario(), info.source)}
	required := r.required[id]
	check := func(name string) {
		_, err := cliapp.ResolveArgField(info.input, name, schema)
		if err == nil {
			return
		}
		analysis.callable = false
		if required[name] {
			analysis.requiredBroken = true
		}
		analysis.issues = append(analysis.issues, &bindingsv1.BindingIssue{
			Scenario: binding.GetScenario(), BindingId: id, Argument: name,
			RequestType: string(info.input.FullName()), Reason: err.Error(),
			CandidateFields: descriptorFieldNames(info.input),
		})
	}
	for _, positional := range schema.Positionals {
		if positional.LocalOnly {
			continue
		}
		check(positional.Name)
	}
	for _, flag := range schema.Flags {
		if flag.LocalOnly {
			continue
		}
		check(flag.Name)
	}
	return analysis
}

func descriptorFieldNames(md protoreflect.MessageDescriptor) []string {
	return cliapp.DescriptorFieldNames(md)
}

func resolvedPath(argumentName string, path []protoreflect.FieldDescriptor) string {
	parts := make([]string, 0, len(path))
	for _, field := range path {
		parts = append(parts, string(field.JSONName()))
	}
	if len(parts) == 2 {
		return argumentName + " -> " + strings.Join(parts, ".")
	}
	return strings.Join(parts, ".")
}

func (r *Registry) sourceBelongsTo(scenario, source string) bool {
	path := filepath.ToSlash(source)
	if scenario == "vrooli" && strings.HasPrefix(path, "cli/v1/") {
		return true
	}
	for _, prefix := range []string{
		scenario + "/",
		"schemas/" + scenario + "/",
		"packages/proto/schemas/" + scenario + "/",
		"scenarios/" + scenario + "/",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, prefix := range r.shared {
		if strings.HasPrefix(path, filepath.ToSlash(prefix)) {
			return true
		}
	}
	return false
}

// Authorize is the last governance check before dispatch. The registry owns
// the vocabulary (effect and permissions come from the manifest); a session
// only supplies grants and an explicit confirmation for destructive work.
func (r *Registry) Authorize(id string, grants []string, confirmed bool) error {
	r = r.active()
	b, ok := r.byID[id]
	if !ok {
		return fmt.Errorf("%w: %s", errNoBinding, id)
	}
	grantSet := make(map[string]struct{}, len(grants))
	for _, grant := range grants {
		grantSet[strings.TrimSpace(grant)] = struct{}{}
	}
	if b.GetEffect() == "destructive" {
		matched := false
		for _, permission := range b.GetPermissions() {
			if _, ok := grantSet[permission]; ok {
				matched = true
				break
			}
		}
		if _, ok := grantSet["effect:destructive"]; ok {
			matched = true
		}
		if !matched {
			return fmt.Errorf("destructive binding %q requires an explicit grant", id)
		}
	}
	if b.GetRequiresConfirmation() && !confirmed {
		return fmt.Errorf("binding %q requires explicit confirmation", id)
	}
	return nil
}

// ValidateArguments performs descriptor-backed JSON validation before a
// Connect request is issued. Unknown fields and type errors name their field
// through protojson's canonical diagnostic. Manifest-required arguments are
// checked in addition to protobuf required fields.
func (r *Registry) ValidateArguments(id string, args map[string]any) error {
	r = r.active()
	info, ok := r.methods[id]
	if !ok {
		return fmt.Errorf("%w: %s", errNoBinding, id)
	}
	if args == nil {
		args = map[string]any{}
	}
	canonical, err := r.canonicalArguments(id, args)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return fmt.Errorf("encode arguments: %w", err)
	}
	msg := dynamicpb.NewMessage(info.input)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, msg); err != nil {
		return fmt.Errorf("invalid arguments for %s: %w", id, err)
	}
	for field := range r.required[id] {
		resolved, resolveErr := cliapp.ResolveArgField(info.input, field, r.schemas[id])
		if resolveErr != nil || len(resolved.Path) == 0 || !hasResolvedField(msg, resolved.Path) {
			return fmt.Errorf("missing required field %q", field)
		}
	}
	for i := 0; i < info.input.Fields().Len(); i++ {
		fd := info.input.Fields().Get(i)
		if fd.Cardinality() == protoreflect.Required && !msg.Has(fd) {
			return fmt.Errorf("missing required field %q", fd.Name())
		}
	}
	return nil
}

func hasResolvedField(root protoreflect.Message, path []protoreflect.FieldDescriptor) bool {
	current := root
	for _, field := range path {
		if !current.Has(field) {
			return false
		}
		if field.IsList() {
			return current.Get(field).List().Len() > 0
		}
		if field.IsMap() {
			return current.Get(field).Map().Len() > 0
		}
		if field.Kind() == protoreflect.MessageKind {
			current = current.Get(field).Message()
		}
	}
	return true
}

func (r *Registry) canonicalArguments(id string, args map[string]any) (map[string]any, error) {
	info, ok := r.methods[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errNoBinding, id)
	}
	if args == nil {
		args = map[string]any{}
	}
	schema := r.schemas[id]
	out := make(map[string]any, len(args))
	for name, value := range args {
		if schemaArgLocalOnly(schema, name) {
			continue
		}
		resolved, err := cliapp.ResolveArgField(info.input, name, schema)
		if err != nil {
			var resolutionErr *cliapp.ArgFieldResolutionError
			if errors.As(err, &resolutionErr) {
				return nil, err
			}
			return nil, fmt.Errorf("argument %q: %w", name, err)
		}
		if len(resolved.Path) == 0 {
			return nil, fmt.Errorf("argument %q has an empty proto path", name)
		}
		putResolvedArgument(out, resolved.Path, value)
	}
	return out, nil
}

func schemaArgLocalOnly(schema cliapp.ArgSchema, name string) bool {
	for _, positional := range schema.Positionals {
		if positional.LocalOnly && positional.Name == name {
			return true
		}
	}
	for _, flag := range schema.Flags {
		if !flag.LocalOnly && !containsString(flag.Aliases, name) {
			continue
		}
		if flag.Name == name || containsString(flag.Aliases, name) {
			return flag.LocalOnly
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func putResolvedArgument(out map[string]any, path []protoreflect.FieldDescriptor, value any) {
	current := out
	for _, field := range path[:len(path)-1] {
		key := field.JSONName()
		nested, ok := current[key].(map[string]any)
		if !ok {
			nested = make(map[string]any)
			current[key] = nested
		}
		current = nested
	}
	current[path[len(path)-1].JSONName()] = value
}

// ResolveOperation accepts canonical binding IDs and the concise operation
// names used by the Act denominator. Ambiguous names intentionally resolve to
// all candidates; a cell is covered only when at least one governed binding
// exists for every operation it names.
func (r *Registry) ResolveOperation(operation string) []*bindingsv1.Binding {
	r = r.active()
	key := normalizeField(strings.Trim(operation, "`"))
	if found := r.operation[key]; len(found) > 0 {
		return found
	}
	parts := strings.FieldsFunc(key, func(r rune) bool { return r == '/' || r == '.' || r == ':' || r == ' ' })
	if len(parts) == 0 {
		return nil
	}
	out := make([]*bindingsv1.Binding, 0)
	for _, b := range r.bindings {
		candidate := normalizeField(b.GetId() + " " + b.GetService() + " " + b.GetMethod())
		matched := true
		for _, part := range parts {
			if !strings.Contains(candidate, part) {
				matched = false
				break
			}
		}
		if matched {
			out = append(out, b)
		}
	}
	return out
}

// ResolveByIntent is the in-kernel discovery seam. A future search-hub
// adapter can replace the local index without changing the program contract;
// direct binding calls remain available when search is unavailable.
func (r *Registry) ResolveByIntent(intent string) ([]*bindingsv1.Binding, string) {
	r = r.active()
	terms := strings.Fields(normalizeField(intent))
	if len(terms) == 0 {
		return nil, "intent is empty"
	}
	out := make([]*bindingsv1.Binding, 0)
	for _, binding := range r.bindings {
		text := normalizeField(strings.Join([]string{binding.GetId(), binding.GetDescription(), binding.GetEffect()}, " "))
		matched := true
		for _, term := range terms {
			if !strings.Contains(text, term) {
				matched = false
				break
			}
		}
		if matched {
			out = append(out, proto.Clone(binding).(*bindingsv1.Binding))
		}
	}
	if len(out) == 0 {
		return nil, "search-hub unavailable or no governed binding matched the intent"
	}
	return out, "local registry fallback"
}

// ResolveActCells applies the strict all-operations join rule.
func (r *Registry) ResolveActCells(ctx context.Context, cells []*bindingsv1.ActCell) []*bindingsv1.ActCellVerdict {
	r = r.active()
	_ = ctx
	out := make([]*bindingsv1.ActCellVerdict, 0, len(cells))
	for _, cell := range cells {
		v := &bindingsv1.ActCellVerdict{Id: cell.GetId(), AuthoredStatus: cell.GetAuthoredStatus(), Audited: true}
		for _, op := range cell.GetOperations() {
			resolved, complete, reasons := r.resolveActOperation(op)
			if resolved {
				v.ResolvedOperations = append(v.ResolvedOperations, op)
			}
			if !complete {
				v.UnresolvedOperations = append(v.UnresolvedOperations, op)
				v.Reasons = append(v.Reasons, reasons...)
			}
		}
		switch {
		case len(v.ResolvedOperations) == len(cell.GetOperations()) && len(v.UnresolvedOperations) == 0 && len(v.ResolvedOperations) > 0:
			v.Verdict = bindingsv1.ActVerdict_ACT_VERDICT_NOW
		case len(v.ResolvedOperations) > 0:
			v.Verdict = bindingsv1.ActVerdict_ACT_VERDICT_IN_REACH
		case strings.EqualFold(v.GetAuthoredStatus(), "NOW"), strings.EqualFold(v.GetAuthoredStatus(), "COVERED"):
			v.Verdict = bindingsv1.ActVerdict_ACT_VERDICT_IN_REACH
		default:
			v.Verdict = bindingsv1.ActVerdict_ACT_VERDICT_AUTHORED
		}
		out = append(out, v)
	}
	return out
}

// Sweep executes only safe, reachable, read-effect bindings with no required
// manifest arguments. It is intentionally operator-provenance evidence: the
// sweep exercises the same registry Execute path while remaining absent from
// the default agent friction corpus.
func (r *Registry) Sweep(ctx context.Context, scenario, effect string, dryRun bool) *bindingsv1.SweepBindingsResponse {
	r = r.active()
	if strings.TrimSpace(effect) == "" {
		effect = "read"
	}
	statuses, _ := r.reachability(ctx, scenario)
	response := &bindingsv1.SweepBindingsResponse{Provenance: "PROVENANCE_OPERATOR"}
	type pendingSweep struct {
		result *bindingsv1.SweepBindingResult
		id     string
		args   map[string]any
	}
	pending := make([]pendingSweep, 0)
	for _, binding := range r.bindings {
		result := &bindingsv1.SweepBindingResult{BindingId: binding.GetId(), Scenario: binding.GetScenario()}
		if scenario != "" && binding.GetScenario() != scenario {
			result.SkippedReason = "scenario filter excluded"
			response.Skipped++
			response.Results = append(response.Results, result)
			continue
		}
		if !strings.EqualFold(binding.GetEffect(), effect) {
			result.SkippedReason = "not read-effect"
			response.Skipped++
			response.Results = append(response.Results, result)
			continue
		}
		status := statuses[binding.GetScenario()]
		if !status.reachable {
			result.SkippedReason = "not reachable: " + status.reason
			response.Skipped++
			response.Results = append(response.Results, result)
			continue
		}
		analysis := r.analyze(binding.GetId())
		if !analysis.callable {
			result.SkippedReason = "unpopulatable required field"
			if len(analysis.issues) > 0 {
				result.SkippedReason += ": " + analysis.issues[0].GetArgument()
			}
			response.Skipped++
			response.Results = append(response.Results, result)
			continue
		}
		args, err := r.sweepArguments(binding.GetId())
		if err != nil {
			result.SkippedReason = "unpopulatable required field: " + err.Error()
			response.Skipped++
			response.Results = append(response.Results, result)
			continue
		}
		response.Eligible++
		if dryRun {
			response.Results = append(response.Results, result)
			continue
		}
		pending = append(pending, pendingSweep{result: result, id: binding.GetId(), args: args})
	}
	if !dryRun {
		workers := 16
		semaphore := make(chan struct{}, workers)
		outcomes := make(chan pendingSweep, len(pending))
		var wait sync.WaitGroup
		for _, item := range pending {
			item := item
			wait.Add(1)
			go func() {
				defer wait.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				started := time.Now()
				_, err := r.Execute(ctx, item.id, item.args, nil, false, InvocationMetadata{ProgramID: "sweep", Provenance: "PROVENANCE_OPERATOR"}, &http.Client{Timeout: 15 * time.Second})
				item.result.Attempted = true
				item.result.LatencyMs = time.Since(started).Milliseconds()
				if err != nil {
					item.result.Outcome = "failed"
					item.result.Reason = err.Error()
				} else {
					item.result.Outcome = "success"
				}
				outcomes <- item
			}()
		}
		wait.Wait()
		close(outcomes)
		for item := range outcomes {
			response.Attempted++
			if item.result.Outcome == "failed" {
				response.Failed++
			} else {
				response.Succeeded++
			}
			response.Results = append(response.Results, item.result)
		}
	}
	return response
}

func (r *Registry) sweepArguments(id string) (map[string]any, error) {
	args := make(map[string]any)
	for name := range r.required[id] {
		resolved, err := cliapp.ResolveArgField(r.methods[id].input, name, r.schemas[id])
		if err != nil || len(resolved.Path) == 0 {
			return nil, fmt.Errorf("%s", name)
		}
		field := resolved.Path[len(resolved.Path)-1]
		value, ok := defaultSweepValue(field, name)
		if !ok {
			return nil, fmt.Errorf("%s", name)
		}
		args[name] = value
	}
	return args, nil
}

func defaultSweepValue(field protoreflect.FieldDescriptor, name string) (any, bool) {
	if field.IsList() {
		return []any{}, true
	}
	if field.IsMap() {
		return map[string]any{}, true
	}
	switch field.Kind() {
	case protoreflect.BoolKind:
		return false, true
	case protoreflect.StringKind:
		if strings.Contains(strings.ToLower(name), "scenario") {
			return "program-runtime", true
		}
		return "sweep-probe", true
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sint64Kind, protoreflect.Int64Kind:
		return int64(1), true
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return uint64(1), true
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return float64(1), true
	case protoreflect.BytesKind:
		return "", true
	case protoreflect.EnumKind:
		values := field.Enum().Values()
		if values.Len() == 0 {
			return nil, false
		}
		return string(values.Get(0).Name()), true
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return map[string]any{}, true
	default:
		return nil, false
	}
}

// resolveActOperation bridges the denominator's human-readable owner field
// to the registry's mechanical namespace. A cell can name several owners;
// every named owner must have a governed binding for a NOW verdict. A partial
// owner set is IN_REACH, while an entirely non-callable set preserves the
// authored status. Every outcome is still audited because unresolved and
// external-only owners carry explicit reasons rather than remaining unknown.
func (r *Registry) resolveActOperation(operation string) (resolved, complete bool, reasons []string) {
	if matches := r.ResolveOperation(operation); len(matches) > 0 {
		return true, true, nil
	}
	tokens := actOwnerTokens(operation)
	if len(tokens) == 0 {
		return false, false, []string{fmt.Sprintf("%s: owner is not a manifest-backed scenario", operation)}
	}
	resolvedCount := 0
	for _, token := range tokens {
		if matches := r.ResolveOperation(token); len(matches) > 0 {
			resolvedCount++
			continue
		}
		matches := r.scenarioBindings(token)
		if len(matches) > 0 {
			resolvedCount++
			continue
		}
		if r.scenarioKnown(token) {
			reasons = append(reasons, fmt.Sprintf("%s: scenario has no governed binding", token))
		} else {
			reasons = append(reasons, fmt.Sprintf("%s: owner is external or unresolved", token))
		}
	}
	return resolvedCount > 0, resolvedCount == len(tokens), reasons
}

func actOwnerTokens(owner string) []string {
	owner = strings.ToLower(strings.TrimSpace(owner))
	if owner == "" || owner == "—" || owner == "-" {
		return nil
	}
	var out []string
	for _, part := range strings.FieldsFunc(owner, func(r rune) bool { return r == ',' || r == '+' || r == ';' }) {
		part = strings.TrimSpace(strings.Trim(part, "`()"))
		if part == "" || strings.Contains(part, "project cli") || strings.Contains(part, "per-scenario") || strings.Contains(part, "api-core") || part == "fleet" {
			continue
		}
		if strings.HasSuffix(part, " fleet") {
			part = strings.TrimSpace(strings.TrimSuffix(part, " fleet"))
		}
		if strings.Contains(part, " ") {
			continue
		}
		out = append(out, part)
	}
	return out
}

func (r *Registry) scenarioBindings(scenario string) []*bindingsv1.Binding {
	if strings.HasPrefix(scenario, "*-") {
		suffix := strings.TrimPrefix(scenario, "*")
		var out []*bindingsv1.Binding
		for _, binding := range r.bindings {
			if strings.HasSuffix(binding.GetScenario(), suffix) {
				out = append(out, binding)
			}
		}
		return out
	}
	var out []*bindingsv1.Binding
	for _, binding := range r.bindings {
		if binding.GetScenario() == scenario {
			out = append(out, binding)
		}
	}
	return out
}

func (r *Registry) scenarioKnown(scenario string) bool {
	if strings.HasPrefix(scenario, "*-") {
		suffix := strings.TrimPrefix(scenario, "*")
		for _, binding := range r.bindings {
			if strings.HasSuffix(binding.GetScenario(), suffix) {
				return true
			}
		}
		for _, unbound := range r.unbound {
			if strings.HasSuffix(unbound.GetScenario(), suffix) {
				return true
			}
		}
		return false
	}
	for _, binding := range r.bindings {
		if binding.GetScenario() == scenario {
			return true
		}
	}
	for _, unbound := range r.unbound {
		if unbound.GetScenario() == scenario {
			return true
		}
	}
	return false
}

// ActConfidence is derived from the audit result, not from the authored
// denominator. A fully audited grid is measured but remains partial while it
// contains explicit external/unbound cells; an incomplete grid is sketch.
func ActConfidence(verdicts []*bindingsv1.ActCellVerdict) string {
	if len(verdicts) == 0 {
		return "SKETCH"
	}
	for _, verdict := range verdicts {
		if verdict == nil || !verdict.GetAudited() {
			return "SKETCH"
		}
	}
	return "PARTIAL"
}

// Count returns snapshot sizes for health and measures surfaces.
func (r *Registry) Count() (bound, unbound int) {
	r = r.active()
	return len(r.bindings), len(r.unbound)
}

// SkippedManifestCount reports manifests that were isolated during the
// immutable registry load. It is intentionally separate from Unbound so
// operators can distinguish malformed fleet input from a valid, intentionally
// unbound command.
func (r *Registry) SkippedManifestCount() int {
	r = r.active()
	return len(r.skipped)
}

// Ensure the imported registry package remains used even when a build omits
// the descriptor path in a platform-specific test.
var _ protoregistry.Files
