package aisearch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"prompt-manager/internal/skills"
	"prompt-manager/internal/store"

	"golang.org/x/sync/errgroup"
)

// ItemSnapshot is the opaque per-item payload a CollectionDescriptor carries
// from LoadAll through ComposeText/BuildPayload/PointID. Each descriptor
// type-asserts back to its concrete shape.
type ItemSnapshot any

// CollectionDescriptor parameterizes the Reconciler over each entity kind.
// One descriptor per qdrant collection; the Reconciler iterates a list.
type CollectionDescriptor struct {
	Kind  EntityKind
	Store VectorStore
	// LoadAll returns every on-disk item for this collection.
	LoadAll func(ctx context.Context) ([]ItemSnapshot, error)
	// ComposeText returns the embedding-input text for one snapshot.
	ComposeText func(snap ItemSnapshot) string
	// BuildPayload returns the full qdrant payload (including payload_hash)
	// for one snapshot at the given embedding text.
	BuildPayload func(snap ItemSnapshot, embeddingText string) map[string]interface{}
	// PointID returns the qdrant point ID (UUIDv5) for one snapshot.
	PointID func(snap ItemSnapshot) string
	// DisplayName is a human-readable label used in error reports.
	DisplayName func(snap ItemSnapshot) string
}

// ErrReconcileBusy is returned by RunOnce when another reconcile is already
// in flight. Callers (the SyncLoop) treat this as a no-op success.
var ErrReconcileBusy = errors.New("reconcile already in progress")

// Reconciler is the single component that decides what qdrant work needs to
// happen across one or more collections.
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

// NewReconciler wires up a Reconciler. parallelism is clamped to
// [1, MaxReconcileParallelism].
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

// Plan walks every descriptor and reports what needs upserting and what needs
// deleting. Plan is read-only; Apply mutates qdrant.
//
// A failure in one descriptor does not abort the rest — the failure surfaces
// later as a ReconcileError on Apply.
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

		// Per-collection failures are stashed as an empty CollectionDriftReport
		// so the kind is still represented in the plan; Apply records errors
		// for downstream visibility.
		colReport := CollectionDriftReport{Kind: d.Kind}

		items, err := d.LoadAll(ctx)
		if err != nil {
			log.Printf("[aisearch] reconciler: LoadAll(%s) failed: %v", d.Kind, err)
			report.Collections = append(report.Collections, colReport)
			continue
		}

		stored, err := d.Store.ScrollIDs(ctx)
		if err != nil {
			log.Printf("[aisearch] reconciler: ScrollIDs(%s) failed: %v", d.Kind, err)
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
					Kind:        d.Kind,
					PointID:     pid,
					Name:        d.DisplayName(snap),
					PayloadHash: hash,
					Snapshot:    snap,
				})
			case existing.PayloadHash == "":
				colReport.LegacyCount++
				colReport.ToUpsert = append(colReport.ToUpsert, ItemRef{
					Kind:        d.Kind,
					PointID:     pid,
					Name:        d.DisplayName(snap),
					PayloadHash: hash,
					Snapshot:    snap,
				})
			case existing.PayloadHash == hash:
				colReport.UnchangedCount++
			default:
				colReport.ToUpsert = append(colReport.ToUpsert, ItemRef{
					Kind:        d.Kind,
					PointID:     pid,
					Name:        d.DisplayName(snap),
					PayloadHash: hash,
					Snapshot:    snap,
				})
			}
		}

		// Anything in qdrant but not on disk is a ghost.
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
//
// Per-item failures become ReconcileError entries; the run continues to make
// progress on the rest. Per-collection scroll/load failures from Plan are
// already represented as empty CollectionDriftReports in the plan.
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

	// Upserts run in parallel across all collections. Embedding is the
	// expensive step; bounding global concurrency keeps Ollama from being
	// overrun.
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

	// Deletes happen after upserts. One BatchDelete per collection.
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

// RunOnce composes Plan + Apply with singleton semantics. A second concurrent
// caller receives ErrReconcileBusy.
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

// Cancel aborts an in-flight RunOnce. No-op when nothing is running.
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

