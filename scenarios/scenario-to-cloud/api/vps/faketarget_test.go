package vps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/cloudrelease"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// fakeTarget is one host running the native `vrooli cloud-target` owner. It
// answers every verb the executor dispatches with the owner's JSON shapes,
// keeps the durable target state (fence, receipts, staged releases, active
// pointer, activation intent, persistent data, demand on shared resources,
// edge routes) and records the order of effects so tests can prove the
// activation ordering and the coexistence rules.
type fakeTarget struct {
	mu         sync.Mutex
	platform   string
	calls      []reach.Command
	delivered  []reach.ArtifactFile
	manifests  map[string]string // remote manifest path -> release id
	fence      uint64
	receipts   map[string]string // "op/step" -> stored JSON reply
	staged     map[string]bool
	active     *fakeActive
	intent     *fakeIntent
	running    map[string]bool
	resources  map[string]bool
	demand     map[string]map[string]bool // resource -> scenarios demanding it
	wants      map[string]map[string]bool // scenario -> resources it depends on
	routes     map[string]int
	routeOwner map[string]string // host -> deployment that owns the snippet
	// legacy is the in-place workdir tree: "<scenario>/<rel>" -> files.
	legacy map[string]map[string]string
	// persistent is the persistent-data root: binding id -> files.
	persistent map[string]map[string]string
	bound      map[string]string // "<scenario>/<rel>" -> binding id
	unmapped   []string
	crashAt    string
	failNext   string
	events     []string
	scenario   string // demand attribution for resource starts
	backups    int
}

type fakeActive struct {
	Active   string `json:"active_release"`
	Previous string `json:"previous_release,omitempty"`
	Strategy string `json:"strategy"`
}

type fakeIntent struct {
	Candidate string `json:"candidate"`
	Previous  string `json:"previous,omitempty"`
}

func newFakeTarget() *fakeTarget {
	return &fakeTarget{
		platform: "linux/amd64", manifests: map[string]string{}, receipts: map[string]string{}, staged: map[string]bool{},
		running: map[string]bool{}, resources: map[string]bool{}, demand: map[string]map[string]bool{}, wants: map[string]map[string]bool{}, routes: map[string]int{}, routeOwner: map[string]string{},
		legacy: map[string]map[string]string{}, persistent: map[string]map[string]string{}, bound: map[string]string{},
	}
}

var _ reach.Reach = (*fakeTarget)(nil)

func flagsOf(args []string) (map[string][]string, []string) {
	flags := map[string][]string{}
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			name := strings.TrimPrefix(arg, "--")
			if name == "json" || name == "restart" {
				flags[name] = append(flags[name], "true")
				continue
			}
			if i+1 < len(args) {
				flags[name] = append(flags[name], args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, arg)
	}
	return flags, positional
}

func first(flags map[string][]string, name string) string {
	if v := flags[name]; len(v) > 0 {
		return v[0]
	}
	return ""
}

func reply(value any) reach.Result {
	raw, _ := json.Marshal(value)
	return reach.Result{ExitCode: 0, Stdout: string(raw), Transport: identity.TransportSSH}
}

func refusal(code, message string) reach.Result {
	raw, _ := json.Marshal(map[string]any{"error": map[string]any{"code": code, "message": message}})
	return reach.Result{ExitCode: 2, Stdout: string(raw), Transport: identity.TransportSSH}
}

func (t *fakeTarget) Negotiate(context.Context, identity.TargetRef) (reach.Capabilities, error) {
	return reach.Capabilities{Transport: identity.TransportSSH, Online: true, NativeCLI: true, Platform: t.platform, ProtocolVersion: "cloud-target/v1"}, nil
}

