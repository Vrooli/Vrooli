package validationmatrix

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

// A bridge node advertises the dispatch verbs it accepts, not the platforms it
// can build for. Those are different vocabularies, so a ramp cannot learn from
// the registry alone whether a node has an Apple toolchain or an Android SDK.
//
// The honest way to answer that is to ask the node, through the same allowlisted
// dispatch path everything else uses: `host inventory --json` is collected by
// the control plane, so the facts are probed rather than declared. Results are
// cached because a toolchain changes on the timescale of an install, while a
// target inventory may be requested many times per run.

// HostFactsTTL bounds how long a probed remote toolchain observation is reused.
// It is deliberately short enough that installing Xcode becomes visible within
// one working pause, and long enough that listing targets does not dispatch a
// job per node per call.
const HostFactsTTL = 5 * time.Minute

// hostProbeVerb and hostProbeArgs must stay within the node's allowlist. They
// name a read-only control-plane probe; nothing here mutates remote state.
const hostProbeVerb = "host inventory"

var hostProbeArgs = []string{"--json"}

// HostTool mirrors the control plane's runtime-tool fact. An empty Version on a
// present tool means the probe did not read one, never that none exists.
type HostTool struct {
	Present bool   `json:"present"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
}

// HostFacts is the subset of the control-plane host snapshot a delivery ramp
// needs in order to classify a node. It is intentionally narrow: a ramp decides
// platform capability, not host policy.
type HostFacts struct {
	OS              string              `json:"os"`
	Arch            string              `json:"arch"`
	RuntimeTools    map[string]HostTool `json:"runtime_tools"`
	ProbeStatuses   map[string]string   `json:"probe_statuses"`
	SessionType     string              `json:"session_type"`
	DisplayAttached bool                `json:"display_attached"`
}

// Tool reports one probed tool. The second result distinguishes "probed and
// absent" from "never probed", which the caller must not conflate.
func (f HostFacts) Tool(name string) (HostTool, bool) {
	tool, probed := f.RuntimeTools[name]
	return tool, probed
}

// HasTool reports whether a tool was probed and found present.
func (f HostFacts) HasTool(name string) bool {
	tool, probed := f.RuntimeTools[name]
	return probed && tool.Present
}

// HostProber resolves remote host facts for one node.
type HostProber interface {
	ProbeHost(ctx context.Context, nodeID string) (HostFacts, error)
}

type cachedHostFacts struct {
	facts    HostFacts
	observed time.Time
	err      error
}

// dispatchHostProber probes a node through bridge dispatch and reads the
// resulting JSON out of the durable run's log events.
type dispatchHostProber struct {
	dispatcher Dispatcher
	runs       Runs
	ttl        time.Duration
	now        func() time.Time

	mu    sync.Mutex
	cache map[string]cachedHostFacts
}

func newDispatchHostProber(dispatcher Dispatcher, runs Runs) *dispatchHostProber {
	return &dispatchHostProber{
		dispatcher: dispatcher,
		runs:       runs,
		ttl:        HostFactsTTL,
		now:        time.Now,
		cache:      map[string]cachedHostFacts{},
	}
}

func (p *dispatchHostProber) ProbeHost(ctx context.Context, nodeID string) (HostFacts, error) {
	if p == nil || p.dispatcher == nil || p.runs == nil {
		return HostFacts{}, fmt.Errorf("bridge host probe is not configured")
	}
	if cached, ok := p.cachedFacts(nodeID); ok {
		return cached.facts, cached.err
	}
	facts, err := p.probe(ctx, nodeID)
	p.mu.Lock()
	p.cache[nodeID] = cachedHostFacts{facts: facts, observed: p.now(), err: err}
	p.mu.Unlock()
	return facts, err
}

func (p *dispatchHostProber) cachedFacts(nodeID string) (cachedHostFacts, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cached, ok := p.cache[nodeID]
	if !ok || p.now().Sub(cached.observed) >= p.ttl {
		return cachedHostFacts{}, false
	}
	return cached, true
}

func (p *dispatchHostProber) probe(ctx context.Context, nodeID string) (HostFacts, error) {
	runID, err := dispatchProbeJob(ctx, p.dispatcher, nodeID)
	if err != nil {
		return HostFacts{}, err
	}
	if _, err := p.runs.WaitRun(ctx, connect.NewRequest(&runsv1.WaitRunRequest{Id: runID, TimeoutSeconds: 120})); err != nil {
		return HostFacts{}, fmt.Errorf("wait host probe run %s: %w", runID, err)
	}
	detail, err := p.runs.GetRun(ctx, connect.NewRequest(&runsv1.GetRunRequest{Id: runID}))
	if err != nil {
		return HostFacts{}, fmt.Errorf("read host probe run %s: %w", runID, err)
	}
	if detail == nil || detail.Msg == nil {
		return HostFacts{}, fmt.Errorf("host probe run %s returned no detail", runID)
	}
	if run := detail.Msg.Run; run != nil && run.Status != runsv1.RunStatus_RUN_STATUS_PASSED {
		return HostFacts{}, fmt.Errorf("host probe run %s did not pass", runID)
	}
	return ParseHostFacts(RunLogText(detail.Msg.Events))
}

func dispatchProbeJob(ctx context.Context, dispatcher Dispatcher, nodeID string) (string, error) {
	dispatched, err := dispatcher.DispatchJob(ctx, connect.NewRequest(&dispatchv1.DispatchJobRequest{
		NodeId:         nodeID,
		Verb:           hostProbeVerb,
		Args:           append([]string(nil), hostProbeArgs...),
		TimeoutSeconds: 120,
	}))
	if err != nil {
		return "", fmt.Errorf("dispatch host probe: %w", err)
	}
	if dispatched == nil || dispatched.Msg == nil || strings.TrimSpace(dispatched.Msg.RunId) == "" {
		return "", fmt.Errorf("host probe dispatch returned no durable run identity")
	}
	return dispatched.Msg.RunId, nil
}

// RunLogText concatenates a durable run's log events in sequence order. Bridge
// emits command output as log chunks, so this is how a dispatched probe's
// payload is recovered.
func RunLogText(events []*sharedv1.RunEvent) string {
	var builder strings.Builder
	for _, event := range events {
		if event == nil || event.Kind != sharedv1.RunEventKind_RUN_EVENT_KIND_LOG {
			continue
		}
		builder.WriteString(event.LogChunk)
	}
	return builder.String()
}

// ParseHostFacts extracts the host snapshot from dispatched probe output. The
// control plane may print human preamble around the JSON document, so the
// parser locates the outermost object rather than assuming a clean stream.
func ParseHostFacts(output string) (HostFacts, error) {
	trimmed := strings.TrimSpace(output)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end <= start {
		return HostFacts{}, fmt.Errorf("host probe output contains no JSON document")
	}
	var facts HostFacts
	if err := json.Unmarshal([]byte(trimmed[start:end+1]), &facts); err != nil {
		return HostFacts{}, fmt.Errorf("decode host probe output: %w", err)
	}
	if strings.TrimSpace(facts.OS) == "" {
		return HostFacts{}, fmt.Errorf("host probe output declares no operating system")
	}
	if facts.RuntimeTools == nil {
		facts.RuntimeTools = map[string]HostTool{}
	}
	return facts, nil
}
