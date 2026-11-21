package evidenceretention

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operatordecision"
)

// An agent cannot prune evidence on its own authority. Instead it asks: the
// request raises an operator decision on the shared operatordecision rail
// (surfaced in the web-console decision inbox), and an approval resolved
// through Agent Manager's signed notification-hub webhook makes Agent Manager
// itself apply exactly the previewed prune. An approved ask is the only
// authority; there is no separate grant.
const (
	// ApprovalSource and ApprovalEventType identify the decision. The event
	// type keeps Agent Manager's established underscore token.
	ApprovalSource    = "agent-manager"
	ApprovalEventType = "agent_manager.evidence_prune.decision_requested.v1"

	approvalKind = "evidence-prune"
	// correlationRequestID carries our ask id: the resolution arrives keyed by
	// notification-hub's own id.
	correlationRequestID = "evidence_prune_request_id"
	approveAnswer        = "approve"
	approvalApplyTimeout = 30 * time.Minute

	extraEfforts      = "efforts"
	extraClasses      = "classes"
	extraPaths        = "paths"
	extraReason       = "reason"
	extraRequestedBy  = "requested_by"
	extraRunID        = "raised_by_run_id"
	extraSession      = "web_console_session_id"
	extraPreviewBytes = "preview_bytes"
	extraPreviewFiles = "preview_files"
	extraSkipped      = "preview_skipped"
	extraAnswer       = "answer"
	extraReceipt      = "receipt_path"
	extraApplyError   = "apply_error"
	// extraSubjectKey marks an ask raised under a hub subject key. An ask without
	// it was raised by an older build and can only be closed by its event id.
	extraSubjectKey = "subject_key"
	// extraBreakdown is the preview's size, file and path counts per class, for
	// the page. An ask raised before it existed shows its page without the table.
	extraBreakdown = "preview_by_class"
)

// Approval attributes a prune applied on an operator decision.
type Approval struct {
	DecisionID  string    `json:"decisionId"`
	RequestedBy string    `json:"requestedBy,omitempty"`
	ApprovedBy  string    `json:"approvedBy"`
	ApprovedAt  time.Time `json:"approvedAt"`
}

// ApprovalRequest is an agent's ask to prune named efforts.
type ApprovalRequest struct {
	Efforts []string
	Classes []Class
	Reason  string
	// RequestedBy is the verified agent subject (its run id); it is also the
	// flood-cap subject.
	RequestedBy string
	// WebConsoleSessionID is the agent's stable delivery handle, if any.
	WebConsoleSessionID string
}

// ApprovalTicket is the answer to an approval request.
type ApprovalTicket struct {
	DecisionID string  `json:"decision_id"`
	Raised     bool    `json:"-"`
	Preview    Receipt `json:"preview"`
}

// DecisionRaiser is the seam over operatordecision.Raiser.
type DecisionRaiser interface {
	Raise(ctx context.Context, ask operatordecision.PendingAsk, facts operatordecision.Facts) (bool, error)
	PublishWithdrawal(ctx context.Context, w operatordecision.Withdrawal) error
}

// Approvals raises prune asks and applies approved ones.
type Approvals struct {
	service *Service
	store   operatordecision.PendingStore
	raiser  DecisionRaiser
	mu      sync.Mutex // serializes resolution so a replay cannot apply twice
	// run executes an approved prune. It detaches from the webhook request in
	// production because a prune can outlast the delivery timeout.
	run func(func())
}

// NewApprovals wires the approval path. store is also the raiser's store.
func NewApprovals(service *Service, store operatordecision.PendingStore, raiser DecisionRaiser) *Approvals {
	return &Approvals{service: service, store: store, raiser: raiser, run: func(f func()) { go f() }}
}

// NewApprovalRaiser builds the substrate raiser for this decision.
func NewApprovalRaiser(store operatordecision.PendingStore, publisher operatordecision.Publisher) operatordecision.Raiser {
	return operatordecision.Raiser{Source: ApprovalSource, Kind: approvalKind, EventType: ApprovalEventType, Store: store, Publisher: publisher}
}

// subjectKeyFor names the one open operator decision for a prune scope, so a
// request raised again after the first one aged out updates it instead of adding
// a second.
func subjectKeyFor(id string) string { return approvalKind + ":" + id }

// ApprovalID is the stable correlation id for one (efforts, classes) scope,
// so a retried request dedupes onto the one outstanding ask.
func ApprovalID(efforts []string, classes []Class) string {
	e, _ := json.Marshal(efforts)
	c, _ := json.Marshal(classes)
	return operatordecision.CorrelationID(approvalKind, string(e), string(c))
}

