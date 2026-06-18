// Package channel is the node-agent's side of the dial-out wire protocol
// (packages/proto/.../v1/channel/channel.proto). It holds the persistent
// dial-out channel back to the control plane: the node opens an SSE stream the
// control plane pushes ServerFrames down (HandshakeAck/JobPush/ProvisionCommand/
// ControlPing) and, in the other direction, calls the PresenceService
// ReportHeartbeat Connect-RPC on a cadence carrying its self-reported health.
// This is the NAT/firewall-proof half — the node always initiates; there is no
// inbound port on the node.
//
// Phase 1 holds the stream + runs the heartbeat loop with reconnect/backoff
// using a STUB ?node= credential. Phase 2 swaps in the per-node Ed25519 mutual
// auth (SECURITY.md boundary 2): the heartbeat calls get signed and the SSE
// token is bound to the node key, and the node verifies the control plane's
// pushes against the key it pinned at bootstrap. JobPush/ProvisionCommand frame
// handling lands in Phases 3/4 — readFrames already decodes the ServerFrame
// envelope so those phases only add dispatch.
package channel

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"vrooli-bridge/agent/internal/buildinfo"
	"vrooli-bridge/agent/internal/config"
	"vrooli-bridge/agent/internal/exec"
	"vrooli-bridge/agent/internal/health"
	"vrooli-bridge/agent/internal/nodecred"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence/presence_v1connect"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"
)

// ProtocolVersion is the agent's implemented wire protocol version. It MUST
// match CHANNEL_PROTOCOL_VERSION documented in channel.proto; the control
// plane negotiates against it in the handshake and flags a mismatch
// NEEDS_UPDATE rather than silently mis-driving the node.
const ProtocolVersion uint32 = 1

// channelEventsPath is the dial-out SSE route the node opens and holds. It
// mirrors handlers/channel/endpoints.go::channel_events.
const channelEventsPath = "/api/v1/channel/events"

const (
	defaultMinBackoff = 1 * time.Second
	defaultMaxBackoff = 30 * time.Second
	// sessionStableFor is how long a session must hold before its reconnect
	// backoff resets to the minimum — so a flapping control plane backs off but
	// a healthy one that briefly drops reconnects promptly.
	sessionStableFor = 10 * time.Second
)

// ErrNotConfigured is returned by Dial when the agent has no control plane to
// dial out to yet (unpaired). It is an expected, non-fatal condition.
var ErrNotConfigured = errors.New("agent not configured to dial a control plane (no control-plane URL / node id)")

// Client holds the agent's channel configuration and transport seams.
type Client struct {
	cfg        config.Config
	httpClient *http.Client
	rpc        presence_v1connect.PresenceServiceClient
	runsRPC    runs_v1connect.RunsServiceClient
	sampler    health.Sampler
	cred       *nodecred.Credential
	logger     *log.Logger
	now        func() time.Time
	minBackoff time.Duration
	maxBackoff time.Duration

	// baseCtx is the top-level dial context, captured at Dial. Job execution is
	// anchored to it (not the per-session SSE context) so a running job survives
	// a brief channel reconnect and is only cancelled on agent shutdown.
	baseCtx context.Context
}

// Option customises a Client (transport, sampler, clock, backoff) for tests and
// service-install variants without changing the dial logic.
type Option func(*Client)

// WithHTTPClient overrides the HTTP client used for both the SSE stream and the
// heartbeat Connect-RPC. The default has no timeout (the SSE stream is
// long-lived); callers that want a heartbeat timeout layer it at the transport.
func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.httpClient = hc } }

// WithSampler overrides the health sampler (tests inject a fixed snapshot).
func WithSampler(s health.Sampler) Option { return func(c *Client) { c.sampler = s } }

// WithCredential sets the node's mutual-auth keypair. When set, the agent signs
// every heartbeat (X-Bridge-* headers) and binds the dial-out SSE token to the
// key, so the control plane can verify the node's identity. Without it the agent
// falls back to the unauthenticated ?node= form (pre-pairing / Phase-1).
func WithCredential(cred *nodecred.Credential) Option { return func(c *Client) { c.cred = cred } }

