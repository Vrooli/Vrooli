package generality

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/cloudrelease"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/vps"
)

// fakeTarget is one host running the native `vrooli cloud-target` owner,
// modelled from outside the vps package: it answers the owner's verbs with
// the owner's JSON shapes, keeps the durable target state (fence, receipts
// per (operation, step), staged releases, active pointer, running
// scenarios, resource demand, edge routes, persistent data) and records
// every call so the tests can prove argv discipline and receipt binding.
type fakeTarget struct {
	mu        sync.Mutex
	machineID string
	platform  string
	calls     []reach.Command
	delivered []reach.ArtifactFile
	manifests map[string]string // remote manifest path -> release id
	fence     uint64
	receipts  map[string]fakeReceipt
	staged    map[string]bool
	active    *fakeActive
	running   map[string]bool
	resources map[string]bool
	demand    map[string]map[string]bool
	routes    map[string]fakeRoute
	persist   map[string]bool // binding id -> bound at activation
	backedUp  map[string]int  // binding id -> recovery points that covered it
	backups   int
	events    []string
	scenario  string
}

type fakeActive struct {
	Active   string `json:"active_release"`
	Previous string `json:"previous_release,omitempty"`
	Strategy string `json:"strategy"`
}

type fakeRoute struct {
	UpstreamPort int
	ListenerID   string
	Deployment   string
}

// fakeReceipt is what the owner persisted for one (operation, step).
type fakeReceipt struct {
	Operation string
	Step      string
	Fence     uint64
	Reply     string
}