// --- Descriptor constructors -------------------------------------------------

// skillSnap is the descriptor-private snapshot for a skill: metadata + the
// folder it lives in + its full content.
type skillSnap struct {
	Meta    *skills.Metadata
	Folder  string
	Content string
}

// NewSkillDescriptor wires a CollectionDescriptor for the skill collection.
// The descriptor reads every skill via skillStore.GetAll + GetContent.
func NewSkillDescriptor(store VectorStore, skillStore skills.SkillStore) CollectionDescriptor {
	return CollectionDescriptor{
		Kind:  KindSkill,
		Store: store,
		LoadAll: func(ctx context.Context) ([]ItemSnapshot, error) {
			all, err := skillStore.GetAll()
			if err != nil {
				return nil, err
			}
			out := make([]ItemSnapshot, 0, len(all))
			for i := range all {
				meta := all[i]
				folder, filename := extractFolderAndFile(meta.File)
				if folder == "" {
					continue
				}
				content, err := skillStore.GetContent(folder, filename)
				if err != nil {
					log.Printf("[aisearch] reconciler: skill %s content read failed: %v", meta.ID, err)
					continue
				}
				m := meta // independent copy
				out = append(out, &skillSnap{Meta: &m, Folder: folder, Content: content})
			}
			return out, nil
		},
		ComposeText: func(snap ItemSnapshot) string {
			ss := snap.(*skillSnap)
			return composeEmbeddingText(ss.Meta, ss.Content)
		},
		BuildPayload: func(snap ItemSnapshot, text string) map[string]interface{} {
			ss := snap.(*skillSnap)
			return buildSkillPayload(ss.Meta, ss.Folder, text)
		},
		PointID: func(snap ItemSnapshot) string {
			return qdrantPointID(snap.(*skillSnap).Meta.ID)
		},
		DisplayName: func(snap ItemSnapshot) string {
			return snap.(*skillSnap).Meta.Name
		},
	}
}

type agentSnap struct {
	Agent *store.Agent
	Prose string
}

// NewAgentDescriptor wires a CollectionDescriptor for the agent collection.
// proseReader may be nil — when nil, the agent's standing prose is omitted from
// the embedding text and only its metadata is indexed.
func NewAgentDescriptor(vstore VectorStore, agentStore AgentStoreReader) CollectionDescriptor {
	proseReader, _ := agentStore.(AgentProseReader)
	return CollectionDescriptor{
		Kind:  KindAgent,
		Store: vstore,
		LoadAll: func(ctx context.Context) ([]ItemSnapshot, error) {
			agents, err := agentStore.List(ctx)
			if err != nil {
				return nil, err
			}
			out := make([]ItemSnapshot, 0, len(agents))
			for i := range agents {
				a := agents[i]
				snap := &agentSnap{Agent: &a}
				if proseReader != nil {
					if c, err := proseReader.GetProse(ctx, a.ID); err == nil {
						snap.Prose = c
					}
				}
				out = append(out, snap)
			}
			return out, nil
		},
		ComposeText: func(snap ItemSnapshot) string {
			as := snap.(*agentSnap)
			return composeAgentEmbeddingText(as.Agent, as.Prose)
		},
		BuildPayload: func(snap ItemSnapshot, text string) map[string]interface{} {
			return buildAgentPayload(snap.(*agentSnap).Agent, text)
		},
		PointID: func(snap ItemSnapshot) string {
			return agentPointID(snap.(*agentSnap).Agent.ID)
		},
		DisplayName: func(snap ItemSnapshot) string {
			return snap.(*agentSnap).Agent.DisplayName
		},
	}
}

type teamSnap struct {
	Team        *store.Team
	MemberNames []string
}