// WithLogger overrides the logger.
func WithLogger(l *log.Logger) Option { return func(c *Client) { c.logger = l } }

// WithClock overrides the time source used for heartbeat timestamps.
func WithClock(now func() time.Time) Option { return func(c *Client) { c.now = now } }

// WithBackoff overrides the reconnect backoff bounds.
func WithBackoff(min, max time.Duration) Option {
	return func(c *Client) { c.minBackoff, c.maxBackoff = min, max }
}

// NewClient constructs a channel client from resolved config. The PresenceService
// Connect client is built over the same HTTP client and the control-plane base
// URL; it is only invoked once Dial confirms the agent is paired.
func NewClient(cfg config.Config, opts ...Option) *Client {
	c := &Client{
		cfg:        cfg,
		httpClient: &http.Client{},
		logger:     log.Default(),
		now:        time.Now,
		sampler:    health.NewSystemSampler(cfg.StateDir),
		minBackoff: defaultMinBackoff,
		maxBackoff: defaultMaxBackoff,
	}
	for _, opt := range opts {
		opt(c)
	}
	base := strings.TrimRight(cfg.ControlPlaneURL, "/")
	c.rpc = presence_v1connect.NewPresenceServiceClient(c.httpClient, base)
	c.runsRPC = runs_v1connect.NewRunsServiceClient(c.httpClient, base)
	return c
}

// Handshake builds the Handshake frame the agent presents when it opens the
// channel. It is pure (no I/O) so both Dial and tests can assert on it. The node
// identity, OS/arch, and advertised capabilities come from config + the build
// target; the agent version is the build fingerprint.
func (c *Client) Handshake() *channelv1.Handshake {
	return &channelv1.Handshake{
		ProtocolVersion: ProtocolVersion,
		NodeId:          c.cfg.NodeID,
		AgentVersion:    buildinfo.Fingerprint(),
		Os:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		Capabilities:    append([]string(nil), c.cfg.Capabilities...),
	}
}

