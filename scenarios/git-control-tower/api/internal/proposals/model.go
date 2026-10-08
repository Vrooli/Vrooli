// Package proposals owns commit proposals: editable drafts bound to exact
// repository content. Anchoring, creating, editing, refreshing and
// withdrawing write only GCT's database. Apply is human-only: it stages
// exactly the proposal paths and commits the rendered message under a
// consumed single-use intent for operation "repo.apply_proposal".
package proposals

import (
	"context"
	"errors"
	"time"

	"git-control-tower/internal/trailers"
)

// OperationApply is the human-control operation that authorizes Apply.
const OperationApply = "repo.apply_proposal"

type State string

const (
	StateOpen       State = "open"
	StateCommitted  State = "committed"
	StateWithdrawn  State = "withdrawn"
	StateSuperseded State = "superseded"
)

type ChangeKind string

const (
	KindAdded    ChangeKind = "added"
	KindModified ChangeKind = "modified"
	KindDeleted  ChangeKind = "deleted"
)

// Flag codes. Stored flags come from create or refresh; dynamic flags are
// recomputed on every read because they depend on other records or the
// index.
const (
	FlagMixedPrior         = "mixed_prior_uncommitted"
	FlagOtherOpenProposal  = "other_open_proposal"
	FlagSandboxPending     = "sandbox_pending"
	FlagAlreadyStaged      = "already_staged"
	FlagNoAnchor           = "no_anchor"
	FlagOutsideAnchorScope = "outside_anchor_scope"
)

// Freshness states. Freshness is computed on read and never stored.
const (
	FreshnessFresh         = "fresh"
	FreshnessDrifted       = "drifted"
	FreshnessBaseMoved     = "base_moved"
	FreshnessUnknown       = "unknown"
	FreshnessNotApplicable = "not_applicable"
)

// Refusal codes for Apply. A refusal means nothing was staged or committed.
const (
	RefuseContentDrift  = "content_drift"
	RefuseBaseMoved     = "base_moved"
	RefuseForeignStaged = "foreign_staged"
	RefuseState         = "state"
	RefuseRevision      = "revision"
	RefuseIndexMismatch = "index_mismatch"
	RefusePrecommit     = "precommit_failed"
)

// DeletedBlob marks an absent working-tree file in current-content maps.
const DeletedBlob = "deleted"

// DirectoryBlob marks a path that is a directory or submodule. Proposals
// hold files only.
const DirectoryBlob = "directory"

var (
	ErrNotFound   = errors.New("proposal not found")
	ErrConflict   = errors.New("proposal changed; reload it and retry")
	ErrNotOpen    = errors.New("proposal is not open")
	ErrInvalid    = errors.New("invalid proposal request")
	ErrForbidden  = errors.New("verified human authority with a consumed apply intent is required")
	ErrValidation = errors.New("proposal message has validation errors")
)