// normalizeScope validates, de-duplicates and sorts the requested scope.
func normalizeScope(efforts []string, classes []Class) ([]string, []Class, error) {
	seen := map[string]bool{}
	var outE []string
	for _, name := range efforts {
		name = strings.TrimSpace(name)
		if name == "" || name != filepath.Base(name) || name == "." || name == ".." {
			return nil, nil, fmt.Errorf("%w: effort directory name %q", ErrInvalidRequest, name)
		}
		if !seen[name] {
			seen[name] = true
			outE = append(outE, name)
		}
	}
	if len(outE) == 0 {
		return nil, nil, fmt.Errorf("%w: an approval request must name at least one effort", ErrInvalidRequest)
	}
	seenC := map[Class]bool{}
	var outC []Class
	for _, c := range classes {
		if !knownClass(c) {
			return nil, nil, fmt.Errorf("%w: unknown class %q", ErrInvalidRequest, c)
		}
		if !seenC[c] {
			seenC[c] = true
			outC = append(outC, c)
		}
	}
	sort.Strings(outE)
	sort.Slice(outC, func(i, j int) bool { return outC[i] < outC[j] })
	return outE, outC, nil
}

// Request raises (idempotently) an operator decision to prune the named
// efforts. The preview is the same dry run a human sees; an outstanding ask
// keeps its original preview so a retry republishes identical facts. A
// preview with nothing removable raises nothing. Request never removes.
func (a *Approvals) Request(ctx context.Context, req ApprovalRequest) (ApprovalTicket, error) {
	efforts, classes, err := normalizeScope(req.Efforts, req.Classes)
	if err != nil {
		return ApprovalTicket{}, err
	}
	if strings.TrimSpace(req.Reason) == "" || len(req.Reason) > 512 {
		return ApprovalTicket{}, fmt.Errorf("%w: a prune request requires a reason of 1-512 bytes", ErrInvalidRequest)
	}
	id := ApprovalID(efforts, classes)
	ticket := ApprovalTicket{DecisionID: id}
	existing, err := a.store.Get(ctx, id)
	if err != nil {
		return ticket, err
	}
	if err := a.retireKeylessAsk(ctx, existing); err != nil {
		return ticket, err
	}
	var ask operatordecision.PendingAsk
	if existing != nil && existing.Outstanding(time.Now()) {
		ask = *existing
		ticket.Preview = previewFromAsk(existing)
	} else {
		preview, err := a.service.Prune(ctx, PruneRequest{Efforts: efforts, Classes: classes, DryRun: true})
		if err != nil {
			return ticket, err
		}
		ticket.Preview = preview
		if len(preview.Removed) == 0 {
			// Nothing is left to remove (it was pruned another way, or nothing
			// qualifies any more): an ask still open for this scope is moot.
			return ticket, a.withdrawMoot(ctx, existing, "Nothing is left to prune for this request, so it no longer needs a decision.")
		}
		ask = newAsk(id, efforts, classes, preview, req)
	}
	raised, err := a.raiser.Raise(ctx, ask, approvalFacts(&ask))
	ticket.Raised = raised
	return ticket, err
}

// retireKeylessAsk closes a pending ask an older build raised without a subject
// key, by its event id, and marks it keyed so the raise below publishes the
// current facts under the key. Idempotent: a keyed ask is left alone.
func (a *Approvals) retireKeylessAsk(ctx context.Context, ask *operatordecision.PendingAsk) error {
	if ask == nil || ask.State != operatordecision.PendingStatePending || ask.Extra[extraSubjectKey] != "" {
		return nil
	}
	if ask.EventID != "" {
		if err := a.raiser.PublishWithdrawal(ctx, operatordecision.Withdrawal{RequesterRef: ask.EventID, Reason: "Replaced by an updated version of this request."}); err != nil {
			return err
		}
	}
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	ask.Extra[extraSubjectKey] = subjectKeyFor(ask.ID)
	ask.EventID, ask.Published = "", false
	return a.store.Put(ctx, ask)
}

// withdrawMoot closes the ask's open decision at the hub and records it as
// withdrawn. It does nothing for an ask that is not pending. The hub is told
// first, so a failure leaves the ask pending and the next request retries.
func (a *Approvals) withdrawMoot(ctx context.Context, ask *operatordecision.PendingAsk, reason string) error {
	if ask == nil || ask.State != operatordecision.PendingStatePending {
		return nil
	}
	w := operatordecision.Withdrawal{SubjectKey: subjectKeyFor(ask.ID), Reason: reason}
	if ask.Extra[extraSubjectKey] == "" {
		w.SubjectKey, w.RequesterRef = "", ask.EventID
	}
	if err := a.raiser.PublishWithdrawal(ctx, w); err != nil {
		return err
	}
	now := time.Now().UTC()
	ask.State, ask.ResolvedAt = operatordecision.PendingStateWithdrawn, &now
	return a.store.Put(ctx, ask)
}