func newFakeTarget(machineID string) *fakeTarget {
	return &fakeTarget{
		machineID: machineID, platform: "linux/amd64", manifests: map[string]string{}, receipts: map[string]fakeReceipt{}, staged: map[string]bool{},
		running: map[string]bool{}, resources: map[string]bool{}, demand: map[string]map[string]bool{}, routes: map[string]fakeRoute{}, persist: map[string]bool{}, backedUp: map[string]int{},
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
		_ = json.Unmarshal([]byte(stored.Reply), &value)
		value["replayed"] = true
		return reply(value)
	}
	details, outcome, code := body()
	value := map[string]any{"receipt": map[string]any{"outcome": outcome, "details": details, "fence": fence, "operation": first(flags, "operation"), "step": first(flags, "step")}, "replayed": false}
	if code != "" {
		value["error"] = map[string]any{"code": code, "message": code}
	}
	raw, _ := json.Marshal(value)
	t.receipts[key] = fakeReceipt{Operation: first(flags, "operation"), Step: first(flags, "step"), Fence: fence, Reply: string(raw)}
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
			switch action {
			case "process.stop.scoped":
				var subject map[string]map[string]any
				raw, _ := vps.DecodeJSONArg(first(flags, "subject"))
				_ = json.Unmarshal(raw, &subject)
				scenario, _ := subject["process"]["scenario"].(string)
				t.stopScenario(scenario)
			case "apt.packages.ensure", "edge.ufw.allow":
			default:
				return map[string]any{"action": action}, "failed", "action_not_allowed"
			}
			t.events = append(t.events, "repair:"+action)
			return map[string]any{"action": action}, "succeeded", ""
		}), nil
	case "cloud-target data inventory":
		return reply(map[string]any{"deployment_id": first(flags, "deployment"), "scenario": first(flags, "scenario"), "entries": []any{}, "uncovered_count": 0}), nil
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
		if res.ExitCode != 0 {
			return res, nil
		}
		var value map[string]any
		_ = json.Unmarshal([]byte(res.Stdout), &value)
		var bindings []map[string]any
		for _, b := range flags["binding"] {
			raw, _ := vps.DecodeJSONArg(b)
			var one map[string]any
			_ = json.Unmarshal(raw, &one)
			bindings = append(bindings, one)
			if id, _ := one["id"].(string); id != "" {
				t.backedUp[id]++
			}
		}
		value["recovery_point"] = map[string]any{"format_version": 1, "id": first(flags, "operation") + "-" + first(flags, "step"), "deployment_id": first(flags, "deployment"), "bindings": bindings, "refs": map[string]any{"schema_version": first(flags, "schema-version"), "configuration_digest": first(flags, "configuration-digest")}, "captured_at": time.Now().UTC().Format(time.RFC3339Nano), "encrypted": true, "key_ref": first(flags, "key-ref"), "migration_posture": first(flags, "migration-posture"), "checksums": map[string]any{}}
		return reply(value), nil
	case "cloud-target release activate":
		return t.effect(flags, func() (map[string]any, string, string) { return t.activate(flags) }), nil
	case "cloud-target release list":
		return reply(t.listing(first(flags, "deployment"))), nil
	case "cloud-target edge route-apply":
		return t.effect(flags, func() (map[string]any, string, string) {
			raw, _ := vps.DecodeJSONArg(first(flags, "spec"))
			var spec struct {
				Domain string `json:"domain"`
				Routes []struct {
					Host         string `json:"host"`
					UpstreamPort int    `json:"upstream_port"`
					ListenerID   string `json:"listener_id"`
				} `json:"routes"`
			}
			if err := json.Unmarshal(raw, &spec); err != nil || len(spec.Routes) == 0 {
				return nil, "failed", "edge_spec_invalid"
			}
			for _, route := range spec.Routes {
				if route.UpstreamPort <= 0 || route.UpstreamPort == 22 || route.UpstreamPort == 18767 {
					return nil, "failed", "edge_private_listener"
				}
				if owner, ok := t.routes[route.Host]; ok && owner.Deployment != first(flags, "deployment") {
					return map[string]any{"host": route.Host}, "failed", "edge_route_owned_by_other_deployment"
				}
				t.routes[route.Host] = fakeRoute{UpstreamPort: route.UpstreamPort, ListenerID: route.ListenerID, Deployment: first(flags, "deployment")}
				t.events = append(t.events, fmt.Sprintf("route:%s:%d", route.Host, route.UpstreamPort))
			}
			return map[string]any{"domain": spec.Domain}, "succeeded", ""
		}), nil
	case "cloud-target edge route-rollback":
		return t.effect(flags, func() (map[string]any, string, string) {
			owner := first(flags, "deployment")
			removed := 0
			for host, route := range t.routes {
				if route.Deployment == owner {
					delete(t.routes, host)
					removed++
				}
			}
			if removed == 0 {
				return nil, "failed", "edge_rollback_not_eligible"
			}
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
		}
		t.events = append(t.events, "resource:start:"+name)
		return reach.Result{ExitCode: 0, Stdout: "{}"}, nil
	case "resource stop":
		name := positional[0]
		if len(t.demand[name]) > 0 {
			return reach.Result{ExitCode: 1, Stdout: "{}", Stderr: "resource still demanded"}, nil
		}
		t.resources[name] = false
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
			_ = json.Unmarshal([]byte(stored.Reply), &value)
			receipt, _ := value["receipt"].(map[string]any)
			receipt["schema_version"] = 1
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
	for _, spec := range flags["data-binding"] {
		bid, _, _ := strings.Cut(spec, "=")
		t.persist[bid] = true
	}
	scenarios := flags["scenario"]
	if strategy == "maintenance" {
		for _, s := range scenarios {
			t.stopScenario(s)
		}
	}
	for _, s := range scenarios {
		t.running[s] = true
		for _, port := range flags["port"] {
			t.events = append(t.events, "port:"+s+":"+port)
		}
	}
	t.events = append(t.events, "runtime:"+id)
	t.active = &fakeActive{Active: id, Previous: previous, Strategy: strategy}
	t.events = append(t.events, "pointer:"+id)
	return map[string]any{"active_release": id, "previous_release": previous, "strategy": strategy, "legacy_unmapped": []string{}}, "succeeded", ""
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
	return out
}

// snapshot returns copies of the durable state for assertions.
func (t *fakeTarget) snapshot() fakeSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := fakeSnapshot{Fence: t.fence, Routes: map[string]fakeRoute{}, Running: map[string]bool{}, Resources: map[string]bool{}, Persistent: map[string]bool{}, BackedUp: map[string]int{}, Backups: t.backups}
	if t.active != nil {
		copyActive := *t.active
		s.Active = &copyActive
	}
	for k, v := range t.routes {
		s.Routes[k] = v
	}
	for k, v := range t.running {
		s.Running[k] = v
	}
	for k, v := range t.resources {
		s.Resources[k] = v
	}
	for k, v := range t.persist {
		s.Persistent[k] = v
	}
	for k, v := range t.backedUp {
		s.BackedUp[k] = v
	}
	for _, r := range t.receipts {
		s.Receipts = append(s.Receipts, r)
	}
	sort.Slice(s.Receipts, func(i, j int) bool { return s.Receipts[i].Step < s.Receipts[j].Step })
	s.Calls = append(s.Calls, t.calls...)
	s.Events = append(s.Events, t.events...)
	return s
}

type fakeSnapshot struct {
	Fence      uint64
	Active     *fakeActive
	Routes     map[string]fakeRoute
	Running    map[string]bool
	Resources  map[string]bool
	Persistent map[string]bool
	BackedUp   map[string]int
	Backups    int
	Receipts   []fakeReceipt
	Calls      []reach.Command
	Events     []string
}
