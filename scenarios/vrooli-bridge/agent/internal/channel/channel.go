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
// pushes against the key it pinned at bootstrap. Phase 3 dispatches JobPush
// frames to the non-privileged runner (internal/exec); Phase 4 dispatches
// ProvisionCommand frames to the STRUCTURALLY SEPARATE privileged helper
// (internal/privsep). readFrames decodes the ServerFrame envelope with
// DiscardUnknown so a newer control plane never breaks an older agent.
package channel

import (
	"bufio"
	"context"
	"crypto/ecdh"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/cliresolve"

	"vrooli-bridge/agent/internal/buildinfo"
	"vrooli-bridge/agent/internal/config"
	"vrooli-bridge/agent/internal/cpverify"
	"vrooli-bridge/agent/internal/credentialgrant"
	"vrooli-bridge/agent/internal/credentialpush"
	"vrooli-bridge/agent/internal/exec"
	"vrooli-bridge/agent/internal/health"
	"vrooli-bridge/agent/internal/nodecred"
	"vrooli-bridge/agent/internal/privsep"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts/artifacts_v1connect"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup/cleanup_v1connect"
	credentialgrantv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	credentialgrantconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant/credentialgrant_v1connect"
	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence/presence_v1connect"
	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision/provision_v1connect"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
	"github.com/vrooli/vrooli/packages/proto/privilegedops"
)

// ProtocolVersion is the agent's implemented wire protocol version. It MUST
// match CHANNEL_PROTOCOL_VERSION documented in channel.proto; the control
// plane negotiates against it in the handshake and flags a mismatch
// NEEDS_UPDATE rather than silently mis-driving the node.
const ProtocolVersion uint32 = 2

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
	cfg            config.Config
	machineArch    string
	httpClient     *http.Client
	rpc            presence_v1connect.PresenceServiceClient
	grantRPC       credentialgrantconnect.CredentialGrantServiceClient
	runsRPC        runs_v1connect.RunsServiceClient
	artifactsRPC   artifacts_v1connect.ArtifactsServiceClient
	provisionRPC   provision_v1connect.ProvisionServiceClient
	cleanupRPC     cleanup_v1connect.CleanupServiceClient
	sampler        health.Sampler
	cred           *nodecred.Credential
	encryption     *nodecred.EncryptionCredential
	grantStore     credentialgrant.Store
	credentialSink credentialpush.Sink
	ephemeral      *credentialpush.EphemeralStore
	cpVerifier     *cpverify.Verifier
	logger         *log.Logger
	now            func() time.Time
	minBackoff     time.Duration
	maxBackoff     time.Duration

	// rejectedFrames counts control-plane pushes dropped because they did not
	// verify against the pinned control-plane key (unsigned, mis-signed, or
	// wrong-key). It is surfaced on every heartbeat's HealthSnapshot details so an
	// operator sees a node being fed impostor frames.
	rejectedFrames atomic.Uint64

	// rejectedCredentialPushes counts signed pushes that the node refused after
	// signature verification because local grant consent or decryption/storage
	// policy did not allow them. It is reported on the next heartbeat without
	// exposing any credential value.
	rejectedCredentialPushes atomic.Uint64

	// baseCtx is the top-level dial context, captured at Dial. Job execution is
	// anchored to it (not the per-session SSE context) so a running job survives
	// a brief channel reconnect and is only cancelled on agent shutdown.
	baseCtx context.Context

	// runningJobs maps an in-flight run id to the cancel func for its execution
	// context, so a control-plane AbortJob frame (OT-P1-004) stops the run's
	// process instead of letting it run to completion as an ignored stale
	// completion.
	mu            sync.Mutex
	runningJobs   map[string]context.CancelFunc
	runningRelays map[string]*relayState
	sessions      map[string]*nodeSession
	relayReporter RelayResponseReporter
	commandRunner exec.CommandRunner
	shutdown      func()
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

// WithEncryptionCredential supplies the independent X25519 key used only for
// decrypting grant-governed credential pushes.
func WithEncryptionCredential(cred *nodecred.EncryptionCredential) Option {
	return func(c *Client) { c.encryption = cred }
}

func WithCredentialGrants(grants credentialgrant.Store) Option {
	return func(c *Client) { c.grantStore = grants }
}

func WithCredentialSink(sink credentialpush.Sink) Option {
	return func(c *Client) { c.credentialSink = sink }
}

func WithEphemeralCredentials(store *credentialpush.EphemeralStore) Option {
	return func(c *Client) { c.ephemeral = store }
}

// WithCPVerifier pins the control-plane public key the agent verifies every
// server push against (SECURITY.md boundary 2). It is REQUIRED for a paired
// agent: without it handleServerFrame rejects every frame, so a node can never
// be driven by an unsigned or impostor control plane. main wires it from the
// key `pair redeem` pinned at bootstrap; a missing pin is a hard startup failure
// there, not a silent trust-on-first-use here.
func WithCPVerifier(v *cpverify.Verifier) Option { return func(c *Client) { c.cpVerifier = v } }