func newAsk(id string, efforts []string, classes []Class, preview Receipt, req ApprovalRequest) operatordecision.PendingAsk {
	paths := make([]string, 0, len(preview.Removed))
	for _, r := range preview.Removed {
		paths = append(paths, r.Path)
	}
	skipped := make([]string, 0, len(preview.Skipped))
	for _, s := range preview.Skipped {
		skipped = append(skipped, strings.TrimSpace(s.Effort+" "+s.Path)+": "+s.Reason)
	}
	subject := strings.TrimSpace(req.RequestedBy)
	if subject == "" {
		subject = "agent-manager:evidence-prune"
	}
	runID := ""
	if _, err := uuid.Parse(strings.TrimSpace(req.RequestedBy)); err == nil {
		runID = strings.TrimSpace(req.RequestedBy)
	}
	return operatordecision.PendingAsk{
		ID:      id,
		Subject: subject,
		Extra: map[string]string{
			extraEfforts:      mustJSON(efforts),
			extraClasses:      mustJSON(classes),
			extraPaths:        mustJSON(paths),
			extraSkipped:      mustJSON(skipped),
			extraReason:       strings.TrimSpace(req.Reason),
			extraRequestedBy:  strings.TrimSpace(req.RequestedBy),
			extraRunID:        runID,
			extraSession:      strings.TrimSpace(req.WebConsoleSessionID),
			extraPreviewBytes: strconv.FormatInt(preview.RemovedBytes, 10),
			extraPreviewFiles: strconv.Itoa(preview.RemovedFiles),
			extraSubjectKey:   subjectKeyFor(id),
			extraBreakdown:    string(mustJSONMap(breakdown(preview))),
		},
	}
}

// previewFromAsk restates a stored preview without rescanning.
func previewFromAsk(ask *operatordecision.PendingAsk) Receipt {
	r := Receipt{Mode: "operator", DryRun: true, Removed: []Removal{}}
	r.RemovedBytes, _ = strconv.ParseInt(ask.Extra[extraPreviewBytes], 10, 64)
	r.RemovedFiles, _ = strconv.Atoi(ask.Extra[extraPreviewFiles])
	for _, p := range decodeList(ask.Extra[extraPaths]) {
		r.Removed = append(r.Removed, Removal{Path: p})
	}
	return r
}

// Handles claims only evidence-prune resolutions.
func (a *Approvals) Handles(sourceEventType string) bool {
	return sourceEventType == ApprovalEventType
}

// Apply records the operator's answer and, on an explicit approval by a named
// operator, applies exactly the approved prune. It is replay-safe: the ask is
// marked decided before any removal, so a redelivered resolution is a benign
// no-op and the prune runs at most once.
func (a *Approvals) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	id := strings.TrimSpace(res.Correlation[correlationRequestID])
	if id == "" {
		return true, nil
	}
	req, approved, err := a.decide(ctx, id, res)
	if err != nil {
		return false, err
	}
	if approved {
		a.run(func() { a.applyApproved(id, req) })
	}
	return true, nil
}

// decide records the answer under the lock. It reports approved=true only for
// the first delivery of an approval, which alone runs the prune.
func (a *Approvals) decide(ctx context.Context, id string, res operatordecision.DecisionResolution) (PruneRequest, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ask, err := a.store.Get(ctx, id)
	if err != nil {
		return PruneRequest{}, false, err
	}
	if ask == nil || ask.State != operatordecision.PendingStatePending {
		return PruneRequest{}, false, nil
	}
	now := time.Now().UTC()
	approvedBy := strings.TrimSpace(res.AnsweredBy)
	// Only an explicit, attributed operator answer approves. A default or an
	// expiry never deletes, and neither does an answer without an approver.
	approved := res.State == operatordecision.StateAnswered && strings.TrimSpace(res.Answer) == approveAnswer && approvedBy != ""
	ask.ResolvedAt = &now
	ask.ResolvedBy = approvedBy
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	if res.State == operatordecision.StateWithdrawn {
		// The hub closed the decision (replaced, or no longer confirmed): no answer.
		ask.State = operatordecision.PendingStateWithdrawn
		return PruneRequest{}, false, a.store.Put(ctx, ask)
	}
	if !approved {
		ask.State = operatordecision.PendingStateRejected
		ask.Extra[extraAnswer] = strings.TrimSpace(res.Answer)
		return PruneRequest{}, false, a.store.Put(ctx, ask)
	}
	ask.State = operatordecision.PendingStateApproved
	ask.Extra[extraAnswer] = approveAnswer
	if err := a.store.Put(ctx, ask); err != nil {
		return PruneRequest{}, false, err
	}
	return approvedPrune(ask, now), true, nil
}