type Actor struct {
	Subject  string `json:"subject,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Verified bool   `json:"verified,omitempty"`
	RunID    string `json:"run_id,omitempty"`
}

// IsHuman reports whether the actor is a verified human.
func (a Actor) IsHuman() bool { return a.Verified && a.Kind == "human" }

type Work struct {
	EffortRef      string   `json:"effort_ref,omitempty"`
	EffortRevision string   `json:"effort_revision,omitempty"`
	Epoch          string   `json:"epoch,omitempty"`
	RunIDs         []string `json:"run_ids,omitempty"`
	Plans          []string `json:"plans,omitempty"`
}

type Message struct {
	Subject        string           `json:"subject"`
	Body           string           `json:"body,omitempty"`
	Trailers       []trailers.Entry `json:"trailers,omitempty"`
	Rendered       string           `json:"rendered"`
	OperatorEdited bool             `json:"operator_edited,omitempty"`
}

type Flag struct {
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

type File struct {
	Path    string     `json:"path"`
	Kind    ChangeKind `json:"kind"`
	SHA256  string     `json:"sha256,omitempty"`
	BlobID  string     `json:"blob_id,omitempty"`
	Deleted bool       `json:"deleted,omitempty"`
	Source  string     `json:"source"`
	Flags   []Flag     `json:"flags,omitempty"`
}

// ContentID is the comparable identity of the file's proposed content.
func (f File) ContentID() string {
	if f.Deleted {
		return DeletedBlob
	}
	return f.BlobID
}

type Exclusion struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

type Evidence struct {
	AnchorID           string   `json:"anchor_id,omitempty"`
	EpochFile          string   `json:"epoch_file,omitempty"`
	AcceptedLineSHA256 string   `json:"accepted_line_sha256,omitempty"`
	GateReceipts       []string `json:"gate_receipts,omitempty"`
	MetricBefore       string   `json:"metric_before,omitempty"`
	MetricAfter        string   `json:"metric_after,omitempty"`
}

type Event struct {
	Action    string    `json:"action"`
	Actor     Actor     `json:"actor"`
	Revision  int       `json:"revision"`
	Detail    string    `json:"detail,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type Proposal struct {
	ID           string      `json:"id"`
	RepositoryID string      `json:"repository_id"`
	State        State       `json:"state"`
	Revision     int         `json:"revision"`
	CreatedBy    Actor       `json:"created_by"`
	BaseHead     string      `json:"base_head,omitempty"`
	Branch       string      `json:"branch,omitempty"`
	Work         Work        `json:"work"`
	Message      Message     `json:"message"`
	Files        []File      `json:"files"`
	Excluded     []Exclusion `json:"excluded,omitempty"`
	Evidence     Evidence    `json:"evidence"`
	Digest       string      `json:"digest"`
	CommitOID    string      `json:"commit_oid,omitempty"`
	SupersededBy string      `json:"superseded_by,omitempty"`
	Events       []Event     `json:"events,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	CommittedAt  *time.Time  `json:"committed_at,omitempty"`
}

// Paths returns the proposal paths in file order.
func (p Proposal) Paths() []string {
	paths := make([]string, 0, len(p.Files))
	for _, file := range p.Files {
		paths = append(paths, file.Path)
	}
	return paths
}

type AnchorFile struct {
	Path    string `json:"path"`
	SHA256  string `json:"sha256,omitempty"`
	BlobID  string `json:"blob_id,omitempty"`
	Deleted bool   `json:"deleted,omitempty"`
}

// ContentID is the comparable identity of the anchored content.
func (f AnchorFile) ContentID() string {
	if f.Deleted {
		return DeletedBlob
	}
	return f.BlobID
}

type Anchor struct {
	ID           string       `json:"id"`
	RepositoryID string       `json:"repository_id"`
	EffortRef    string       `json:"effort_ref,omitempty"`
	Epoch        string       `json:"epoch,omitempty"`
	Scopes       []string     `json:"scopes"`
	Head         string       `json:"head,omitempty"`
	Files        []AnchorFile `json:"files"`
	CreatedBy    Actor        `json:"created_by"`
	CreatedAt    time.Time    `json:"created_at"`
}

// Freshness is the live comparison of an open proposal with the repository.
type Freshness struct {
	State         string            `json:"state"`
	Drifted       []string          `json:"drifted,omitempty"`
	BaseChanged   []string          `json:"base_changed,omitempty"`
	ForeignStaged []string          `json:"foreign_staged,omitempty"`
	CurrentHead   string            `json:"current_head,omitempty"`
	Detail        string            `json:"detail,omitempty"`
	CheckedAt     time.Time         `json:"checked_at"`
	CurrentBlobs  map[string]string `json:"current_blobs,omitempty"`
}

// View is a proposal with read-time facts: freshness, dynamic flags and the
// validation issues of its message.
type View struct {
	Proposal
	Freshness Freshness        `json:"freshness"`
	Issues    []trailers.Issue `json:"issues,omitempty"`
}

// Content is the working-tree identity of one path.
type Content struct {
	BlobID  string
	SHA256  string
	Deleted bool
}

// ContentID is the comparable identity of the content.
func (c Content) ContentID() string {
	if c.Deleted {
		return DeletedBlob
	}
	return c.BlobID
}

// StatusEntry is one dirty path. XY is the porcelain v2 status pair, with
// "??" for untracked paths.
type StatusEntry struct {
	Path string
	XY   string
}

// IndexEntry is one stage-0 index entry.
type IndexEntry struct {
	Mode   string
	BlobID string
}

// CommitFacts describe a created commit for post-commit verification. Blobs
// maps each changed path to its new blob ID, or DeletedBlob.
type CommitFacts struct {
	Blobs   map[string]string
	Message string
}

// Git is the narrow repository seam proposals need. Production adapts the
// git binary; tests use an in-memory repository model. Only Stage and
// RestoreIndex write, and only Apply calls them.
type Git interface {
	Head(ctx context.Context) (string, error)
	Branch(ctx context.Context) (string, error)
	Status(ctx context.Context) ([]StatusEntry, error)
	Hash(ctx context.Context, paths []string) (map[string]Content, error)
	IndexEntries(ctx context.Context, paths []string) (map[string]IndexEntry, error)
	StagedPaths(ctx context.Context) ([]string, error)
	ChangedPaths(ctx context.Context, from, to string, paths []string) ([]string, error)
	Ignored(ctx context.Context, paths []string) ([]string, error)
	ObjectExists(ctx context.Context, oid string) (bool, error)
	Stage(ctx context.Context, paths []string) error
	RestoreIndex(ctx context.Context, paths []string, prior map[string]IndexEntry) error
	CommitFacts(ctx context.Context, oid string) (CommitFacts, error)
}

// SandboxPending returns, per path, the Workspace Sandbox runs whose applied
// changes are still pending commit. Nil means unavailable.
type SandboxPending func(ctx context.Context) (map[string][]string, error)

// Repo identifies the repository an operation runs against.
type Repo struct {
	ID      string
	Git     Git
	Sandbox SandboxPending
}
