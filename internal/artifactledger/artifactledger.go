// Package artifactledger makes every removal from a Vrooli install root
// attributable.
//
// # Why this exists
//
// Installed CLI binaries were disappearing from ~/.vrooli/bin during normal
// operation, repeatedly, for months. Each investigation had to reconstruct what
// happened from the outside -- directory watchers, process tables, forensic
// comparison against the install record -- because nothing in the system
// recorded that a removal had occurred, who performed it, or which rule fired.
// internal/buildinfo already documents the symptom, calling the cause "a
// foreign remover" and shipping a recovery copy instead of a diagnosis.
//
// Attribution is the fix for that. A removal that leaves a receipt can be
// traced in one step instead of one afternoon.
//
// # Why the receipt is written first
//
// The intent record is written *before* the unlink, so a process that dies
// mid-removal is still attributable. A crash is exactly the case where the
// evidence matters most and exactly the case a write-after would lose. The
// outcome record follows and closes the pair.
//
// A removal whose intent cannot be recorded does not happen. Attribution is a
// precondition, not a courtesy: refusing to delete is the safe direction, and
// an unattributable deletion is what produced this whole class of incident.
//
// # Why it depends on no scenario
//
// Vrooli runs several agents in one environment by design, and agent-manager
// issues their identities -- but agent-manager is a scenario like any other and
// can be down, including while the control plane is repairing itself, which is
// precisely when artifacts move. Identity is therefore graded rather than
// required: what the process observes about itself is always recorded, what the
// environment claims is recorded and labelled as a claim, and verified agent
// provenance is recorded when a caller actually holds some. A receipt carrying
// only observed identity is still a valid receipt. The sink is a plain file for
// the same reason: a ledger that needs a running service is unavailable exactly
// when it is most needed.
package artifactledger

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
)

// Schema identifies the receipt format. It is written into every record so a
// reader can tell an old line from a new one without guessing.
const Schema = "vrooli.removal-receipt/1"

// ErrAbandoned reports that a removal was refused once the lock was held,
// because the predicate that authorized it no longer held.
//
// It is deliberately an error rather than a silent nil. A caller has to be able
// to tell "removed" from "correctly declined to remove": returning nil for both
// let an abandoned reclamation be counted as reclaimed, which is precisely the
// kind of quiet miscount that hides a concurrency bug.
var ErrAbandoned = errors.New("removal abandoned: authorization no longer holds")

// Outcome values close an intent record.
const (
	OutcomeIntent  = "intent"
	OutcomeRemoved = "removed"
	OutcomeAbsent  = "absent"
	OutcomeFailed  = "failed"
	// OutcomeAbandoned records a removal that was authorized outside the lock
	// and refused once the lock was held. It is the visible proof that the
	// check-then-act guard fired, and it is deliberately not an error: the
	// system did the right thing.
	OutcomeAbandoned = "abandoned"
)

// claimedIdentityEnvKeys are the environment values a receipt may record as a
// claim.
//
// Two keys are deliberately excluded. VROOLI_AGENT_IDENTITY_TOKEN authenticates
// an agent to agent-manager, which makes it a credential, and no credential may
// appear in an evidence artifact. VROOLI_AGENT_MANAGER_RUN_ID is legacy: no
// production code reads it, and plan-manager carries a test asserting it must
// never attribute a mutation. Reading it here would revive a signal the
// ecosystem already retired.
var claimedIdentityEnvKeys = struct {
	Session string
	Sandbox string
}{
	Session: "VROOLI_SWARM_MANAGER_SESSION_ID",
	Sandbox: "VROOLI_SANDBOX_ID",
}

// Claimed is identity a process asserts about itself by way of its environment.
//
// A process can set these values itself, so they are a claim and never proof.
// The repository already draws this line: packages/api-core/provenance keeps
// Invocation separate from Provenance precisely because "API clients can forge
// it", and plan-manager refuses to attribute a log entry on an environment
// claim. A receipt records claims because they are useful for correlation, and
// labels them because they are not evidence.
type Claimed struct {
	Session string `json:"session,omitempty"`
	Sandbox string `json:"sandbox,omitempty"`
}