func (t *fakeTarget) Deliver(_ context.Context, _ identity.TargetRef, delivery reach.Delivery) (reach.DeliveryReceipt, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := reach.ValidateDelivery(delivery); err != nil {
		return reach.DeliveryReceipt{}, err
	}
	receipt := reach.DeliveryReceipt{Transport: identity.TransportSSH}
	for _, f := range delivery.Files {
		raw, err := os.ReadFile(f.LocalPath)
		if err != nil {
			return receipt, &reach.Error{Kind: reach.KindInvalidArgument, Detail: err.Error()}
		}
		if f.Role == "release_manifest" {
			var m cloudrelease.Manifest
			if err := json.Unmarshal(raw, &m); err == nil {
				t.manifests[f.RemotePath] = m.ReleaseDigest
			}
		}
		sum := sha256.Sum256(raw)
		t.delivered = append(t.delivered, f)
		receipt.Files = append(receipt.Files, reach.DeliveredFile{Role: f.Role, RemotePath: f.RemotePath, SHA256: hex.EncodeToString(sum[:])})
	}
	t.events = append(t.events, "deliver")
	return receipt, nil
}

// effect wraps a receipted verb: fence check, replay per (operation, step).
func (t *fakeTarget) effect(flags map[string][]string, body func() (map[string]any, string, string)) reach.Result {
	fence, _ := strconv.ParseUint(first(flags, "fence"), 10, 64)
	if fence < t.fence {
		return refusal("fence_stale", fmt.Sprintf("fence %d below accepted %d", fence, t.fence))
	}
	t.fence = fence
	key := first(flags, "operation") + "/" + first(flags, "step")
	if stored, ok := t.receipts[key]; ok {
		var value map[string]any
		_ = json.Unmarshal([]byte(stored), &value)
		value["replayed"] = true
		return reply(value)
	}
	details, outcome, code := body()
	value := map[string]any{"receipt": map[string]any{"outcome": outcome, "details": details}, "replayed": false}
	if code != "" {
		value["error"] = map[string]any{"code": code, "message": code}
	}
	raw, _ := json.Marshal(value)
	t.receipts[key] = string(raw)
	if code != "" {
		return reach.Result{ExitCode: 2, Stdout: string(raw)}
	}
	return reply(value)
}