// Dial holds the persistent dial-out channel, blocking until ctx is cancelled.
// It opens the SSE stream, runs the heartbeat loop alongside it, and on any
// disconnect reconnects with exponential backoff (reset to the minimum after a
// stably-held session). It returns ErrNotConfigured immediately when the agent
// is unpaired, and nil on a clean ctx-cancelled shutdown.
func (c *Client) Dial(ctx context.Context) error {
	if !c.cfg.Paired() {
		return ErrNotConfigured
	}
	c.baseCtx = ctx

	backoff := c.minBackoff
	for {
		if ctx.Err() != nil {
			return nil
		}

		start := c.now()
		err := c.session(ctx)
		if ctx.Err() != nil {
			return nil
		}

		// A session that held for a while is "healthy"; reset backoff so a
		// transient drop reconnects fast. A session that died immediately keeps
		// escalating backoff so a down/flapping control plane is not hammered.
		if c.now().Sub(start) >= sessionStableFor {
			backoff = c.minBackoff
		}
		if err != nil {
			c.logger.Printf("channel: session ended (%v); reconnecting in %s", err, backoff)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > c.maxBackoff {
			backoff = c.maxBackoff
		}
	}
}

// session opens the SSE stream, runs the heartbeat loop alongside it, and
// returns when the stream closes or ctx is cancelled. The heartbeat goroutine is
// torn down before session returns so the next reconnect starts clean.
func (c *Client) session(ctx context.Context) error {
	sctx, cancel := context.WithCancel(ctx)
	defer cancel()

	body, err := c.openChannel(sctx)
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer body.Close()

	c.logger.Printf("channel: dial-out stream open to %s (node %q)", c.cfg.ControlPlaneURL, c.cfg.NodeID)

	hbDone := make(chan struct{})
	go func() {
		defer close(hbDone)
		c.runHeartbeats(sctx)
	}()

	err = c.readFrames(sctx, body)
	cancel() // stop the heartbeat loop
	<-hbDone // and wait for it to exit before releasing the connection
	return err
}

// openChannel opens the dial-out SSE stream and returns its body. The node id
// rides the ?node= query (EventSource cannot set headers) and is mirrored in
// the X-Bridge-Node header for non-browser clients — the Phase 1 STUB
// credential the server reads in handlers/channel/sse_handler.go.
func (c *Client) openChannel(ctx context.Context) (io.ReadCloser, error) {
	u, err := url.Parse(strings.TrimRight(c.cfg.ControlPlaneURL, "/") + channelEventsPath)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if c.cred != nil {
		// Mutual auth: the dial-out token binds this connection to the node key
		// so the control plane verifies the node before holding the stream.
		q.Set("token", c.cred.Token(c.cfg.NodeID, c.now().UTC()))
	} else {
		q.Set("node", c.cfg.NodeID)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("X-Bridge-Node", c.cfg.NodeID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("control plane returned status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// readFrames consumes the SSE stream until it closes or ctx is cancelled. It
// parses the minimal SSE grammar (comment lines are keepalives; `data:` lines
// accumulate into one ServerFrame per blank-line-terminated event) and decodes
// each event as a channel.ServerFrame with DiscardUnknown so a newer control
// plane never breaks an older agent. Phase 1 only logs HandshakeAck verdicts;
// Phases 3/4 dispatch JobPush/ProvisionCommand from handleServerFrame.
func (c *Client) readFrames(ctx context.Context, body io.Reader) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	var data strings.Builder
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}
		line := scanner.Text()
		switch {
		case line == "":
			if data.Len() > 0 {
				c.handleServerFrame(data.String())
				data.Reset()
			}
		case strings.HasPrefix(line, ":"):
			// comment / keepalive ping — ignore.
		case strings.HasPrefix(line, "data:"):
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		default:
			// other SSE fields (event:, id:, retry:) are not used in Phase 1.
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	// Clean EOF: the server closed the stream. Signal the reconnect loop.
	return io.EOF
}

// handleServerFrame decodes one SSE event payload as a ServerFrame and acts on
// it. Phase 1 only surfaces the compatibility verdict from a HandshakeAck;
// unknown / not-yet-handled frame kinds are ignored (DiscardUnknown semantics).
func (c *Client) handleServerFrame(payload string) {
	var frame channelv1.ServerFrame
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(payload), &frame); err != nil {
		c.logger.Printf("channel: dropping unparseable server frame: %v", err)
		return
	}
	if ack := frame.GetAck(); ack != nil {
		if ack.GetCompatibility() == channelv1.CompatibilityStatus_COMPATIBILITY_STATUS_NEEDS_UPDATE {
			c.logger.Printf("channel: control plane flagged this agent NEEDS_UPDATE (%s) — holding presence only", ack.GetReason())
		}
		if !ack.GetAccepted() {
			c.logger.Printf("channel: control plane refused the channel: %s", ack.GetReason())
		}
	}
	if job := frame.GetJob(); job != nil {
		// A typed job push (OT-P0-004). Run it as the non-privileged runner,
		// anchored to the base (not session) context so it survives a reconnect.
		// Errors are streamed back as RunEvents, not returned; a reporter
		// transport failure is logged.
		c.logger.Printf("channel: received job run_id=%q verb=%q scenario=%q", job.GetRunId(), job.GetVerb(), job.GetScenario())
		go c.runJob(job)
	}
	// ProvisionCommand handling lands in Phase 4.
}

// runJob executes a pushed job via the non-privileged runner, streaming
// status/log/exit RunEvents back to the control plane's RunsService (each call
// signed with the node credential). It is launched in its own goroutine so a
// long job does not block the SSE read loop.
func (c *Client) runJob(job *channelv1.JobPush) {
	ctx := c.baseCtx
	if ctx == nil {
		ctx = context.Background()
	}
	reporter := &runEventReporter{rpc: c.runsRPC, cred: c.cred, nodeID: c.cfg.NodeID, now: c.now}
	runner := exec.NewRunner(c.cfg.VrooliBin, c.cfg.WorkDir, reporter, exec.WithClock(c.now))
	if err := runner.Execute(ctx, job); err != nil && ctx.Err() == nil {
		c.logger.Printf("channel: run %q: reporting events failed: %v", job.GetRunId(), err)
	}
}

// runEventReporter implements exec.EventReporter by calling the control plane's
// RunsService.ReportRunEvent, signing each call with the node's per-node
// Ed25519 credential so the control plane verifies the node (and that it only
// reports against its own runs).
type runEventReporter struct {
	rpc    runs_v1connect.RunsServiceClient
	cred   *nodecred.Credential
	nodeID string
	now    func() time.Time
}

func (r *runEventReporter) Report(ctx context.Context, ev *channelv1.RunEvent) error {
	req := connect.NewRequest(&runsv1.ReportRunEventRequest{Event: ev})
	if r.cred != nil {
		for k, v := range r.cred.Headers(r.nodeID, r.now().UTC()) {
			req.Header().Set(k, v)
		}
	}
	_, err := r.rpc.ReportRunEvent(ctx, req)
	return err
}

// runHeartbeats sends a heartbeat immediately on connect and then on the
// configured interval until ctx is cancelled. The sequence is per-connection
// (reset each session), matching the proto's "monotonic per-connection counter".
func (c *Client) runHeartbeats(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.HeartbeatInterval)
	defer ticker.Stop()

	var seq uint64
	c.sendHeartbeat(ctx, &seq)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.sendHeartbeat(ctx, &seq)
		}
	}
}