// Verified is identity that was authenticated upstream.
//
// Only a caller holding a verified agent provenance may populate this. The
// ledger never derives it, because nothing this process can read about itself
// constitutes verification. Kept as plain strings so this package stays free of
// a dependency on the provenance module.
type Verified struct {
	RunID  string `json:"run_id,omitempty"`
	Actor  string `json:"actor,omitempty"`
	Source string `json:"source,omitempty"`
}

// Identity is who performed a removal, separated by how much it can be trusted.
//
// The observed fields are always populated and always true: they are facts the
// process establishes about itself from the host. Claimed and Verified are
// distinct on purpose -- collapsing them is how an unauthenticated environment
// variable ends up reading like an authenticated identity.
type Identity struct {
	// Observed -- computable from the host alone, no service dependency.
	Node    string `json:"node"`
	User    string `json:"user,omitempty"`
	PID     int    `json:"pid"`
	Process string `json:"process"`

	Claimed  Claimed   `json:"claimed,omitzero"`
	Verified *Verified `json:"verified,omitempty"`
}

// Attributed reports whether this identity carries verified agent provenance.
// A claim, however specific, is not attribution.
func (i Identity) Attributed() bool {
	return i.Verified != nil && strings.TrimSpace(i.Verified.RunID) != ""
}

var (
	processNonceOnce sync.Once
	processNonce     string
)

// ProcessNonce identifies this process instance for its lifetime.
//
// It is not a substitute for the process start time that pid-reuse detection
// needs; that arrives with the liveness work, which is where it is first
// actually required. Here it only has to distinguish two receipts written by
// different processes that happen to share a pid across a reboot.
func ProcessNonce() string {
	processNonceOnce.Do(func() {
		buffer := make([]byte, 8)
		if _, err := rand.Read(buffer); err != nil {
			processNonce = fmt.Sprintf("pid-%d", os.Getpid())
			return
		}
		processNonce = hex.EncodeToString(buffer)
	})
	return processNonce
}

// LookupEnv is the environment seam. Tests replace it.
var LookupEnv = os.LookupEnv

// CurrentIdentity resolves the identity of the running process.
//
// It never returns a verified identity: verification happens upstream, and a
// process asking itself who it is cannot produce one. A caller that holds
// verified provenance attaches it with Removal.Verified.
func CurrentIdentity() Identity {
	identity := Identity{
		PID:     os.Getpid(),
		Process: ProcessNonce(),
	}
	if node, err := os.Hostname(); err == nil {
		identity.Node = strings.TrimSpace(node)
	}
	if current, err := user.Current(); err == nil {
		identity.User = strings.TrimSpace(current.Username)
	}
	if value, ok := LookupEnv(claimedIdentityEnvKeys.Session); ok {
		identity.Claimed.Session = strings.TrimSpace(value)
	}
	if value, ok := LookupEnv(claimedIdentityEnvKeys.Sandbox); ok {
		identity.Claimed.Sandbox = strings.TrimSpace(value)
	}
	return identity
}

// Removal describes one artifact a caller intends to remove.
type Removal struct {
	// Path is the artifact being removed.
	Path string
	// Kind describes what the artifact is (binary, build-metadata, manifest).
	Kind string
	// Component names the code path performing the removal, so a receipt
	// points at the caller rather than only at the ledger.
	Component string
	// Predicate is the rule that authorized this removal, in the caller's own
	// words. This is the field that turns a receipt from "something deleted
	// this" into "this rule fired, and here is why".
	Predicate string
	// Subject is the artifact family this removal belongs to, and is what the
	// lock is taken on. A CLI is installed as a triple, and the installer locks
	// the binary path -- so removing a sidecar has to take the binary's lock,
	// not its own, or install and removal would not exclude each other at all.
	// Empty means the removal is its own subject.
	Subject string
	// Verify re-validates, under the lock, the predicate that authorized this
	// removal. It exists because a decision made outside the lock is a hint:
	// between deciding an artifact was reclaimable and deleting it, another
	// agent may have reinstalled the scenario that owns it. Returning an error
	// abandons the removal and records why. Optional: a removal whose authority
	// cannot expire (an operator-authorized uninstall plan) has nothing to
	// re-check.
	Verify func() error
	// Generation, when non-zero, records the artifact generation the decision
	// was made against.
	Generation int64
	// Verified carries authenticated agent provenance when the caller holds
	// some. The ledger never invents it; an absent value records that the
	// removal happened outside a verified agent run, which is itself worth
	// knowing.
	Verified *Verified
}