// approvedPrune rebuilds the prune from the stored ask only: the approved
// efforts, classes and previewed paths. Nothing on the resolution can widen it.
func approvedPrune(ask *operatordecision.PendingAsk, at time.Time) PruneRequest {
	classes := []Class{}
	for _, c := range decodeList(ask.Extra[extraClasses]) {
		classes = append(classes, Class(c))
	}
	paths := decodeList(ask.Extra[extraPaths])
	if paths == nil {
		paths = []string{}
	}
	return PruneRequest{
		Efforts:   decodeList(ask.Extra[extraEfforts]),
		Classes:   classes,
		OnlyPaths: paths,
		Actor:     ask.ResolvedBy,
		Reason:    ask.Extra[extraReason],
		Approval: &Approval{
			DecisionID:  ask.ID,
			RequestedBy: ask.Extra[extraRequestedBy],
			ApprovedBy:  ask.ResolvedBy,
			ApprovedAt:  at,
		},
	}
}

func (a *Approvals) applyApproved(id string, req PruneRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), approvalApplyTimeout)
	defer cancel()
	receipt, err := a.service.Prune(ctx, req)
	a.mu.Lock()
	defer a.mu.Unlock()
	ask, getErr := a.store.Get(ctx, id)
	if getErr != nil || ask == nil {
		return
	}
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	ask.Extra[extraReceipt] = receipt.ReceiptPath
	if err != nil {
		ask.Extra[extraApplyError] = err.Error()
	}
	_ = a.store.Put(ctx, ask)
}

// ApprovalStatus is the observable state of one ask.
type ApprovalStatus struct {
	DecisionID   string     `json:"decision_id"`
	State        string     `json:"state"`
	Efforts      []string   `json:"efforts"`
	Classes      []string   `json:"classes"`
	PreviewBytes int64      `json:"preview_bytes"`
	PreviewFiles int        `json:"preview_files"`
	ResolvedBy   string     `json:"resolved_by,omitempty"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	ReceiptPath  string     `json:"receipt_path,omitempty"`
	ApplyError   string     `json:"apply_error,omitempty"`
}

// Status reports one ask, or nil when unknown.
func (a *Approvals) Status(ctx context.Context, id string) (*ApprovalStatus, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ask, err := a.store.Get(ctx, strings.TrimSpace(id))
	if err != nil || ask == nil {
		return nil, err
	}
	state := ask.State
	if ask.State == operatordecision.PendingStatePending && !ask.Outstanding(time.Now()) {
		state = "expired"
	}
	out := &ApprovalStatus{
		DecisionID: ask.ID, State: state,
		Efforts: decodeList(ask.Extra[extraEfforts]), Classes: decodeList(ask.Extra[extraClasses]),
		ResolvedBy: ask.ResolvedBy, ResolvedAt: ask.ResolvedAt,
		ReceiptPath: ask.Extra[extraReceipt], ApplyError: ask.Extra[extraApplyError],
	}
	out.PreviewBytes, _ = strconv.ParseInt(ask.Extra[extraPreviewBytes], 10, 64)
	out.PreviewFiles, _ = strconv.Atoi(ask.Extra[extraPreviewFiles])
	return out, nil
}

// FileAskStore keeps one JSON file per ask beside the prune receipts. Asks
// are rare and small, and the receipts they lead to already live here.
type FileAskStore struct {
	dir string
	mu  sync.Mutex
}

// NewFileAskStore stores asks under dir.
func NewFileAskStore(dir string) *FileAskStore { return &FileAskStore{dir: dir} }

func (s *FileAskStore) path(id string) (string, error) {
	if id == "" || id != filepath.Base(id) || strings.HasPrefix(id, ".") {
		return "", errors.New("invalid ask id")
	}
	return filepath.Join(s.dir, id+".json"), nil
}

// Get returns the ask, or nil when none exists.
func (s *FileAskStore) Get(_ context.Context, id string) (*operatordecision.PendingAsk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.path(id)
	if err != nil {
		return nil, nil
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ask operatordecision.PendingAsk
	if err := json.Unmarshal(data, &ask); err != nil {
		return nil, err
	}
	return &ask, nil
}

// Put writes the ask atomically.
func (s *FileAskStore) Put(_ context.Context, ask *operatordecision.PendingAsk) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.path(ask.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	return writeJSON(p, ask)
}

// CountOutstandingForSubject counts un-decided, unexpired asks of subject.
func (s *FileAskStore) CountOutstandingForSubject(_ context.Context, subject string, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var ask operatordecision.PendingAsk
		if json.Unmarshal(data, &ask) == nil && ask.Subject == subject && ask.Outstanding(now) {
			n++
		}
	}
	return n, nil
}

func mustJSON[T any](v []T) string {
	if v == nil {
		v = []T{}
	}
	data, _ := json.Marshal(v)
	return string(data)
}

func mustJSONMap[T any](v map[string]T) []byte {
	data, _ := json.Marshal(v)
	return data
}

func decodeJSON(raw string, into any) error {
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), into)
}

func decodeList(raw string) []string {
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
