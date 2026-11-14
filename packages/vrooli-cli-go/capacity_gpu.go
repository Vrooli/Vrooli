package vroolicli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultGPUCapacityTTL = 15 * time.Minute
const gpuClaimCleanupTimeout = 5 * time.Second

// GPUIdentity is a stable observed device identity plus the current CLI index.
type GPUIdentity struct {
	Index int
	UUID  string
	Name  string
}

// GPUCapacityRequest reserves a model/profile VRAM budget. UUID is mandatory;
// the index is read from fresh inventory. Reclaim is explicit and independent
// of scheduling priority; the default preserves other resident owners.
type GPUCapacityRequest struct {
	OwnerID      string
	GPUUUID      string
	VRAMBytes    int64
	TTL          time.Duration
	Priority     string
	AllowReclaim bool
}

// GPUCapacityLease owns one exact operation claim until Release succeeds.
type GPUCapacityLease struct {
	mu           sync.Mutex
	client       *Client
	identity     GPUIdentity
	claimID      string
	generation   int64
	ttl          time.Duration
	released     bool
	warnings     []string
	reclaimBytes int64
	enforce      string
}

func (l *GPUCapacityLease) Identity() GPUIdentity {
	if l == nil {
		return GPUIdentity{}
	}
	return l.identity
}

// ClaimID returns the exact ledger row owned by this lease for diagnostics.
func (l *GPUCapacityLease) ClaimID() string {
	if l == nil {
		return ""
	}
	return l.claimID
}

// ErrGPUCapacityUnavailable is a valid broker denial, not a transport/protocol
// failure. Callers may choose an explicitly supported CPU fallback on this error.
var ErrGPUCapacityUnavailable = errors.New("GPU capacity unavailable")

type GPUCapacityDenied struct {
	Reason       string
	Warnings     []string
	ReclaimBytes int64
	Enforce      string
}

func (e *GPUCapacityDenied) Error() string { return "GPU capacity unavailable: " + e.Reason }
func (e *GPUCapacityDenied) Unwrap() error { return ErrGPUCapacityUnavailable }

func (l *GPUCapacityLease) Generation() int64 {
	if l == nil {
		return 0
	}
	return l.generation
}
func (l *GPUCapacityLease) Warnings() []string {
	if l == nil {
		return nil
	}
	return append([]string(nil), l.warnings...)
}
func (l *GPUCapacityLease) ReclaimBytes() int64 {
	if l == nil {
		return 0
	}
	return l.reclaimBytes
}
func (l *GPUCapacityLease) Enforce() string {
	if l == nil {
		return ""
	}
	return l.enforce
}

