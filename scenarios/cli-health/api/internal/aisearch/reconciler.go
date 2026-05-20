package aisearch

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// payloadHashKey is the field added to every vector-store payload so the
// reconciler can decide if an item needs re-embedding without comparing
// every field. Stays out of the hash input itself.
const payloadHashKey = "payload_hash"

// composePayloadHash returns a stable identifier for the (text, payload)
// pair so the reconciler can skip embedding when neither has changed.
func composePayloadHash(text string, payload map[string]interface{}) string {
	canon, _ := canonicalJSON(stripHashField(payload))
	h := sha256.New()
	_, _ = h.Write([]byte(text))
	_, _ = h.Write([]byte{'|'})
	_, _ = h.Write(canon)
	sum := h.Sum(nil)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func stripHashField(payload map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(payload))
	for k, v := range payload {
		if k == payloadHashKey {
			continue
		}
		out[k] = v
	}
	return out
}

func canonicalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(canonicalize(v))
}

func canonicalize(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]interface{}, len(x))
		for _, k := range keys {
			out[k] = canonicalize(x[k])
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, v := range x {
			out[i] = canonicalize(v)
		}
		return out
	default:
		return v
	}
}

// qdrantNamespace is the fixed UUID namespace used to derive point IDs from
// command full-paths. Same constant as prompt-manager so collisions across
// scenarios are impossible (they use different name prefixes).
var qdrantNamespace = [16]byte{
	0x6b, 0xa7, 0xb8, 0x10,
	0x9d, 0xad, 0x11, 0xd1,
	0x80, 0xb4, 0x00, 0xc0,
	0x4f, 0xd4, 0x30, 0xc8,
}

// PointIDForCommand returns the deterministic UUIDv5 used as the Qdrant point
// ID for a CommandRecord. The full path is the natural identity key.
func PointIDForCommand(fullPath string) string {
	name := strings.TrimSpace(fullPath)
	if name == "" {
		name = "unknown"
	}
	return uuidV5(qdrantNamespace, "cli-health:"+name)
}

func uuidV5(namespace [16]byte, name string) string {
	hash := sha1.New()
	_, _ = hash.Write(namespace[:])
	_, _ = hash.Write([]byte(name))
	sum := hash.Sum(nil)

	var uuid [16]byte
	copy(uuid[:], sum[:16])

	uuid[6] = (uuid[6] & 0x0f) | 0x50
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	hexStr := hex.EncodeToString(uuid[:])
	return hexStr[0:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16] + "-" + hexStr[16:20] + "-" + hexStr[20:32]
}

// ItemSnapshot is the opaque per-item payload a CollectionDescriptor carries
// from LoadAll through ComposeText/BuildPayload/PointID.
type ItemSnapshot any

// CollectionDescriptor parameterizes the Reconciler over each entity kind.
type CollectionDescriptor struct {
	Kind         EntityKind
	Store        VectorStore
	LoadAll      func(ctx context.Context) ([]ItemSnapshot, error)
	ComposeText  func(snap ItemSnapshot) string
	BuildPayload func(snap ItemSnapshot, embeddingText string) map[string]interface{}
	PointID      func(snap ItemSnapshot) string
	DisplayName  func(snap ItemSnapshot) string
}

// ErrReconcileBusy is returned by RunOnce when another reconcile is in flight.
var ErrReconcileBusy = errors.New("reconcile already in progress")

// Reconciler decides what qdrant work needs to happen across collections.
type Reconciler struct {
	Embedder    Embedder
	Descriptors []CollectionDescriptor
	Parallelism int
	Clock       func() time.Time

	mu         sync.Mutex
	running    bool
	cancel     context.CancelFunc
	lastPlan   *DriftReport
	lastResult *ApplyResult
	lastError  string
	canceled   bool
	startedAt  time.Time
	finishedAt time.Time
}

// NewReconciler wires up a Reconciler.
func NewReconciler(embedder Embedder, descriptors []CollectionDescriptor, parallelism int) *Reconciler {
	if parallelism <= 0 {
		parallelism = DefaultReconcileParallelism
	}
	if parallelism > MaxReconcileParallelism {
		parallelism = MaxReconcileParallelism
	}
	return &Reconciler{
		Embedder:    embedder,
		Descriptors: descriptors,
		Parallelism: parallelism,
		Clock:       time.Now,
	}
}

// Plan walks every descriptor and reports drift. Read-only.
func (r *Reconciler) Plan(ctx context.Context) (*DriftReport, error) {
	plannedAt := r.now()
	report := &DriftReport{
		PlannedAt:   plannedAt,
		Collections: make([]CollectionDriftReport, 0, len(r.Descriptors)),
	}

	for _, d := range r.Descriptors {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		colReport := CollectionDriftReport{Kind: d.Kind}

		items, err := d.LoadAll(ctx)
		if err != nil {
			log.Printf("[cli-health/aisearch] reconciler: LoadAll(%s) failed: %v", d.Kind, err)
			report.Collections = append(report.Collections, colReport)
			continue
		}

		stored, err := d.Store.ScrollIDs(ctx)
		if err != nil {
			log.Printf("[cli-health/aisearch] reconciler: ScrollIDs(%s) failed: %v", d.Kind, err)
			report.Collections = append(report.Collections, colReport)
			continue
		}

		seen := make(map[string]struct{}, len(items))
		for _, snap := range items {
			pid := d.PointID(snap)
			seen[pid] = struct{}{}

			text := d.ComposeText(snap)
			payload := d.BuildPayload(snap, text)
			hash, _ := payload[payloadHashKey].(string)

			existing, ok := stored[pid]
			switch {
			case !ok:
				colReport.ToUpsert = append(colReport.ToUpsert, ItemRef{
					Kind: d.Kind, PointID: pid, Name: d.DisplayName(snap), PayloadHash: hash, Snapshot: snap,
				})
			case existing.PayloadHash == "":
				colReport.LegacyCount++
				colReport.ToUpsert = append(colReport.ToUpsert, ItemRef{
					Kind: d.Kind, PointID: pid, Name: d.DisplayName(snap), PayloadHash: hash, Snapshot: snap,
				})
			case existing.PayloadHash == hash:
				colReport.UnchangedCount++
			default:
				colReport.ToUpsert = append(colReport.ToUpsert, ItemRef{
					Kind: d.Kind, PointID: pid, Name: d.DisplayName(snap), PayloadHash: hash, Snapshot: snap,
				})
			}
		}

		var ghosts []string
		for pid := range stored {
			if _, ok := seen[pid]; !ok {
				ghosts = append(ghosts, pid)
			}
		}
		sort.Strings(ghosts)
		colReport.ToDelete = ghosts

		report.Collections = append(report.Collections, colReport)
	}
	return report, nil
}

