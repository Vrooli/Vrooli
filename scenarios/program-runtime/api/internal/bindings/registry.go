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
	"strings"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/repo-contract-go"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

var errNoBinding = errors.New("binding does not exist")

type methodInfo struct {
	input   protoreflect.MessageDescriptor
	output  protoreflect.MessageDescriptor
	service protoreflect.FullName
}

// Registry is an immutable snapshot of the callable fleet surface. It is
// loaded once at API boot; callers receive cloned protobuf messages.
type Registry struct {
	bindings  []*bindingsv1.Binding
	unbound   []*bindingsv1.UnboundCapability
	methods   map[string]methodInfo
	required  map[string]map[string]bool
	byID      map[string]*bindingsv1.Binding
	operation map[string][]*bindingsv1.Binding
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
		if target.Kind != repocontract.TargetKindScenario {
			continue
		}
		path, err := repocontract.ScenarioCLIManifestPath(repoRoot, target.ID)
		if err != nil {
			return nil, fmt.Errorf("resolve CLI manifest for %s: %w", target.ID, err)
		}
		manifestPaths = append(manifestPaths, path)
	}
	return LoadFiles(descriptorPath, manifestPaths)
}

// LoadFiles builds a registry from an explicit descriptor and manifest set.
// It is the deterministic seam used by unit tests and fixture-based checks.
func LoadFiles(descriptorPath string, manifestPaths []string) (*Registry, error) {
	data, err := os.ReadFile(descriptorPath)
	if err != nil {
		return nil, fmt.Errorf("read descriptor image: %w", err)
	}
	var set descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(data, &set); err != nil {
		return nil, fmt.Errorf("decode descriptor image: %w", err)
	}
	files, err := protodesc.NewFiles(&set)
	if err != nil {
		return nil, fmt.Errorf("load descriptor image: %w", err)
	}
	methods := make(map[string]methodInfo)
	serviceMethods := make(map[string]methodInfo)
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			for j := 0; j < svc.Methods().Len(); j++ {
				m := svc.Methods().Get(j)
				info := methodInfo{input: m.Input(), output: m.Output(), service: svc.FullName()}
				full := string(svc.FullName()) + "." + string(m.Name())
				short := string(svc.Name()) + "." + string(m.Name())
				methods[full] = info
				serviceMethods[short] = info
			}
		}
		return true
	})

	r := &Registry{
		methods:   methods,
		required:  make(map[string]map[string]bool),
		byID:      make(map[string]*bindingsv1.Binding),
		operation: make(map[string][]*bindingsv1.Binding),
	}
	for _, path := range manifestPaths {
		if err := r.addManifest(path, serviceMethods); err != nil {
			return nil, err
		}
	}
	for _, path := range manifestPaths {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			name := filepath.Base(filepath.Dir(filepath.Dir(path)))
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

// Execute performs one governed outbound Connect JSON call. The bridge is
// intentionally behind this method so Python never gets a direct network or
// filesystem capability: it can only ask the Go registry to validate,
// authorize, and dispatch a manifest-backed method.
func (r *Registry) Execute(ctx context.Context, id string, args map[string]any, grants []string, confirmed bool, client *http.Client) (map[string]any, error) {
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
	info, ok := r.methods[id]
	if !ok {
		return nil, fmt.Errorf("binding %q has no descriptor method", id)
	}
	base, err := discovery.ResolveScenarioURLDefault(ctx, binding.GetScenario())
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", binding.GetScenario(), err)
	}
	raw, err := json.Marshal(args)
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
	var result map[string]any
	if err := json.Unmarshal(jsonResponse, &result); err != nil {
		return nil, fmt.Errorf("decode %s response object: %w", id, err)
	}
	return result, nil
}

func (r *Registry) addManifest(path string, serviceMethods map[string]methodInfo) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read CLI manifest %s: %w", path, err)
	}
	m, err := cliapp.ParseManifest(data)
	if err != nil {
		return fmt.Errorf("parse CLI manifest %s: %w", path, err)
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
			info, ok := serviceMethods[b.Service+"."+b.Method]
			if !ok {
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
			r.bindings = append(r.bindings, binding)
			r.byID[id] = binding
			r.methods[id] = info
			req := make(map[string]bool)
			for _, p := range command.Positionals {
				if p.Required {
					req[normalizeField(p.Name)] = true
				}
			}
			for _, f := range command.Flags {
				if f.Required {
					req[normalizeField(f.Name)] = true
				}
			}
			r.required[id] = req
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
	return nil
}

func (r *Registry) addUnbound(c *bindingsv1.UnboundCapability) { r.unbound = append(r.unbound, c) }

func normalizeField(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "-", "_"))
}

func (r *Registry) List(scenario, group string) []*bindingsv1.Binding {
	out := make([]*bindingsv1.Binding, 0)
	for _, b := range r.bindings {
		if scenario != "" && b.GetScenario() != scenario {
			continue
		}
		if group != "" && b.GetGroup() != group {
			continue
		}
		out = append(out, proto.Clone(b).(*bindingsv1.Binding))
	}
	return out
}

func (r *Registry) Unbound(scenario string) []*bindingsv1.UnboundCapability {
	out := make([]*bindingsv1.UnboundCapability, 0)
	for _, c := range r.unbound {
		if scenario == "" || c.GetScenario() == scenario {
			out = append(out, proto.Clone(c).(*bindingsv1.UnboundCapability))
		}
	}
	return out
}

func (r *Registry) Binding(id string) (*bindingsv1.Binding, bool) {
	b, ok := r.byID[id]
	if !ok {
		return nil, false
	}
	return proto.Clone(b).(*bindingsv1.Binding), true
}

// Authorize is the last governance check before dispatch. The registry owns
// the vocabulary (effect and permissions come from the manifest); a session
// only supplies grants and an explicit confirmation for destructive work.
func (r *Registry) Authorize(id string, grants []string, confirmed bool) error {
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
	info, ok := r.methods[id]
	if !ok {
		return fmt.Errorf("%w: %s", errNoBinding, id)
	}
	if args == nil {
		args = map[string]any{}
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("encode arguments: %w", err)
	}
	msg := dynamicpb.NewMessage(info.input)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, msg); err != nil {
		return fmt.Errorf("invalid arguments for %s: %w", id, err)
	}
	for field := range r.required[id] {
		fd := msg.Descriptor().Fields().ByName(protoreflect.Name(field))
		if fd == nil || !msg.Has(fd) {
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

// ResolveOperation accepts canonical binding IDs and the concise operation
// names used by the Act denominator. Ambiguous names intentionally resolve to
// all candidates; a cell is covered only when at least one governed binding
// exists for every operation it names.
func (r *Registry) ResolveOperation(operation string) []*bindingsv1.Binding {
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
		default:
			v.Verdict = bindingsv1.ActVerdict_ACT_VERDICT_AUTHORED
		}
		out = append(out, v)
	}
	return out
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
func (r *Registry) Count() (bound, unbound int) { return len(r.bindings), len(r.unbound) }

// Ensure the imported registry package remains used even when a build omits
// the descriptor path in a platform-specific test.
var _ protoregistry.Files