// WithLogger overrides the logger.
func WithLogger(l *log.Logger) Option { return func(c *Client) { c.logger = l } }

// WithClock overrides the time source used for heartbeat timestamps.
func WithClock(now func() time.Time) Option { return func(c *Client) { c.now = now } }

// WithBackoff overrides the reconnect backoff bounds.
func WithBackoff(min, max time.Duration) Option {
	return func(c *Client) { c.minBackoff, c.maxBackoff = min, max }
}

// RelayResponseReporter is the node-authenticated response transport seam.
// Tests use a collector; production uses PresenceService.ReportRelayResponse.
type RelayResponseReporter interface {
	ReportRelayResponse(context.Context, *sharedv1.RelayResponse) error
}

func WithRelayResponseReporter(reporter RelayResponseReporter) Option {
	return func(c *Client) { c.relayReporter = reporter }
}

func WithCommandRunner(runner exec.CommandRunner) Option {
	return func(c *Client) { c.commandRunner = runner }
}

// WithShutdown supplies the process lifecycle hook used after a successful
// node cleanup. The cleanup receipt is reported first; only then does the
// managed agent stop, allowing its just-removed service unit to disappear
// without orphaning the control-plane operation.
func WithShutdown(shutdown func()) Option { return func(c *Client) { c.shutdown = shutdown } }

// NewClient constructs a channel client from resolved config. The PresenceService
// Connect client is built over the same HTTP client and the control-plane base
// URL; it is only invoked once Dial confirms the agent is paired.
func NewClient(cfg config.Config, opts ...Option) *Client {
	c := &Client{
		cfg:         cfg,
		machineArch: MachineArchitecture(),
		httpClient:  &http.Client{},
		logger:      log.Default(),
		now:         time.Now,
		ephemeral:   credentialpush.NewEphemeralStore(),
		sampler:     health.NewSystemSampler(cfg.StateDir),
		minBackoff:  defaultMinBackoff,
		maxBackoff:  defaultMaxBackoff,
	}
	for _, opt := range opts {
		opt(c)
	}
	base := strings.TrimRight(cfg.ControlPlaneURL, "/")
	c.rpc = presence_v1connect.NewPresenceServiceClient(c.httpClient, base)
	c.grantRPC = credentialgrantconnect.NewCredentialGrantServiceClient(c.httpClient, base)
	c.runsRPC = runs_v1connect.NewRunsServiceClient(c.httpClient, base)
	c.artifactsRPC = artifacts_v1connect.NewArtifactsServiceClient(c.httpClient, base)
	c.provisionRPC = provision_v1connect.NewProvisionServiceClient(c.httpClient, base)
	c.cleanupRPC = cleanup_v1connect.NewCleanupServiceClient(c.httpClient, base)
	return c
}

// Handshake builds the Handshake frame the agent presents when it opens the
// channel. It is pure (no I/O) so both Dial and tests can assert on it. The node
// identity, OS/arch, and advertised capabilities come from config + the build
// target; the agent version is the build fingerprint.
func (c *Client) Handshake() *channelv1.Handshake {
	return &channelv1.Handshake{
		ProtocolVersion:   ProtocolVersion,
		NodeId:            c.cfg.NodeID,
		AgentVersion:      buildinfo.Fingerprint(),
		Os:                runtime.GOOS,
		Arch:              runtime.GOARCH,
		MachineArch:       c.machineArch,
		BinaryArch:        runtime.GOARCH,
		Capabilities:      append([]string(nil), c.cfg.Capabilities...),
		SupportsWebsocket: true,
	}
}