// Receipt is one durable line in the ledger.
type Receipt struct {
	Schema     string   `json:"schema"`
	ID         string   `json:"id"`
	Outcome    string   `json:"outcome"`
	Path       string   `json:"path"`
	Kind       string   `json:"kind,omitempty"`
	Component  string   `json:"component"`
	Predicate  string   `json:"predicate"`
	Generation int64    `json:"generation,omitempty"`
	Identity   Identity `json:"identity"`
	RecordedAt string   `json:"recorded_at"`
	Error      string   `json:"error,omitempty"`
}

// Ledger appends receipts to a directory of daily JSONL files.
type Ledger struct {
	dir      string
	now      func() time.Time
	identity func() Identity
}

// New returns a ledger writing beneath the runtime state directory for home.
func New(home string) (*Ledger, error) {
	state, err := repocontract.RuntimeHomeEntryPath(filepath.Clean(strings.TrimSpace(home)), repocontract.HomeKeyState)
	if err != nil {
		return nil, fmt.Errorf("resolve removal receipt directory: %w", err)
	}
	return NewAt(filepath.Join(state, "removal-receipts")), nil
}

// NewAt returns a ledger writing into an explicit directory.
func NewAt(dir string) *Ledger {
	return &Ledger{
		dir:      filepath.Clean(dir),
		now:      func() time.Time { return time.Now().UTC() },
		identity: CurrentIdentity,
	}
}

// WithClock replaces the ledger clock. For tests.
func (l *Ledger) WithClock(now func() time.Time) *Ledger {
	if now != nil {
		l.now = now
	}
	return l
}

// WithIdentity replaces identity resolution. For tests.
func (l *Ledger) WithIdentity(resolve func() Identity) *Ledger {
	if resolve != nil {
		l.identity = resolve
	}
	return l
}

// Dir reports where receipts are written.
func (l *Ledger) Dir() string { return l.dir }

// Guard is the single seam through which an install-root artifact may be
// removed.
//
// It does three things that a bare unlink does not. It takes the same advisory
// lock the installer takes, so a removal cannot interleave with an install of
// the same artifact. It re-runs the caller's predicate while holding that lock,
// so a decision made outside the lock cannot authorize a deletion after the
// world has changed underneath it. And it records the removal either way.
//
// remove is called only after the intent record is durable. If the intent
// cannot be written, remove is never called: a removal this process cannot
// account for must not happen.
func (l *Ledger) Guard(removal Removal, remove func() error) error {
	if remove == nil {
		return fmt.Errorf("artifactledger: remove function is required")
	}
	if strings.TrimSpace(removal.Path) == "" {
		return fmt.Errorf("artifactledger: removal path is required")
	}
	if strings.TrimSpace(removal.Predicate) == "" {
		// A receipt without a rule records that something happened but not why,
		// which is the state this package exists to end.
		return fmt.Errorf("artifactledger: removal of %s needs a predicate naming the rule that authorized it", removal.Path)
	}

	subject := filepath.Clean(strings.TrimSpace(removal.Subject))
	if strings.TrimSpace(removal.Subject) == "" {
		subject = filepath.Clean(removal.Path)
	}
	// Nothing to lock, nothing to remove, nothing to attribute. Removal paths
	// here are idempotent by design and are routinely called for artifacts that
	// are already gone; recording those would bury the removals that matter
	// under receipts for events that never happened.
	if _, err := os.Stat(filepath.Dir(subject)); err != nil {
		if os.IsNotExist(err) {
			return fs.ErrNotExist
		}
		return fmt.Errorf("resolve install directory for %s: %w", removal.Path, err)
	}

	release, err := lockArtifact(subject)
	if err != nil {
		return fmt.Errorf("lock %s before removing %s: %w", subject, removal.Path, err)
	}
	defer release()

	// Re-check under the lock. An artifact that is already absent produces no
	// records: the ledger describes removals, not attempts on empty space.
	if _, err := os.Lstat(filepath.Clean(removal.Path)); err != nil {
		if os.IsNotExist(err) {
			return fs.ErrNotExist
		}
		return fmt.Errorf("inspect %s before removal: %w", removal.Path, err)
	}

	receipt := Receipt{
		Schema:     Schema,
		ID:         newReceiptID(),
		Outcome:    OutcomeIntent,
		Path:       filepath.Clean(removal.Path),
		Kind:       strings.TrimSpace(removal.Kind),
		Component:  strings.TrimSpace(removal.Component),
		Predicate:  strings.TrimSpace(removal.Predicate),
		Generation: removal.Generation,
		Identity:   l.identity(),
		RecordedAt: l.now().UTC().Format(time.RFC3339Nano),
	}
	if removal.Verified != nil {
		verified := *removal.Verified
		receipt.Identity.Verified = &verified
	}

	// Re-validate under the lock. This is the whole check-then-act fix, and it
	// runs before the intent record so an abandoned removal reads as one event
	// rather than an intent nothing ever closed.
	if removal.Verify != nil {
		if verifyErr := removal.Verify(); verifyErr != nil {
			abandoned := receipt
			abandoned.Outcome = OutcomeAbandoned
			abandoned.Error = verifyErr.Error()
			if appendErr := l.append(abandoned); appendErr != nil {
				return fmt.Errorf("record abandoned removal of %s: %w", receipt.Path, appendErr)
			}
			return fmt.Errorf("%w: %s: %v", ErrAbandoned, receipt.Path, verifyErr)
		}
	}

	if err := l.append(receipt); err != nil {
		return fmt.Errorf("record removal intent for %s: %w", receipt.Path, err)
	}

	removeErr := remove()

	outcome := receipt
	outcome.RecordedAt = l.now().UTC().Format(time.RFC3339Nano)
	switch {
	case removeErr == nil:
		outcome.Outcome = OutcomeRemoved
	case os.IsNotExist(removeErr):
		outcome.Outcome = OutcomeAbsent
	default:
		outcome.Outcome = OutcomeFailed
		outcome.Error = removeErr.Error()
	}
	appendErr := l.append(outcome)

	if removeErr != nil {
		return removeErr
	}
	if appendErr != nil {
		return fmt.Errorf("record removal outcome for %s: %w", receipt.Path, appendErr)
	}
	return nil
}