// sendHeartbeat samples health and reports one heartbeat. A failure is logged
// (unless ctx was cancelled mid-flight) and does not abort the loop — the next
// tick retries, and a persistently failing stream is handled by the SSE side
// closing, which triggers reconnect.
func (c *Client) sendHeartbeat(ctx context.Context, seq *uint64) {
	*seq++
	now := c.now().UTC()
	hb := &channelv1.Heartbeat{
		NodeId:   c.cfg.NodeID,
		Sequence: *seq,
		Health:   snapshotToProto(c.sampler.Sample()),
		SentAt:   timestamppb.New(now),
	}
	connReq := connect.NewRequest(&presencev1.ReportHeartbeatRequest{Heartbeat: hb})
	if c.cred != nil {
		// Mutual auth: sign the call so the control plane verifies this node.
		for k, v := range c.cred.Headers(c.cfg.NodeID, now) {
			connReq.Header().Set(k, v)
		}
	}
	resp, err := c.rpc.ReportHeartbeat(ctx, connReq)
	if err != nil {
		if ctx.Err() == nil {
			c.logger.Printf("channel: heartbeat seq=%d failed: %v", *seq, err)
		}
		return
	}
	if resp.Msg.GetCompatibility() == channelv1.CompatibilityStatus_COMPATIBILITY_STATUS_NEEDS_UPDATE {
		c.logger.Printf("channel: heartbeat verdict NEEDS_UPDATE — update the agent to receive jobs")
	}
}

// snapshotToProto translates the agent's health.Snapshot into the wire
// HealthSnapshot.
func snapshotToProto(s health.Snapshot) *channelv1.HealthSnapshot {
	out := &channelv1.HealthSnapshot{
		ToolchainPresent:   s.ToolchainPresent,
		DiskHeadroomBytes:  s.DiskHeadroomBytes,
		ContainerRuntimeUp: s.ContainerRuntimeUp,
		Details:            s.Details,
	}
	if !s.ReportedAt.IsZero() {
		out.ReportedAt = timestamppb.New(s.ReportedAt.UTC())
	}
	return out
}