func (t *fakeTarget) Exec(_ context.Context, _ identity.TargetRef, cmd reach.Command) (reach.Result, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := reach.ValidateCommand(cmd); err != nil {
		return reach.Result{}, err
	}
	t.calls = append(t.calls, cmd)
	flags, positional := flagsOf(cmd.Args)
	switch cmd.Verb {
	case "cloud-target host repair":
		return t.effect(flags, func() (map[string]any, string, string) {
			action := first(flags, "action")
			var subject map[string]map[string]any
			raw, _ := DecodeJSONArg(first(flags, "subject"))
			_ = json.Unmarshal(raw, &subject)
			switch action {
			case "process.stop.scoped":
				scenario, _ := subject["process"]["scenario"].(string)
				t.stopScenario(scenario)
			case "apt.packages.ensure", "edge.ufw.allow", "docker.prune.unused-images", "docker.prune.unused-volumes":
			default:
				return map[string]any{"action": action}, "failed", "action_not_allowed"
			}
			t.events = append(t.events, "repair:"+action)
			return map[string]any{"action": action}, "succeeded", ""
		}), nil
	case "cloud-target data inventory":
		scenario := first(flags, "scenario")
		var bindings []map[string]string
		raw, _ := DecodeJSONArg(first(flags, "bindings"))
		_ = json.Unmarshal(raw, &bindings)
		entries := []map[string]any{}
		uncovered := 0
		for key := range t.legacy {
			scen, rel, _ := strings.Cut(key, "/")
			if scen != scenario {
				continue
			}
			entry := map[string]any{"path": rel, "covered": false}
			for _, b := range bindings {
				if rel == b["path"] || strings.HasPrefix(rel, b["path"]+"/") {
					entry["covered"], entry["binding_id"] = true, b["id"]
				}
			}
			if entry["covered"] == false {
				uncovered++
			}
			entries = append(entries, entry)
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i]["path"].(string) < entries[j]["path"].(string) })
		return reply(map[string]any{"deployment_id": first(flags, "deployment"), "scenario": scenario, "entries": entries, "uncovered_count": uncovered}), nil
	case "cloud-target release verify":
		id, ok := t.manifests[first(flags, "release-manifest")]
		if !ok {
			return refusal("release_manifest_invalid", "no manifest delivered at "+first(flags, "release-manifest")), nil
		}
		return reply(map[string]any{"report": map[string]any{"verified": true, "release_digest": id}}), nil
	case "cloud-target release stage":
		return t.effect(flags, func() (map[string]any, string, string) {
			id, ok := t.manifests[first(flags, "release-manifest")]
			if !ok {
				return nil, "failed", "release_manifest_invalid"
			}
			if t.staged[id] {
				return map[string]any{"release": id}, "unchanged", ""
			}
			t.staged[id] = true
			t.events = append(t.events, "stage:"+id)
			return map[string]any{"release": id}, "succeeded", ""
		}), nil
	case "cloud-target data backup":
		res := t.effect(flags, func() (map[string]any, string, string) {
			t.backups++
			t.events = append(t.events, "backup")
			return map[string]any{"bindings": len(flags["binding"])}, "succeeded", ""
		})
		if res.ExitCode == 0 {
			var value map[string]any
			_ = json.Unmarshal([]byte(res.Stdout), &value)
			id := first(flags, "operation") + "-" + first(flags, "step")
			var bindings []map[string]any
			for _, b := range flags["binding"] {
				raw, _ := DecodeJSONArg(b)
				var one map[string]any
				_ = json.Unmarshal(raw, &one)
				bindings = append(bindings, one)
			}
			value["recovery_point"] = map[string]any{"format_version": 1, "id": id, "deployment_id": first(flags, "deployment"), "bindings": bindings, "refs": map[string]any{"schema_version": first(flags, "schema-version"), "configuration_digest": first(flags, "configuration-digest")}, "captured_at": time.Now().UTC().Format(time.RFC3339Nano), "encrypted": true, "key_ref": first(flags, "key-ref"), "migration_posture": first(flags, "migration-posture"), "checksums": map[string]any{}}
			return reply(value), nil
		}
		return res, nil
	case "cloud-target release activate":
		return t.effect(flags, func() (map[string]any, string, string) { return t.activate(flags) }), nil
	case "cloud-target release list":
		return reply(t.listing(first(flags, "deployment"))), nil
	case "cloud-target edge route-apply":
		return t.effect(flags, func() (map[string]any, string, string) {
			raw, _ := DecodeJSONArg(first(flags, "spec"))
			var spec struct {
				Domain string `json:"domain"`
				Routes []struct {
					Host         string `json:"host"`
					UpstreamPort int    `json:"upstream_port"`
				} `json:"routes"`
			}
			if err := json.Unmarshal(raw, &spec); err != nil || len(spec.Routes) == 0 {
				return nil, "failed", "edge_spec_invalid"
			}
			for _, route := range spec.Routes {
				if route.UpstreamPort == 22 || route.UpstreamPort == 18767 {
					return nil, "failed", "edge_private_listener"
				}
				t.routes[route.Host] = route.UpstreamPort
				t.routeOwner[route.Host] = first(flags, "deployment")
				t.events = append(t.events, fmt.Sprintf("route:%s:%d", route.Host, route.UpstreamPort))
			}
			return map[string]any{"domain": spec.Domain}, "succeeded", ""
		}), nil
	case "cloud-target edge route-rollback":
		return t.effect(flags, func() (map[string]any, string, string) {
			owner := first(flags, "deployment")
			removed := 0
			for host, dep := range t.routeOwner {
				if dep == owner {
					delete(t.routes, host)
					delete(t.routeOwner, host)
					removed++
				}
			}
			if removed == 0 {
				return nil, "failed", "edge_rollback_not_eligible"
			}
			t.events = append(t.events, "route:removed:"+owner)
			return map[string]any{"removed": removed}, "succeeded", ""
		}), nil
	case "setup":
		t.events = append(t.events, "setup")
		return reach.Result{ExitCode: 0, Stdout: "{}"}, nil
	case "resource start":
		name := positional[0]
		t.resources[name] = true
		if t.demand[name] == nil {
			t.demand[name] = map[string]bool{}
		}
		if t.scenario != "" {
			t.demand[name][t.scenario] = true
			if t.wants[t.scenario] == nil {
				t.wants[t.scenario] = map[string]bool{}
			}
			t.wants[t.scenario][name] = true
		}
		t.events = append(t.events, "resource:start:"+name)
		return reach.Result{ExitCode: 0, Stdout: "{}"}, nil
	case "resource stop":
		name := positional[0]
		if len(t.demand[name]) > 0 {
			return reach.Result{ExitCode: 1, Stdout: "{}", Stderr: "resource still demanded"}, nil
		}
		t.resources[name] = false
		t.events = append(t.events, "resource:stop:"+name)
		return reach.Result{ExitCode: 0, Stdout: "{}"}, nil
	case "scenario start":
		t.running[positional[0]] = true
		t.events = append(t.events, "scenario:start:"+positional[0])
		return reach.Result{ExitCode: 0, Stdout: "{}"}, nil
	case "stop":
		t.running = map[string]bool{}
		return reach.Result{ExitCode: 0, Stdout: "{}"}, nil
	case "cloud-target receipt get":
		key := first(flags, "operation") + "/" + first(flags, "step")
		if stored, ok := t.receipts[key]; ok {
			var value map[string]any
			_ = json.Unmarshal([]byte(stored), &value)
			receipt, _ := value["receipt"].(map[string]any)
			receipt["schema_version"] = 1
			receipt["fence"] = t.fence
			return reply(receipt), nil
		}
		return refusal("receipt_not_found", key), nil
	}
	return reach.Result{}, &reach.Error{Kind: reach.KindInvalidArgument, Detail: "fake target has no verb " + cmd.Verb}
}