// NewTeamDescriptor wires a CollectionDescriptor for the team collection.
// relStore may be nil — member names are then omitted from embedding text.
func NewTeamDescriptor(vstore VectorStore, teamStore TeamStoreReader, relStore TeamRelReader) CollectionDescriptor {
	return CollectionDescriptor{
		Kind:  KindTeam,
		Store: vstore,
		LoadAll: func(ctx context.Context) ([]ItemSnapshot, error) {
			teams, err := teamStore.List(ctx)
			if err != nil {
				return nil, err
			}
			out := make([]ItemSnapshot, 0, len(teams))
			for i := range teams {
				tm := teams[i]
				snap := &teamSnap{Team: &tm}
				if relStore != nil {
					if members, err := relStore.ListTeamMembers(ctx, tm.ID); err == nil {
						for _, m := range members {
							snap.MemberNames = append(snap.MemberNames, m.AgentID)
						}
					}
				}
				out = append(out, snap)
			}
			return out, nil
		},
		ComposeText: func(snap ItemSnapshot) string {
			ts := snap.(*teamSnap)
			return composeTeamEmbeddingText(ts.Team, ts.MemberNames)
		},
		BuildPayload: func(snap ItemSnapshot, text string) map[string]interface{} {
			ts := snap.(*teamSnap)
			return buildTeamPayload(ts.Team, len(ts.MemberNames), text)
		},
		PointID: func(snap ItemSnapshot) string {
			return teamPointID(snap.(*teamSnap).Team.ID)
		},
		DisplayName: func(snap ItemSnapshot) string {
			return snap.(*teamSnap).Team.DisplayName
		},
	}
}

type topicSnap struct {
	Topic   *store.Topic
	Content string
}

// NewTopicDescriptor wires a CollectionDescriptor for the topic collection.
func NewTopicDescriptor(vstore VectorStore, topicStore TopicStoreReader) CollectionDescriptor {
	return CollectionDescriptor{
		Kind:  KindTopic,
		Store: vstore,
		LoadAll: func(ctx context.Context) ([]ItemSnapshot, error) {
			topics, err := topicStore.List(ctx)
			if err != nil {
				return nil, err
			}
			out := make([]ItemSnapshot, 0, len(topics))
			for i := range topics {
				t := topics[i]
				topic, content, err := topicStore.GetWithContent(ctx, t.ID)
				if err != nil {
					log.Printf("[aisearch] reconciler: topic %s read failed: %v", t.ID, err)
					continue
				}
				out = append(out, &topicSnap{Topic: topic, Content: content})
			}
			return out, nil
		},
		ComposeText: func(snap ItemSnapshot) string {
			ts := snap.(*topicSnap)
			return composeTopicEmbeddingText(ts.Topic, ts.Content)
		},
		BuildPayload: func(snap ItemSnapshot, text string) map[string]interface{} {
			return buildTopicPayload(snap.(*topicSnap).Topic, text)
		},
		PointID: func(snap ItemSnapshot) string {
			return topicPointID(snap.(*topicSnap).Topic.ID)
		},
		DisplayName: func(snap ItemSnapshot) string {
			return snap.(*topicSnap).Topic.Name
		},
	}
}

type actionSnap struct {
	Action *store.Action
}

// NewActionDescriptor wires a CollectionDescriptor for the action collection.
func NewActionDescriptor(vstore VectorStore, actionStore store.ActionStore) CollectionDescriptor {
	return CollectionDescriptor{
		Kind:  KindAction,
		Store: vstore,
		LoadAll: func(ctx context.Context) ([]ItemSnapshot, error) {
			actions, err := actionStore.List(ctx)
			if err != nil {
				return nil, err
			}
			out := make([]ItemSnapshot, 0, len(actions))
			for i := range actions {
				a := actions[i]
				out = append(out, &actionSnap{Action: &a})
			}
			return out, nil
		},
		ComposeText: func(snap ItemSnapshot) string {
			return composeActionEmbeddingText(snap.(*actionSnap).Action)
		},
		BuildPayload: func(snap ItemSnapshot, text string) map[string]interface{} {
			return buildActionPayload(snap.(*actionSnap).Action, text)
		},
		PointID: func(snap ItemSnapshot) string {
			return actionPointID(snap.(*actionSnap).Action.ID)
		},
		DisplayName: func(snap ItemSnapshot) string {
			return snap.(*actionSnap).Action.Name
		},
	}
}