// SyncCredentialGrants reconciles metadata consent with the control plane at
// startup. It is the offline-revocation safety net: grants revoked while the
// node was away are removed from the local grant store and their corresponding
// grant-owned authority entries are purged. No credential value is returned by
// this RPC.
func (c *Client) SyncCredentialGrants(ctx context.Context) error {
	if c.grantRPC == nil || c.grantStore == nil || c.cfg.NodeID == "" {
		return nil
	}
	req := connect.NewRequest(&credentialgrantv1.SyncNodeGrantsRequest{NodeId: c.cfg.NodeID})
	if c.cred != nil {
		for key, value := range c.cred.Headers(c.cfg.NodeID, c.now().UTC()) {
			req.Header().Set(key, value)
		}
	}
	response, err := c.grantRPC.SyncNodeGrants(ctx, req)
	if err != nil {
		return fmt.Errorf("sync credential grants: %w", err)
	}
	active := make(map[string]credentialgrant.Grant, len(response.Msg.GetGrants()))
	for _, grant := range response.Msg.GetGrants() {
		if grant.GetNodeId() != "" && grant.GetNodeId() != c.cfg.NodeID {
			continue
		}
		metadata := credentialgrant.Grant{ID: grant.GetId(), NodeID: c.cfg.NodeID, LogicalID: grant.GetLogicalId(), Field: grant.GetField(), Class: grant.GetClass(), Retention: grant.GetRetention(), Generation: grant.GetGeneration()}
		if err := c.grantStore.Put(metadata); err != nil {
			return fmt.Errorf("store active credential grant metadata: %w", err)
		}
		active[metadata.LogicalID+":"+metadata.Field] = metadata
	}
	for _, local := range c.grantStore.List() {
		key := local.LogicalID + ":" + local.Field
		if _, ok := active[key]; ok || local.Revoked {
			continue
		}
		if err := c.grantStore.Revoke(local.LogicalID, local.Field); err != nil {
			return fmt.Errorf("revoke stale local credential grant: %w", err)
		}
		if c.credentialSink != nil {
			if err := c.credentialSink.Delete(local.LogicalID, local.Field); err != nil {
				return fmt.Errorf("purge stale granted credential: %w", err)
			}
		}
	}
	return nil
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
	q.Set("pv", strconv.FormatUint(uint64(ProtocolVersion), 10))
	q.Set("machine_arch", c.machineArch)
	q.Set("binary_arch", runtime.GOARCH)
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
		_ = resp.Body.Close() // #nosec G104 -- the response is being discarded after a non-200 status; there is no recovery action for Close failure.
		return nil, fmt.Errorf("control plane returned status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// MachineArchitecture returns the architecture reported by the host kernel,
// normalized to the Go/toolchain vocabulary used by Bridge. This is distinct
// from runtime.GOARCH, which describes the agent binary and may be translated
// on the host (for example Rosetta on macOS).
func MachineArchitecture() string {
	if runtime.GOOS == "windows" {
		if raw := os.Getenv("PROCESSOR_ARCHITEW6432"); raw != "" {
			return normalizeMachineArchitecture(raw)
		}
		if raw := os.Getenv("PROCESSOR_ARCHITECTURE"); raw != "" {
			return normalizeMachineArchitecture(raw)
		}
	} else {
		if raw, err := osexec.Command("uname", "-m").Output(); err == nil {
			if normalized := normalizeMachineArchitecture(string(raw)); normalized != "" {
				return normalized
			}
		}
	}
	return normalizeMachineArchitecture(runtime.GOARCH)
}

func normalizeMachineArchitecture(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	case "armv7", "armv7l", "arm":
		return "arm"
	case "i386", "i486", "i586", "i686", "x86", "386":
		return "386"
	default:
		return strings.TrimSpace(raw)
	}
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

// handleServerFrame verifies one SSE event payload against the pinned
// control-plane key and, only if it verifies, acts on the ServerFrame it carries
// (SECURITY.md boundary 2). A frame that is unsigned, mis-signed, signed by a
// different key, or otherwise unverifiable is rejected BEFORE any dispatch,
// provisioning, or abort handler sees it: it is logged and counted
// (rejectedFrames, surfaced on the next heartbeat) and dropped. There is no path
// by which an unverified frame reaches a handler. A HandshakeAck's compatibility
// verdict is surfaced; unknown / not-yet-handled frame kinds are ignored
// (DiscardUnknown semantics).
func (c *Client) handleServerFrame(payload string) {
	if c.cpVerifier == nil {
		// A paired agent is always wired with a verifier (main fails hard on a
		// missing pin). Its absence here means a mis-wired build, not an untrusted
		// control plane — refuse to act rather than trust the frame.
		c.rejectedFrames.Add(1)
		c.logger.Printf("channel: rejecting server frame — no pinned control-plane key wired (refusing to act on unverified pushes)")
		return
	}
	frame, err := c.cpVerifier.Open([]byte(payload))
	if err != nil {
		n := c.rejectedFrames.Add(1)
		c.logger.Printf("channel: rejected unverified server frame (%v); total rejected=%d", err, n)
		return
	}
	if frame.GetJob() != nil || frame.GetProvision() != nil || frame.GetCleanup() != nil || frame.GetAbort() != nil {
		c.sendDeliveryAck(frame)
	}
	if session := frame.GetSession(); session != nil {
		c.handleSessionFrame(session)
	}
	if ack := frame.GetAck(); ack != nil {
		if ack.GetCompatibility() == sharedv1.CompatibilityStatus_COMPATIBILITY_STATUS_NEEDS_UPDATE {
			c.logger.Printf("channel: control plane flagged this agent NEEDS_UPDATE (%s) — holding presence only", ack.GetReason())
		}
		if !ack.GetAccepted() {
			c.logger.Printf("channel: control plane refused the channel: %s", ack.GetReason())
		}
	}
	if job := frame.GetJob(); job != nil {
		if c.cfg.PresenceOnly {
			c.logger.Printf("channel: rejecting job run_id=%q because agent is in presence-only posture", job.GetRunId())
			return
		}
		// A typed job push (OT-P0-004). Run it as the non-privileged runner,
		// anchored to the base (not session) context so it survives a reconnect.
		// Errors are streamed back as RunEvents, not returned; a reporter
		// transport failure is logged.
		c.logger.Printf("channel: received job run_id=%q verb=%q scenario=%q", job.GetRunId(), job.GetVerb(), job.GetScenario())
		go c.runJob(job)
	}
	if relayRequest := frame.GetRelay(); relayRequest != nil {
		if c.cfg.PresenceOnly {
			c.logger.Printf("channel: rejecting relay correlation_id=%q because agent is in presence-only posture", relayRequest.GetCorrelationId())
			return
		}
		c.logger.Printf("channel: received relay correlation_id=%q command=%q scenario=%q", relayRequest.GetCorrelationId(), relayRequest.GetCommand(), relayRequest.GetScenario())
		go c.runRelay(relayRequest)
	}
	if relayCancel := frame.GetRelayCancel(); relayCancel != nil {
		c.logger.Printf("channel: received relay cancel correlation_id=%q reason=%q", relayCancel.GetCorrelationId(), relayCancel.GetReason())
		if !c.cancelRelay(relayCancel.GetCorrelationId(), relayCancel.GetReason()) {
			c.logger.Printf("channel: relay cancel for unknown or already-finished correlation_id=%q (ignored)", relayCancel.GetCorrelationId())
		}
	}
	if prov := frame.GetProvision(); prov != nil {
		if c.cfg.PresenceOnly {
			c.logger.Printf("channel: rejecting provisioning op_id=%q because agent is in presence-only posture", prov.GetOpId())
			return
		}
		// A privileged provisioning command (OT-P0-006). It is executed by the
		// STRUCTURALLY SEPARATE privileged helper (internal/privsep), never the
		// runner, and is anchored to the base context so it survives a reconnect.
		c.logger.Printf("channel: received provision op_id=%q target=%q rollback=%q",
			prov.GetOpId(), prov.GetTargetRevision(), prov.GetRollbackRevision())
		go c.runProvision(prov)
	}
	if cleanup := frame.GetCleanup(); cleanup != nil {
		if c.cfg.PresenceOnly {
			c.logger.Printf("channel: rejecting cleanup op_id=%q because agent is in presence-only posture", cleanup.GetOpId())
			return
		}
		c.logger.Printf("channel: received typed cleanup op_id=%q operation=%s", cleanup.GetOpId(), cleanup.GetOperation().String())
		go c.runCleanup(cleanup)
	}
	if abort := frame.GetAbort(); abort != nil {
		// A control-plane cancel (OT-P1-004): stop the in-flight run's process so
		// it does not run to completion. Unknown/finished runs are a no-op.
		c.logger.Printf("channel: received abort run_id=%q reason=%q", abort.GetRunId(), abort.GetReason())
		if !c.cancelJob(abort.GetRunId()) {
			c.logger.Printf("channel: abort for unknown or already-finished run %q (ignored)", abort.GetRunId())
		}
	}
	if push := frame.GetCredentialPush(); push != nil {
		c.handleCredentialPush(push)
	}
	if purge := frame.GetCredentialPurge(); purge != nil {
		c.handleCredentialPurge(purge)
	}
	if grant := frame.GetCredentialGrant(); grant != nil {
		c.handleCredentialGrant(grant)
	}
}

func (c *Client) handleCredentialGrant(grant *channelv1.CredentialGrant) {
	if grant == nil || c.grantStore == nil {
		return
	}
	if grant.GetNodeId() != "" && grant.GetNodeId() != c.cfg.NodeID {
		return
	}
	if grant.GetRevoked() {
		if err := c.grantStore.Revoke(grant.GetLogicalId(), grant.GetField()); err != nil {
			c.logger.Printf("channel: credential grant revoke failed for logical_id=%q field=%q: %v", grant.GetLogicalId(), grant.GetField(), err)
		}
		return
	}
	if err := c.grantStore.Put(credentialgrant.Grant{
		ID: grant.GetGrantId(), NodeID: grant.GetNodeId(), LogicalID: grant.GetLogicalId(), Field: grant.GetField(),
		Class: grant.GetClass(), Retention: grant.GetRetention(), Generation: grant.GetGeneration(),
	}); err != nil {
		c.logger.Printf("channel: credential grant refused for logical_id=%q field=%q: %v", grant.GetLogicalId(), grant.GetField(), err)
	}
}

func (c *Client) handleCredentialPush(push *channelv1.CredentialPush) {
	var private *ecdh.PrivateKey
	if c.encryption != nil {
		private = c.encryption.PrivateKey()
	}
	result, err := credentialpush.Apply(push, c.cfg.NodeID, private, c.grantStore, c.credentialSink)
	if err != nil {
		c.rejectedCredentialPushes.Add(1)
		c.logger.Printf("channel: credential push refused for logical_id=%q field=%q: %v", push.GetLogicalId(), push.GetField(), err)
		c.reportCredentialReceipt(&channelv1.CredentialReceipt{GrantId: push.GetGrantId(), NodeId: c.cfg.NodeID, LogicalId: push.GetLogicalId(), Field: push.GetField(), Generation: push.GetGeneration(), Accepted: false, Reason: "ingest failed"})
		return
	}
	if result.Rejected {
		c.rejectedCredentialPushes.Add(1)
		c.logger.Printf("channel: credential push refused for logical_id=%q field=%q: %s", push.GetLogicalId(), push.GetField(), result.RejectReason)
		c.reportCredentialReceipt(result.Receipt)
		return
	}
	c.reportCredentialReceipt(result.Receipt)
	if len(result.Ephemeral) > 0 {
		if c.ephemeral != nil {
			_ = c.ephemeral.Put(push.GetLogicalId(), push.GetField(), result.Ephemeral)
		}
		credentialpush.Zero(result.Ephemeral)
	}
}

func (c *Client) reportCredentialReceipt(receipt *channelv1.CredentialReceipt) {
	if receipt == nil || c.rpc == nil {
		return
	}
	ctx := c.baseCtx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req := connect.NewRequest(&presencev1.ReportCredentialReceiptRequest{Receipt: &presencev1.CredentialReceipt{
		GrantId: receipt.GetGrantId(), NodeId: receipt.GetNodeId(), LogicalId: receipt.GetLogicalId(), Field: receipt.GetField(), Generation: receipt.GetGeneration(), Accepted: receipt.GetAccepted(), Reason: receipt.GetReason(),
	}})
	if c.cred != nil {
		for key, value := range c.cred.Headers(c.cfg.NodeID, c.now().UTC()) {
			req.Header().Set(key, value)
		}
	}
	if _, err := c.rpc.ReportCredentialReceipt(ctx, req); err != nil {
		c.logger.Printf("channel: credential receipt failed for logical_id=%q field=%q generation=%d: %v", receipt.GetLogicalId(), receipt.GetField(), receipt.GetGeneration(), err)
	}
}

func (c *Client) handleCredentialPurge(purge *channelv1.CredentialPurge) {
	if purge.GetNodeId() != "" && purge.GetNodeId() != c.cfg.NodeID {
		return
	}
	if c.grantStore == nil {
		return
	}
	for _, address := range purge.GetAddresses() {
		parts := strings.SplitN(address, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if err := c.grantStore.Revoke(parts[0], parts[1]); err != nil {
			c.logger.Printf("channel: credential purge refused for address metadata: %v", err)
			continue
		}
		if c.credentialSink != nil {
			if err := c.credentialSink.Delete(parts[0], parts[1]); err != nil {
				c.logger.Printf("channel: credential purge store operation failed for address metadata: %v", err)
			}
		}
	}
}

// sendDeliveryAck reports receipt immediately after signature verification and
// before dispatching any work. It uses the same node credential and RPC as
// heartbeats, so acknowledgements cannot introduce a weaker authentication
// path. A nil RPC is allowed only in isolated frame-handler unit tests.
func (c *Client) sendDeliveryAck(frame *channelv1.ServerFrame) {
	if c.rpc == nil {
		return
	}
	ack := &sharedv1.DeliveryAck{
		FrameId:    frame.GetFrameId(),
		ReceivedAt: timestamppb.New(c.now().UTC()),
	}
	if job := frame.GetJob(); job != nil {
		ack.RunId = job.GetRunId()
	}
	if prov := frame.GetProvision(); prov != nil {
		ack.OpId = prov.GetOpId()
	}
	if cleanup := frame.GetCleanup(); cleanup != nil {
		ack.OpId = cleanup.GetOpId()
	}
	if abort := frame.GetAbort(); abort != nil {
		ack.RunId = abort.GetRunId()
	}
	if ack.GetFrameId() == "" {
		c.logger.Printf("channel: refusing to ack server frame without frame_id")
		return
	}
	req := connect.NewRequest(&presencev1.ReportDeliveryAckRequest{Ack: ack})
	req.Header().Set("X-Bridge-Node", c.cfg.NodeID)
	if c.cred != nil {
		for k, v := range c.cred.Headers(c.cfg.NodeID, c.now().UTC()) {
			req.Header().Set(k, v)
		}
	}
	if _, err := c.rpc.ReportDeliveryAck(c.baseCtxOrBackground(), req); err != nil {
		c.logger.Printf("channel: delivery ack frame_id=%q failed: %v", ack.GetFrameId(), err)
	}
}

func (c *Client) baseCtxOrBackground() context.Context {
	if c.baseCtx != nil {
		return c.baseCtx
	}
	return context.Background()
}

// registerJob records the cancel func for a running job so an AbortJob can stop
// it. unregisterJob clears it when the job finishes.
func (c *Client) registerJob(runID string, cancel context.CancelFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runningJobs == nil {
		c.runningJobs = make(map[string]context.CancelFunc)
	}
	c.runningJobs[runID] = cancel
}

func (c *Client) unregisterJob(runID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.runningJobs, runID)
}

// cancelJob cancels a running job's execution context. It reports whether a
// matching in-flight job was found.
func (c *Client) cancelJob(runID string) bool {
	c.mu.Lock()
	cancel := c.runningJobs[runID]
	c.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

type relayState struct {
	cancel context.CancelFunc
	reason string
}

func (c *Client) registerRelay(correlationID string, cancel context.CancelFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runningRelays == nil {
		c.runningRelays = make(map[string]*relayState)
	}
	c.runningRelays[correlationID] = &relayState{cancel: cancel}
}

func (c *Client) unregisterRelay(correlationID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.runningRelays, correlationID)
}

func (c *Client) cancelRelay(correlationID, reason string) bool {
	c.mu.Lock()
	state := c.runningRelays[correlationID]
	if state != nil {
		state.reason = strings.TrimSpace(reason)
	}
	c.mu.Unlock()
	if state == nil {
		return false
	}
	state.cancel()
	return true
}

func (c *Client) relayCancelReason(correlationID string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if state := c.runningRelays[correlationID]; state != nil {
		return state.reason
	}
	return ""
}

// runJob executes a pushed job via the non-privileged runner, streaming
// status/log/exit RunEvents back to the control plane's RunsService (each call
// signed with the node credential). It is launched in its own goroutine so a
// long job does not block the SSE read loop.
func (c *Client) runJob(job *channelv1.JobPush) {
	base := c.baseCtx
	if base == nil {
		base = context.Background()
	}
	// Per-job cancelable context so an AbortJob frame can stop THIS run without
	// affecting others or the channel. Registered under the run id for the
	// lifetime of the execution.
	ctx, cancel := context.WithCancel(base)
	c.registerJob(job.GetRunId(), cancel)
	defer func() {
		cancel()
		c.unregisterJob(job.GetRunId())
	}()

	reporter := &runEventReporter{rpc: c.runsRPC, cred: c.cred, nodeID: c.cfg.NodeID, now: c.now}
	uploader := &artifactUploader{rpc: c.artifactsRPC, cred: c.cred, nodeID: c.cfg.NodeID, now: c.now}
	runnerOpts := []exec.Option{
		exec.WithClock(c.now), exec.WithArtifactUploader(uploader),
		exec.WithArtifactDir(filepath.Join(c.cfg.StateDir, "artifacts")),
		exec.WithCredentialEnvironment(c.ephemeral),
	}
	if c.commandRunner == nil {
		runnerOpts = append(runnerOpts, exec.WithCLIResolver(cliresolve.New(homeForVrooliBinary(c.cfg.VrooliBin))))
	}
	if c.commandRunner != nil {
		runnerOpts = append(runnerOpts, exec.WithCommandRunner(c.commandRunner))
	}
	runner := exec.NewRunner(c.cfg.VrooliBin, c.cfg.WorkDir, reporter,
		runnerOpts...)
	if err := runner.Execute(ctx, job); err != nil && base.Err() == nil {
		c.logger.Printf("channel: run %q: reporting events failed: %v", job.GetRunId(), err)
	}
}

// runRelay executes one short-lived typed command and sends bounded response
// chunks back over the authenticated Presence RPC. The execution context is
// registered before the process starts, so a signed RelayCancel reaches the
// exact command and CommandContext terminates it.
func (c *Client) runRelay(request *channelv1.RelayRequest) {
	base := c.baseCtx
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithCancel(base)
	c.registerRelay(request.GetCorrelationId(), cancel)
	defer func() {
		cancel()
		c.unregisterRelay(request.GetCorrelationId())
	}()

	reporter := c.relayReporter
	if reporter == nil {
		reporter = &relayResponseReporter{rpc: c.rpc, cred: c.cred, nodeID: c.cfg.NodeID, now: c.now}
	}
	reportCtx, stopReporting := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopReporting()
	sequence := uint64(0)
	var reportedBytes uint64
	report := func(kind sharedv1.RelayResponseKind, data []byte, reason string, exitCode int32, total uint64) error {
		sequence++
		return reporter.ReportRelayResponse(reportCtx, &sharedv1.RelayResponse{
			CorrelationId: request.GetCorrelationId(), Kind: kind, Sequence: sequence,
			Data: append([]byte(nil), data...), Reason: reason, ExitCode: exitCode, TotalBytes: total,
		})
	}
	if err := report(sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_ACCEPTED, nil, "", 0, 0); err != nil {
		if base.Err() == nil {
			c.logger.Printf("channel: relay %q acceptance report failed: %v", request.GetCorrelationId(), err)
		}
		return
	}

	maxBytes := request.GetMaxResponseBytes()
	if maxBytes == 0 {
		maxBytes = exec.DefaultRelayMaxResponseBytes
	}
	var reportErr error
	runnerOpts := []exec.Option{
		exec.WithClock(c.now),
	}
	if c.commandRunner != nil {
		runnerOpts = append(runnerOpts, exec.WithCommandRunner(c.commandRunner))
	} else {
		runnerOpts = append(runnerOpts, exec.WithCLIResolver(cliresolve.New(homeForVrooliBinary(c.cfg.VrooliBin))))
	}
	runner := exec.NewRunner(c.cfg.VrooliBin, c.cfg.WorkDir, nil, runnerOpts...)
	result := runner.ExecuteRelay(ctx, request, maxBytes, func(data []byte) {
		if reportErr != nil {
			return
		}
		reportedBytes += uint64(len(data))
		reportErr = report(sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_DATA, data, "", 0, reportedBytes)
		if reportErr != nil {
			cancel()
		}
	})
	if reportErr != nil {
		if base.Err() == nil {
			c.logger.Printf("channel: relay %q data report failed: %v", request.GetCorrelationId(), reportErr)
		}
		return
	}
	reason := result.Reason
	if result.LimitExceeded {
		reason = exec.RelayResponseLimitReason
	}
	if result.Cancelled {
		if requested := c.relayCancelReason(request.GetCorrelationId()); requested != "" {
			reason = requested
		}
		_ = report(sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_TERMINATED, nil, reason, int32(result.ExitCode), result.TotalBytes)
		return
	}
	if result.ExitCode == 0 {
		_ = report(sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_COMPLETED, nil, reason, 0, result.TotalBytes)
		return
	}
	_ = report(sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_FAILED, nil, reason, int32(result.ExitCode), result.TotalBytes)
}

func homeForVrooliBinary(binary string) string {
	binary = strings.TrimSpace(binary)
	if filepath.IsAbs(binary) {
		// Installed Vrooli binaries live at <home>/.vrooli/bin/vrooli.
		// Strip the binary, bin, and .vrooli components so cliresolve can
		// append its own stable .vrooli/bin suffix exactly once.
		return filepath.Dir(filepath.Dir(filepath.Dir(binary)))
	}
	return ""
}

type artifactUploader struct {
	rpc    artifacts_v1connect.ArtifactsServiceClient
	cred   *nodecred.Credential
	nodeID string
	now    func() time.Time
}

var _ exec.ArtifactUploader = (*artifactUploader)(nil)

func (u *artifactUploader) Upload(ctx context.Context, runID, name, mediaType string, data []byte) (string, error) {
	req := connect.NewRequest(&artifactsv1.UploadRunArtifactRequest{
		RunId: runID, Name: name, MediaType: mediaType, Data: data,
	})
	if u.cred != nil {
		for k, v := range u.cred.Headers(u.nodeID, u.now().UTC()) {
			req.Header().Set(k, v)
		}
	}
	resp, err := u.rpc.UploadRunArtifact(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Msg.GetArtifactRef(), nil
}

// runProvision executes a pushed privileged provisioning command via the
// separate privileged helper (internal/privsep), streaming status/log/version/
// exit ProvisionEvents back to the control plane's ProvisionService (each call
// signed with the node credential). It is launched in its own goroutine so a
// long provision does not block the SSE read loop. The runner package is NOT in
// this path — provisioning is structurally separate from job execution.
func (c *Client) runProvision(cmd *channelv1.ProvisionCommand) {
	ctx := c.baseCtx
	if ctx == nil {
		ctx = context.Background()
	}
	reporter := &provisionEventReporter{rpc: c.provisionRPC, cred: c.cred, nodeID: c.cfg.NodeID, now: c.now}
	if strings.TrimSpace(c.cfg.ProvisionSocket) == "" {
		c.reportProvisionUnavailable(ctx, reporter, cmd, "provisioning helper is not installed")
		return
	}
	err := privsep.Run(ctx, c.cfg.ProvisionSocket, c.cfg.ProvisionHelperUID, cmd, func(event *provisionv1.ProvisionEvent) error {
		return reporter.Report(ctx, event)
	})
	if err != nil && ctx.Err() == nil {
		c.logger.Printf("channel: provision %q: helper IPC failed: %v", cmd.GetOpId(), err)
		c.reportProvisionUnavailable(ctx, reporter, cmd, err.Error())
	}
}

func (c *Client) reportProvisionUnavailable(ctx context.Context, reporter *provisionEventReporter, cmd *channelv1.ProvisionCommand, reason string) {
	_ = reporter.Report(ctx, &provisionv1.ProvisionEvent{OpId: cmd.GetOpId(), Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_STATUS, Status: "failed: " + reason})
	_ = reporter.Report(ctx, &provisionv1.ProvisionEvent{OpId: cmd.GetOpId(), Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_EXIT, ExitCode: 127})
}

func (c *Client) runCleanup(cmd *channelv1.CleanupCommand) {
	ctx := c.baseCtxOrBackground()
	reporter := &cleanupEventReporter{rpc: c.cleanupRPC, cred: c.cred, nodeID: c.cfg.NodeID, now: c.now}
	if strings.TrimSpace(c.cfg.ProvisionSocket) == "" {
		_ = reporter.Report(ctx, &cleanupv1.CleanupEvent{OperationId: cmd.GetOpId(), Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_STATUS, Status: "blocked", Reason: "provisioning helper is not installed"})
		_ = reporter.Report(ctx, &cleanupv1.CleanupEvent{OperationId: cmd.GetOpId(), Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 127})
		return
	}
	err := privsep.RunCleanup(ctx, c.cfg.ProvisionSocket, c.cfg.ProvisionHelperUID, cmd, func(event *cleanupv1.CleanupEvent) error { return reporter.Report(ctx, event) })
	if err != nil && ctx.Err() == nil {
		c.logger.Printf("channel: cleanup %q: helper IPC failed: %v", cmd.GetOpId(), err)
		// Do not leave the durable cleanup operation in an apparent running
		// state when the local helper cannot be reached or its IPC stream is
		// malformed. The control plane needs a typed terminal observation so
		// the operator can repair/re-enroll and safely retry the frozen plan.
		_ = reporter.Report(ctx, &cleanupv1.CleanupEvent{
			OperationId: cmd.GetOpId(),
			Kind:        cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_STATUS,
			Status:      "failed",
			Reason:      "privileged helper IPC: " + err.Error(),
		})
		_ = reporter.Report(ctx, &cleanupv1.CleanupEvent{
			OperationId: cmd.GetOpId(),
			Kind:        cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT,
			ExitCode:    127,
		})
		return
	}
	if err == nil && c.shutdown != nil && privilegedops.Name(cmd.GetOperation()) == privilegedops.ApplyFrozenPlan {
		c.shutdown()
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

type relayResponseReporter struct {
	rpc    presence_v1connect.PresenceServiceClient
	cred   *nodecred.Credential
	nodeID string
	now    func() time.Time
}

var _ RelayResponseReporter = (*relayResponseReporter)(nil)

func (r *relayResponseReporter) ReportRelayResponse(ctx context.Context, response *sharedv1.RelayResponse) error {
	if r.rpc == nil {
		return errors.New("presence relay response client is unavailable")
	}
	req := connect.NewRequest(&presencev1.ReportRelayResponseRequest{Response: response})
	if r.cred != nil {
		for k, v := range r.cred.Headers(r.nodeID, r.now().UTC()) {
			req.Header().Set(k, v)
		}
	}
	_, err := r.rpc.ReportRelayResponse(ctx, req)
	return err
}

func (r *runEventReporter) Report(ctx context.Context, ev *sharedv1.RunEvent) error {
	req := connect.NewRequest(&runsv1.ReportRunEventRequest{Event: ev})
	if r.cred != nil {
		for k, v := range r.cred.Headers(r.nodeID, r.now().UTC()) {
			req.Header().Set(k, v)
		}
	}
	_, err := r.rpc.ReportRunEvent(ctx, req)
	return err
}

// provisionEventReporter implements privsep.Reporter by calling the control
// plane's ProvisionService.ReportProvisionEvent, signing each call with the
// node's per-node Ed25519 credential so the control plane verifies the node
// (and that it only reports against its own provisioning ops).
type provisionEventReporter struct {
	rpc    provision_v1connect.ProvisionServiceClient
	cred   *nodecred.Credential
	nodeID string
	now    func() time.Time
}

type cleanupEventReporter struct {
	rpc    cleanup_v1connect.CleanupServiceClient
	cred   *nodecred.Credential
	nodeID string
	now    func() time.Time
}

func (r *cleanupEventReporter) Report(ctx context.Context, ev *cleanupv1.CleanupEvent) error {
	req := connect.NewRequest(&cleanupv1.ReportCleanupEventRequest{Event: ev})
	if r.cred != nil {
		for k, v := range r.cred.Headers(r.nodeID, r.now().UTC()) {
			req.Header().Set(k, v)
		}
	}
	_, err := r.rpc.ReportCleanupEvent(ctx, req)
	return err
}

func (r *provisionEventReporter) Report(ctx context.Context, ev *provisionv1.ProvisionEvent) error {
	req := connect.NewRequest(&provisionv1.ReportProvisionEventRequest{Event: ev})
	if r.cred != nil {
		for k, v := range r.cred.Headers(r.nodeID, r.now().UTC()) {
			req.Header().Set(k, v)
		}
	}
	_, err := r.rpc.ReportProvisionEvent(ctx, req)
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
	health := snapshotToProto(c.sampler.Sample())
	// Surface the count of rejected (unsigned/mis-signed) control-plane pushes so
	// an operator sees a node being fed impostor frames (SECURITY.md boundary 2).
	// Only reported once non-zero so a healthy node's details stay clean.
	if rejected := c.rejectedFrames.Load(); rejected > 0 {
		if health.Details == nil {
			health.Details = map[string]string{}
		}
		health.Details["rejected_cp_frames"] = strconv.FormatUint(rejected, 10)
	}
	if rejected := c.rejectedCredentialPushes.Load(); rejected > 0 {
		if health.Details == nil {
			health.Details = map[string]string{}
		}
		health.Details["rejected_credential_pushes"] = strconv.FormatUint(rejected, 10)
	}
	hb := &sharedv1.Heartbeat{
		NodeId:                   c.cfg.NodeID,
		Sequence:                 *seq,
		Health:                   health,
		SentAt:                   timestamppb.New(now),
		RejectedCredentialPushes: c.rejectedCredentialPushes.Load(),
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
	if resp.Msg.GetCompatibility() == sharedv1.CompatibilityStatus_COMPATIBILITY_STATUS_NEEDS_UPDATE {
		c.logger.Printf("channel: heartbeat verdict NEEDS_UPDATE — update the agent to receive jobs")
	}
}

// snapshotToProto translates the agent's health.Snapshot into the wire
// HealthSnapshot.
func snapshotToProto(s health.Snapshot) *sharedv1.HealthSnapshot {
	out := &sharedv1.HealthSnapshot{
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