// lockArtifact is the seam's exclusion primitive.
//
// The lock file is <subject>.lock, which is the same path
// buildinfo.AcquireBinaryInstallLock uses. That agreement is the entire point:
// two different lock names would leave install and removal free to interleave
// while both appeared to be locking.
var lockArtifact = func(subject string) (func(), error) {
	return platform.AcquireFileLock(subject + ".lock")
}

// append writes one receipt line under an exclusive lock.
//
// The lock matters because several agents share this host by design, and an
// interleaved partial line would corrupt the one record that says what
// happened. O_APPEND alone is not sufficient across platforms.
func (l *Ledger) append(receipt Receipt) error {
	if err := os.MkdirAll(l.dir, 0o700); err != nil {
		return fmt.Errorf("create receipt directory: %w", err)
	}
	path := filepath.Join(l.dir, receipt.RecordedAt[:10]+".jsonl")

	release, err := platform.AcquireFileLock(path + ".lock")
	if err != nil {
		return fmt.Errorf("lock receipt file: %w", err)
	}
	defer release()

	encoded, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("encode receipt: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open receipt file: %w", err)
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		_ = file.Close()
		return fmt.Errorf("write receipt: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync receipt: %w", err)
	}
	return file.Close()
}

// Read returns every receipt in the ledger, oldest file first. A malformed line
// is reported rather than skipped: a ledger that silently drops what it cannot
// parse is not evidence.
func (l *Ledger) Read() ([]Receipt, error) {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read receipt directory: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			names = append(names, entry.Name())
		}
	}
	sortStrings(names)

	var receipts []Receipt
	for _, name := range names {
		data, readErr := os.ReadFile(filepath.Join(l.dir, name))
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", name, readErr)
		}
		for index, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var receipt Receipt
			if err := json.Unmarshal([]byte(line), &receipt); err != nil {
				return nil, fmt.Errorf("decode %s line %d: %w", name, index+1, err)
			}
			receipts = append(receipts, receipt)
		}
	}
	return receipts, nil
}

func newReceiptID() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("rcpt-%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