func (t *fakeTarget) stopScenario(scenario string) {
	delete(t.running, scenario)
	for resource := range t.demand {
		delete(t.demand[resource], scenario)
	}
	t.events = append(t.events, "stop:"+scenario)
}

func (t *fakeTarget) activate(flags map[string][]string) (map[string]any, string, string) {
	id := first(flags, "release")
	if !t.staged[id] {
		return map[string]any{"release": id}, "failed", "release_not_staged"
	}
	strategy := first(flags, "strategy")
	restart := len(flags["restart"]) > 0
	previous := ""
	if t.active != nil {
		previous = t.active.Active
		if t.active.Active == id && !restart {
			return map[string]any{"release": id, "previous_release": t.active.Previous}, "unchanged", ""
		}
		if t.active.Active == id {
			previous = t.active.Previous
		}
	}
	t.intent = &fakeIntent{Candidate: id, Previous: previous}
	if t.crashAt == "activate:before_runtime" {
		t.crashAt = ""
		return map[string]any{"release": id}, "failed", "activation_failed"
	}
	// Persistent data binding: adopt legacy content by move, never delete.
	scenarios := flags["scenario"]
	bound := map[string]bool{}
	for _, spec := range flags["data-binding"] {
		bid, location, _ := strings.Cut(spec, "=")
		if t.persistent[bid] == nil {
			if files, ok := t.legacy[location]; ok {
				t.persistent[bid] = files
				delete(t.legacy, location)
			} else {
				t.persistent[bid] = map[string]string{}
			}
		}
		t.bound[location] = bid
		bound[location] = true
	}
	for _, carry := range flags["legacy-carry"] {
		bid := "legacy:" + carry
		if t.persistent[bid] == nil {
			if files, ok := t.legacy[carry]; ok {
				t.persistent[bid] = files
				delete(t.legacy, carry)
			} else {
				t.persistent[bid] = map[string]string{}
			}
		}
		t.bound[carry] = bid
		bound[carry] = true
	}
	var unmapped []string
	for location := range t.legacy {
		scen, _, _ := strings.Cut(location, "/")
		for _, s := range scenarios {
			if s == scen && !bound[location] {
				unmapped = append(unmapped, "legacy:"+location)
			}
		}
	}
	sort.Strings(unmapped)
	t.unmapped = unmapped
	if t.failNext == id {
		t.failNext = ""
		t.intent = nil
		return map[string]any{"release": id, "prior_active_release": previous}, "failed", "activation_failed"
	}
	if strategy == "maintenance" {
		for _, s := range scenarios {
			t.stopScenario(s)
		}
	}
	for _, s := range scenarios {
		t.running[s] = true
		// A started scenario re-acquires demand on the resources it depends on
		// (the lifecycle owner's dependency demand leases).
		for resource := range t.wants[s] {
			if t.demand[resource] == nil {
				t.demand[resource] = map[string]bool{}
			}
			t.demand[resource][s] = true
		}
		for _, port := range flags["port"] {
			t.events = append(t.events, "port:"+s+":"+port)
		}
	}
	t.events = append(t.events, "runtime:"+id)
	if t.crashAt == "activate:after_runtime" {
		t.crashAt = ""
		return map[string]any{"release": id}, "failed", "activation_failed"
	}
	t.active = &fakeActive{Active: id, Previous: previous, Strategy: strategy}
	t.intent = nil
	t.events = append(t.events, "pointer:"+id)
	details := map[string]any{"active_release": id, "previous_release": previous, "strategy": strategy, "legacy_unmapped": unmapped}
	return details, "succeeded", ""
}