// AcquireGPUCapacity verifies the requested UUID against fresh host inventory,
// then asks the shared capacity owner to admit an exact-index protected claim.
func (c *Client) AcquireGPUCapacity(ctx context.Context, req GPUCapacityRequest) (*GPUCapacityLease, error) {
	if c == nil || strings.TrimSpace(req.OwnerID) == "" || strings.TrimSpace(req.GPUUUID) == "" || req.VRAMBytes <= 0 {
		return nil, errors.New("GPU capacity requires client, owner, exact UUID, and positive measured bytes")
	}
	priority := req.Priority
	if priority == "" {
		priority = "interactive"
	}
	if priority != "interactive" && priority != "service" && priority != "batch" {
		return nil, errors.New("GPU capacity priority must be interactive, service or batch")
	}
	if req.TTL < 0 {
		return nil, errors.New("GPU capacity TTL cannot be negative")
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	inventory, err := c.HostInventory(ctx)
	if err != nil {
		return nil, fmt.Errorf("read GPU inventory before capacity admission: %w", err)
	}
	var identity GPUIdentity
	for _, gpu := range inventory.GetGpus() {
		if gpu.GetUuid() != req.GPUUUID {
			continue
		}
		if identity.UUID != "" {
			return nil, errors.New("host inventory contains duplicate GPU UUID")
		}
		identity = GPUIdentity{Index: int(gpu.GetIndex()), UUID: gpu.GetUuid(), Name: gpu.GetName()}
	}
	if identity.UUID == "" {
		return nil, errors.New("requested GPU UUID is absent from host inventory")
	}
	ttl := req.TTL
	if ttl <= 0 {
		ttl = defaultGPUCapacityTTL
	}
	args := []string{
		"capacity", "claim", "--owner-kind", "op", "--owner-id", req.OwnerID,
		"--resource-kind", "vram", "--gpu-index", strconv.Itoa(identity.Index),
		"--preferred", strconv.FormatInt(req.VRAMBytes, 10), "--floor", strconv.FormatInt(req.VRAMBytes, 10),
		"--priority", priority,
	}
	if !req.AllowReclaim {
		args = append(args, "--no-reclaim")
	}
	args = append(args, "--protected", "--ttl", ttl.String(), "--json")
	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("capacity GPU admission: %w", err)
	}
	var response struct {
		Verdict struct {
			Kind         string   `json:"kind"`
			Reason       string   `json:"reason"`
			Warnings     []string `json:"warnings"`
			ReclaimBytes int64    `json:"reclaim_bytes"`
		} `json:"verdict"`
		Claim struct {
			ClaimID      string `json:"claim_id"`
			Generation   int64  `json:"generation"`
			GPUIndex     *int   `json:"gpu_index"`
			OwnerID      string `json:"owner_id"`
			ResourceKind string `json:"resource_kind"`
			AmountBytes  int64  `json:"amount_bytes"`
			Status       string `json:"status"`
		} `json:"claim"`
		Enforce string `json:"enforce"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return nil, fmt.Errorf("decode capacity GPU admission: %w", err)
	}
	unavailable := response.Verdict.Kind != "grant"
	invalid := response.Claim.ClaimID == "" || response.Claim.Generation <= 0 || response.Claim.GPUIndex == nil || *response.Claim.GPUIndex != identity.Index || response.Claim.OwnerID != req.OwnerID || response.Claim.ResourceKind != "vram" || response.Claim.AmountBytes < req.VRAMBytes || response.Claim.Status != "granted"
	if response.Verdict.Kind == "grant" && response.Verdict.ReclaimBytes > 0 && (!req.AllowReclaim || response.Enforce != "on") {
		unavailable = true
		response.Verdict.Reason = "grant requires reclaim that was not permitted or enforced"
	}
	if unavailable || invalid {
		// Denied/advisory claims are retained by the owner too. Return capacity before
		// any caller can choose fallback; a cleanup failure remains a hard error.
		if response.Claim.ClaimID != "" {
			cleanupCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), gpuClaimCleanupTimeout)
			cleanupErr := c.releaseCapacityClaim(cleanupCtx, response.Claim.ClaimID)
			stop()
			if cleanupErr != nil {
				return nil, fmt.Errorf("GPU admission rejected; cleanup claim %s failed (TTL recovery within %s): %w", response.Claim.ClaimID, ttl, cleanupErr)
			}
		}
		if unavailable {
			return nil, &GPUCapacityDenied{Reason: response.Verdict.Reason, Warnings: response.Verdict.Warnings, ReclaimBytes: response.Verdict.ReclaimBytes, Enforce: response.Enforce}
		}
		return nil, errors.New("GPU capacity grant returned an invalid claim identity, state or budget")
	}
	// The ledger identifies GPUs by index. Reconfirm the physical identity after
	// admission so a hotplug/reorder between inventory and claim cannot account
	// for one device while the worker binds another UUID.
	confirmed, confirmErr := c.HostInventory(ctx)
	count := 0
	if confirmErr == nil {
		for _, gpu := range confirmed.GetGpus() {
			if gpu.GetUuid() == identity.UUID && int(gpu.GetIndex()) == identity.Index {
				count++
			}
		}
	}
	if confirmErr != nil || count != 1 {
		cleanupCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), gpuClaimCleanupTimeout)
		cleanupErr := c.releaseCapacityClaim(cleanupCtx, response.Claim.ClaimID)
		stop()
		if cleanupErr != nil {
			return nil, fmt.Errorf("GPU identity changed or could not be confirmed; cleanup claim %s failed (TTL %s): %w", response.Claim.ClaimID, ttl, cleanupErr)
		}
		return nil, fmt.Errorf("GPU identity changed or could not be confirmed after admission: %v", confirmErr)
	}
	return &GPUCapacityLease{client: c, identity: identity, claimID: response.Claim.ClaimID, generation: response.Claim.Generation, ttl: ttl, warnings: response.Verdict.Warnings, reclaimBytes: response.Verdict.ReclaimBytes, enforce: response.Enforce}, nil
}

// Heartbeat renews the exact captured claim generation and never recreates it.
func (l *GPUCapacityLease) Heartbeat(ctx context.Context) error {
	if l == nil || l.client == nil {
		return errors.New("GPU capacity lease is unavailable")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return errors.New("GPU capacity lease has been released")
	}
	args := []string{"capacity", "heartbeat", "--claim-id", l.claimID, "--generation", strconv.FormatInt(l.generation, 10), "--ttl", l.ttl.String(), "--json"}
	ctx, cancel := l.client.withTimeout(ctx)
	defer cancel()
	out, err := l.client.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("heartbeat GPU capacity claim %s: %w", l.claimID, err)
	}
	var response struct {
		ClaimID    string `json:"claim_id"`
		Generation int64  `json:"generation"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return fmt.Errorf("decode GPU capacity heartbeat: %w", err)
	}
	if response.ClaimID != l.claimID || response.Generation != l.generation || response.Status != "granted" {
		return errors.New("GPU capacity heartbeat returned a mismatched claim identity, generation, or active status")
	}
	return nil
}

// Release releases the exact operation claim. A failed release remains visible
// to the caller and the lease can be retried.
func (l *GPUCapacityLease) Release(ctx context.Context) error {
	if l == nil || l.client == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return nil
	}
	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), gpuClaimCleanupTimeout)
	defer cleanupCancel()
	if err := l.client.releaseCapacityClaim(cleanupCtx, l.claimID); err != nil {
		return fmt.Errorf("%w (TTL recovery within %s)", err, l.ttl)
	}
	l.released = true
	return nil
}

func (c *Client) releaseCapacityClaim(ctx context.Context, claimID string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	if _, err := c.run(ctx, "capacity", "release", "--claim-id", claimID, "--json"); err != nil {
		return fmt.Errorf("release capacity claim %s: %w", claimID, err)
	}
	return nil
}

var _ interface {
	Identity() GPUIdentity
	Heartbeat(context.Context) error
	Release(context.Context) error
} = (*GPUCapacityLease)(nil)