// Apply executes a previously computed plan.
func (r *Reconciler) Apply(ctx context.Context, plan *DriftReport) (*ApplyResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("apply: plan is required")
	}
	startedAt := r.now()
	result := &ApplyResult{
		StartedAt:   startedAt,
		Collections: make([]CollectionApplyResult, len(plan.Collections)),
	}
	for i, c := range plan.Collections {
		result.Collections[i] = CollectionApplyResult{Kind: c.Kind}
	}

	descByKind := make(map[EntityKind]CollectionDescriptor, len(r.Descriptors))
	for _, d := range r.Descriptors {
		descByKind[d.Kind] = d
	}

	var errMu sync.Mutex
	addErr := func(e ReconcileError) {
		errMu.Lock()
		defer errMu.Unlock()
		result.Errors = append(result.Errors, e)
	}

	parallelism := r.Parallelism
	if parallelism <= 0 {
		parallelism = 1
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(parallelism)

	for ci := range plan.Collections {
		ci := ci
		c := plan.Collections[ci]
		desc, ok := descByKind[c.Kind]
		if !ok {
			continue
		}
		for _, ref := range c.ToUpsert {
			ref := ref
			g.Go(func() error {
				if err := gctx.Err(); err != nil {
					addErr(ReconcileError{Kind: c.Kind, PointID: ref.PointID, Name: ref.Name, Op: "embed", Err: err.Error()})
					return nil
				}
				text := desc.ComposeText(ref.Snapshot)
				vec, err := r.Embedder.Embed(gctx, text)
				if err != nil {
					addErr(ReconcileError{Kind: c.Kind, PointID: ref.PointID, Name: ref.Name, Op: "embed", Err: err.Error()})
					return nil
				}
				payload := desc.BuildPayload(ref.Snapshot, text)
				if err := desc.Store.Upsert(gctx, ref.PointID, vec, payload); err != nil {
					addErr(ReconcileError{Kind: c.Kind, PointID: ref.PointID, Name: ref.Name, Op: "upsert", Err: err.Error()})
					return nil
				}
				errMu.Lock()
				result.Collections[ci].Upserted++
				errMu.Unlock()
				return nil
			})
		}
	}
	_ = g.Wait()

	for ci, c := range plan.Collections {
		if len(c.ToDelete) == 0 {
			continue
		}
		desc, ok := descByKind[c.Kind]
		if !ok {
			continue
		}
		if err := desc.Store.BatchDelete(ctx, c.ToDelete); err != nil {
			addErr(ReconcileError{Kind: c.Kind, Op: "delete", Err: err.Error()})
			continue
		}
		result.Collections[ci].Deleted = len(c.ToDelete)
	}

	result.FinishedAt = r.now()
	return result, nil
}

// RunOnce composes Plan + Apply with singleton semantics.
func (r *Reconciler) RunOnce(ctx context.Context) (*DriftReport, *ApplyResult, error) {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil, nil, ErrReconcileBusy
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.running = true
	r.cancel = cancel
	r.canceled = false
	r.startedAt = r.now()
	r.lastError = ""
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.running = false
		r.cancel = nil
		r.finishedAt = r.now()
		r.mu.Unlock()
		cancel()
	}()

	plan, err := r.Plan(runCtx)
	if err != nil {
		r.mu.Lock()
		r.lastError = err.Error()
		r.mu.Unlock()
		return nil, nil, err
	}
	r.mu.Lock()
	r.lastPlan = plan
	r.mu.Unlock()

	apply, err := r.Apply(runCtx, plan)
	r.mu.Lock()
	r.lastResult = apply
	if err != nil {
		r.lastError = err.Error()
	}
	r.mu.Unlock()
	return plan, apply, err
}

// Cancel aborts an in-flight RunOnce.
func (r *Reconciler) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.canceled = true
		r.cancel()
	}
}

// Status returns a snapshot of the Reconciler's last-known state.
func (r *Reconciler) Status() ReconcileStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := ReconcileStatus{
		Running:    r.running,
		LastPlan:   r.lastPlan,
		LastResult: r.lastResult,
		LastError:  r.lastError,
		Canceled:   r.canceled,
	}
	if !r.startedAt.IsZero() {
		st.StartedAt = r.startedAt.Format(time.RFC3339)
	}
	if !r.finishedAt.IsZero() && !r.running {
		st.FinishedAt = r.finishedAt.Format(time.RFC3339)
	}
	return st
}

func (r *Reconciler) now() time.Time {
	if r.Clock != nil {
		return r.Clock()
	}
	return time.Now()
}