func (t *fakeTarget) listing(deploymentID string) map[string]any {
	releases := []map[string]any{}
	ids := make([]string, 0, len(t.staged))
	for id := range t.staged {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		role := "staged"
		if t.active != nil {
			switch id {
			case t.active.Active:
				role = "active"
			case t.active.Previous:
				role = "previous"
			}
		}
		releases = append(releases, map[string]any{"digest": id, "role": role, "state": "complete", "path": "/runtime/releases/" + id})
	}
	out := map[string]any{"deployment_id": deploymentID, "fence": map[string]any{"current": t.fence}, "releases": releases}
	if t.active != nil {
		out["active"] = t.active
	}
	if t.intent != nil {
		out["interrupted_activation"] = t.intent
	}
	return out
}

// verbCalls returns the verbs dispatched, in order.
func (t *fakeTarget) verbs() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, 0, len(t.calls))
	for _, c := range t.calls {
		out = append(out, c.Verb)
	}
	return out
}

// fixtureRelease builds a complete release directory beside a bundle so the
// plan binds a canonical release id: releases/<digest>/{bundle.tar.gz,
// release-manifest.json, vrooli-linux-amd64, .complete}.
func fixtureRelease(t *testing.T, content string) (bundlePath, releaseID string) {
	t.Helper()
	bundle := []byte("fixture-bundle-" + content)
	binary := []byte("fixture-native-cli")
	sumBundle := sha256.Sum256(bundle)
	sumBinary := sha256.Sum256(binary)
	manifest := cloudrelease.Manifest{
		SchemaVersion: cloudrelease.ManifestSchemaVersion, BundleSHA256: hex.EncodeToString(sumBundle[:]),
		NativeCLI:     cloudrelease.NativeCLI{SHA256: hex.EncodeToString(sumBinary[:]), GOOS: "linux", GOARCH: "amd64"},
		ClosureDigest: strings.Repeat("c", 64), ConfigurationDigest: strings.Repeat("d", 64),
		Provenance: cloudrelease.Provenance{Builder: "scenario-to-cloud", Policy: "development-local-unsigned"},
		Limits:     cloudrelease.Limits{MaxEntries: 1000, MaxExpandedBytes: 1 << 20, MaxEntryBytes: 1 << 16},
	}
	digest, err := cloudrelease.ComputeReleaseDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ReleaseDigest = digest
	dir := filepath.Join(t.TempDir(), "releases", digest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(manifest)
	for name, body := range map[string][]byte{cloudrelease.BundleFileName: bundle, cloudrelease.ManifestFileName: raw, cloudrelease.NativeCLIFileName("linux", "amd64"): binary, cloudrelease.CompleteMarker: []byte("")} {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, cloudrelease.BundleFileName), digest
}
