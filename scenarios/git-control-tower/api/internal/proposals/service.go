package proposals

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"git-control-tower/internal/trailers"
)

const (
	maxAnchorFiles   = 50000
	maxProposalPaths = 2000
	maxOpenPerEffort = 5
)

// Service owns proposal lifecycle rules. It never stages or commits outside
// Apply.
type Service struct {
	store *Store
	now   func() time.Time
	newID func(prefix string) string
}

func NewService(store *Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }, newID: newID}
}

// WithClock replaces the clock for tests.
func (s *Service) WithClock(now func() time.Time) *Service {
	if now != nil {
		s.now = now
	}
	return s
}

// ValidationError carries the issues that blocked an agent write.
type ValidationError struct{ Issues []trailers.Issue }

func (e *ValidationError) Error() string {
	var parts []string
	for _, issue := range e.Issues {
		if issue.Severity == trailers.SeverityError {
			parts = append(parts, issue.Key+": "+issue.Message)
		}
	}
	return ErrValidation.Error() + ": " + strings.Join(parts, "; ")
}

func (e *ValidationError) Unwrap() error { return ErrValidation }

type AnchorRequest struct {
	EffortRef string
	Epoch     string
	Scopes    []string
	Actor     Actor
}

// Anchor fingerprints the files already dirty in a scope.
func (s *Service) Anchor(ctx context.Context, repo Repo, req AnchorRequest) (Anchor, error) {
	if len(req.Scopes) == 0 {
		return Anchor{}, fmt.Errorf("%w: at least one scope is required", ErrInvalid)
	}
	matcher, err := newScopeMatcher(req.Scopes)
	if err != nil {
		return Anchor{}, err
	}
	head, err := repo.Git.Head(ctx)
	if err != nil {
		return Anchor{}, err
	}
	status, err := repo.Git.Status(ctx)
	if err != nil {
		return Anchor{}, err
	}
	var dirty []string
	for _, entry := range status {
		if matcher.Match(entry.Path) {
			dirty = append(dirty, entry.Path)
		}
	}
	dirty = uniqueSorted(dirty)
	if len(dirty) > maxAnchorFiles {
		return Anchor{}, fmt.Errorf("%w: scope has %d dirty files; narrow it below %d", ErrInvalid, len(dirty), maxAnchorFiles)
	}
	contents, err := repo.Git.Hash(ctx, dirty)
	if err != nil {
		return Anchor{}, err
	}
	anchor := Anchor{
		ID: s.newID("gcta-"), RepositoryID: repo.ID, EffortRef: strings.TrimSpace(req.EffortRef),
		Epoch: trailers.NormalizeEpoch(req.Epoch), Scopes: append([]string(nil), req.Scopes...), Head: head,
		Files: make([]AnchorFile, 0, len(dirty)), CreatedBy: req.Actor, CreatedAt: s.now(),
	}
	for _, path := range dirty {
		content := contents[path]
		anchor.Files = append(anchor.Files, AnchorFile{Path: path, SHA256: content.SHA256, BlobID: content.BlobID, Deleted: content.Deleted})
	}
	return anchor, s.store.SaveAnchor(ctx, anchor)
}

type CreateRequest struct {
	AnchorID     string
	Work         Work
	Subject      string
	Body         string
	Trailers     []trailers.Entry
	Paths        []string
	Evidence     Evidence
	Actor        Actor
	ValidateOnly bool
}

type CreateResult struct {
	View         View
	Issues       []trailers.Issue
	Stored       bool
	SupersededID string
}

// Create builds a proposal from listed paths, checks them against the anchor
// and stores it unless ValidateOnly is set. Agent writes with message errors
// are refused; human writes keep the issues as warnings.
func (s *Service) Create(ctx context.Context, repo Repo, req CreateRequest) (CreateResult, error) {
	paths, err := cleanPaths(req.Paths)
	if err != nil {
		return CreateResult{}, err
	}
	if len(paths) == 0 {
		return CreateResult{}, fmt.Errorf("%w: list at least one path", ErrInvalid)
	}
	if len(paths) > maxProposalPaths {
		return CreateResult{}, fmt.Errorf("%w: %d paths exceed the %d-path limit", ErrInvalid, len(paths), maxProposalPaths)
	}
	work := normalizeWork(req.Work)
	anchor, hasAnchor, err := s.resolveAnchor(ctx, repo.ID, req.AnchorID, work)
	if err != nil {
		return CreateResult{}, err
	}
	snapshot, err := readSnapshot(ctx, repo)
	if err != nil {
		return CreateResult{}, err
	}
	var anchorPtr *Anchor
	if hasAnchor {
		anchorPtr = &anchor
	}
	files, excluded, err := buildFiles(ctx, repo, snapshot, paths, anchorPtr, true)
	if err != nil {
		return CreateResult{}, err
	}

	// A new proposal for the same effort and epoch supersedes the open one;
	// otherwise an effort may hold only a few open proposals at once.
	var replaced []Proposal
	if work.EffortRef != "" {
		openForEffort, err := s.store.List(ctx, ListFilter{RepositoryID: repo.ID, EffortRef: work.EffortRef, Limit: 200})
		if err != nil {
			return CreateResult{}, err
		}
		for _, old := range openForEffort {
			if work.Epoch != "" && old.Work.Epoch == work.Epoch {
				replaced = append(replaced, old)
			}
		}
		if remaining := len(openForEffort) - len(replaced); remaining >= maxOpenPerEffort {
			return CreateResult{}, fmt.Errorf("%w: effort %s already has %d open proposals; withdraw or apply some first", ErrInvalid, work.EffortRef, remaining)
		}
	}

	now := s.now()
	proposal := Proposal{
		ID: s.newID("gctp-"), RepositoryID: repo.ID, State: StateOpen, Revision: 1, CreatedBy: req.Actor,
		BaseHead: snapshot.head, Branch: snapshot.branch, Work: work, Files: files, Excluded: excluded,
		Evidence: req.Evidence, CreatedAt: now, UpdatedAt: now,
	}
	if hasAnchor {
		proposal.Evidence.AnchorID = anchor.ID
	}
	subject, body, operatorEdited, carriedFrom := req.Subject, req.Body, req.Actor.IsHuman(), ""
	for _, old := range replaced {
		if old.Message.OperatorEdited && !req.Actor.IsHuman() {
			subject, body, operatorEdited, carriedFrom = old.Message.Subject, old.Message.Body, true, old.ID
		}
	}
	proposal.Message = buildMessage(proposal.ID, work, subject, body, req.Trailers, operatorEdited)
	issues := trailers.Validate(proposal.Message.Subject, proposal.Message.Body, proposal.Message.Trailers, trailers.ValidateOptions{AgentAuthored: !req.Actor.IsHuman()})
	if !req.Actor.IsHuman() && trailers.HasErrors(issues) {
		return CreateResult{Issues: issues}, &ValidationError{Issues: issues}
	}
	proposal.Digest = Digest(proposal.BaseHead, proposal.Files, proposal.Message.Rendered)
	detail := fmt.Sprintf("%d files", len(files))
	if carriedFrom != "" {
		detail += "; operator message carried from " + carriedFrom
	}
	proposal.Events = []Event{{Action: "created", Actor: req.Actor, Revision: 1, Detail: detail, Timestamp: now}}

	result := CreateResult{Issues: issues}
	if !req.ValidateOnly {
		if err := s.store.Insert(ctx, proposal); err != nil {
			return CreateResult{}, err
		}
		result.Stored = true
		for _, old := range replaced {
			old.State, old.SupersededBy, old.UpdatedAt = StateSuperseded, proposal.ID, now
			old.Events = append(old.Events, Event{Action: "superseded", Actor: req.Actor, Revision: old.Revision, Detail: "replaced by " + proposal.ID, Timestamp: now})
			if err := s.store.Update(ctx, old, old.Revision, StateOpen); err == nil {
				result.SupersededID = old.ID
			}
		}
	}
	views, err := s.views(ctx, repo, []Proposal{proposal})
	if err != nil {
		return CreateResult{}, err
	}
	result.View = views[0]
	return result, nil
}

func (s *Service) resolveAnchor(ctx context.Context, repositoryID, anchorID string, work Work) (Anchor, bool, error) {
	if id := strings.TrimSpace(anchorID); id != "" {
		anchor, err := s.store.GetAnchor(ctx, id)
		if err != nil {
			return Anchor{}, false, err
		}
		if anchor.RepositoryID != repositoryID {
			return Anchor{}, false, fmt.Errorf("%w: anchor %s belongs to another repository", ErrInvalid, id)
		}
		return anchor, true, nil
	}
	if work.EffortRef == "" || work.Epoch == "" {
		return Anchor{}, false, nil
	}
	return s.store.LatestAnchor(ctx, repositoryID, work.EffortRef, work.Epoch)
}

func normalizeWork(work Work) Work {
	normalized := Work{EffortRef: strings.TrimSpace(work.EffortRef), EffortRevision: strings.TrimSpace(work.EffortRevision), Epoch: trailers.NormalizeEpoch(work.Epoch)}
	for _, run := range work.RunIDs {
		if run = strings.ToLower(strings.TrimSpace(run)); run != "" && !contains(normalized.RunIDs, run) {
			normalized.RunIDs = append(normalized.RunIDs, run)
		}
	}
	for _, plan := range work.Plans {
		if plan = strings.TrimSpace(plan); plan != "" && !contains(normalized.Plans, plan) {
			normalized.Plans = append(normalized.Plans, plan)
		}
	}
	return normalized
}

// buildMessage renders generated work trailers in registry order, then the
// caller's trailers in their order, with exact duplicates collapsed.
func buildMessage(id string, work Work, subject, body string, extra []trailers.Entry, operatorEdited bool) Message {
	entries := trailers.WorkEntries(trailers.Work{
		EffortRef: work.EffortRef, EffortRevision: work.EffortRevision, Epoch: work.Epoch,
		RunIDs: work.RunIDs, Plans: work.Plans, ProposalID: id,
	})
	for _, entry := range extra {
		if strings.TrimSpace(entry.Key) != "" {
			entries = append(entries, trailers.Entry{Key: entry.Key, Value: entry.Value})
		}
	}
	return finishMessage(Message{Subject: strings.TrimSpace(subject), Body: trailers.CleanBody(body), Trailers: entries, OperatorEdited: operatorEdited})
}

func finishMessage(message Message) Message {
	message.Subject = strings.TrimSpace(message.Subject)
	message.Body = trailers.CleanBody(message.Body)
	message.Trailers = trailers.Normalize(message.Trailers)
	message.Rendered = trailers.Render(message.Subject, message.Body, message.Trailers)
	return message
}

// snapshot is one read of repository state used to build files.
type snapshot struct {
	head   string
	branch string
	xy     map[string]string
}

func readSnapshot(ctx context.Context, repo Repo) (snapshot, error) {
	head, err := repo.Git.Head(ctx)
	if err != nil {
		return snapshot{}, err
	}
	branch, err := repo.Git.Branch(ctx)
	if err != nil {
		return snapshot{}, err
	}
	status, err := repo.Git.Status(ctx)
	if err != nil {
		return snapshot{}, err
	}
	snap := snapshot{head: head, branch: branch, xy: map[string]string{}}
	for _, entry := range status {
		snap.xy[entry.Path] = entry.XY
	}
	return snap, nil
}

// buildFiles hashes listed paths and checks them against the anchor. In
// strict mode (create) a listed path without its own change is refused; in
// lenient mode (refresh) it is excluded with a reason.
func buildFiles(ctx context.Context, repo Repo, snap snapshot, paths []string, anchor *Anchor, strict bool) ([]File, []Exclusion, error) {
	var excluded []Exclusion
	var clean []string
	for _, path := range paths {
		if _, dirty := snap.xy[path]; !dirty {
			clean = append(clean, path)
		}
	}
	if len(clean) > 0 {
		ignored, err := repo.Git.Ignored(ctx, clean)
		if err != nil {
			return nil, nil, err
		}
		if strict && len(ignored) > 0 {
			return nil, nil, fmt.Errorf("%w: ignored paths cannot be proposed: %s", ErrInvalid, strings.Join(ignored, ", "))
		}
		if strict {
			return nil, nil, fmt.Errorf("%w: paths have no uncommitted change (list files, not directories): %s", ErrInvalid, strings.Join(clean, ", "))
		}
		for _, path := range clean {
			excluded = append(excluded, Exclusion{Path: path, Reason: "now_clean", Detail: "no uncommitted change remains"})
		}
	}
	var matcher scopeMatcher
	var anchored map[string]AnchorFile
	var deltaCandidates []string
	if anchor != nil {
		var err error
		if matcher, err = newScopeMatcher(anchor.Scopes); err != nil {
			return nil, nil, err
		}
		anchored = map[string]AnchorFile{}
		for _, file := range anchor.Files {
			anchored[file.Path] = file
		}
		for path := range snap.xy {
			if matcher.Match(path) {
				deltaCandidates = append(deltaCandidates, path)
			}
		}
	}
	dirtyListed := subtract(paths, clean)
	contents, err := repo.Git.Hash(ctx, uniqueSorted(append(append([]string(nil), dirtyListed...), deltaCandidates...)))
	if err != nil {
		return nil, nil, err
	}
	changedSinceAnchor := func(path string) bool {
		prior, wasDirty := anchored[path]
		return !wasDirty || prior.ContentID() != contents[path].ContentID()
	}
	var unchanged, directories []string
	files := make([]File, 0, len(dirtyListed))
	for _, path := range dirtyListed {
		content := contents[path]
		if content.BlobID == DirectoryBlob {
			directories = append(directories, path)
			continue
		}
		file := File{Path: path, Kind: changeKind(snap.xy[path], content), SHA256: content.SHA256, BlobID: content.BlobID, Deleted: content.Deleted, Source: "listed"}
		switch {
		case anchor == nil:
			file.Flags = append(file.Flags, Flag{Code: FlagNoAnchor})
		case !matcher.Match(path):
			file.Flags = append(file.Flags, Flag{Code: FlagOutsideAnchorScope, Detail: anchor.ID})
		case !changedSinceAnchor(path):
			unchanged = append(unchanged, path)
			continue
		default:
			if _, wasDirty := anchored[path]; wasDirty {
				file.Flags = append(file.Flags, Flag{Code: FlagMixedPrior, Detail: anchor.ID})
			}
		}
		if file.Deleted {
			file.SHA256, file.BlobID = "", ""
		}
		files = append(files, file)
	}
	if len(directories) > 0 {
		if strict {
			return nil, nil, fmt.Errorf("%w: directories and submodules cannot be proposed: %s", ErrInvalid, strings.Join(directories, ", "))
		}
		for _, path := range directories {
			excluded = append(excluded, Exclusion{Path: path, Reason: "directory", Detail: "directories and submodules are not supported"})
		}
	}
	if len(unchanged) > 0 {
		if strict {
			return nil, nil, fmt.Errorf("%w: paths are unchanged since anchor %s: %s", ErrInvalid, anchor.ID, strings.Join(unchanged, ", "))
		}
		for _, path := range unchanged {
			excluded = append(excluded, Exclusion{Path: path, Reason: "unchanged_since_anchor", Detail: anchor.ID})
		}
	}
	listed := map[string]struct{}{}
	for _, path := range paths {
		listed[path] = struct{}{}
	}
	for _, path := range uniqueSorted(deltaCandidates) {
		if _, ok := listed[path]; !ok && changedSinceAnchor(path) {
			excluded = append(excluded, Exclusion{Path: path, Reason: "not_listed", Detail: "changed in the anchor scope since " + anchor.ID})
		}
	}
	if repo.Sandbox != nil && len(files) > 0 {
		if pending, err := repo.Sandbox(ctx); err == nil {
			for i := range files {
				for _, run := range pending[files[i].Path] {
					files[i].Flags = append(files[i].Flags, Flag{Code: FlagSandboxPending, Detail: run})
				}
			}
		}
	}
	sort.Slice(excluded, func(i, j int) bool { return excluded[i].Path < excluded[j].Path })
	return files, excluded, nil
}

func changeKind(xy string, content Content) ChangeKind {
	switch {
	case content.Deleted:
		return KindDeleted
	case xy == "??", strings.HasPrefix(xy, "A"), strings.HasPrefix(xy, "R"), strings.HasPrefix(xy, "C"):
		return KindAdded
	default:
		return KindModified
	}
}

// List returns proposals in queue order with read-time facts.
func (s *Service) List(ctx context.Context, repo Repo, filter ListFilter) ([]View, int, error) {
	filter.RepositoryID = repo.ID
	proposals, err := s.store.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	open, err := s.store.CountOpen(ctx, repo.ID)
	if err != nil {
		return nil, 0, err
	}
	views, err := s.views(ctx, repo, proposals)
	return views, open, err
}

// Get returns one proposal with read-time facts. The repository must be the
// proposal's repository.
func (s *Service) Get(ctx context.Context, repo Repo, id string) (View, error) {
	proposal, err := s.store.Get(ctx, id)
	if err != nil {
		return View{}, err
	}
	if proposal.RepositoryID != repo.ID {
		return View{}, fmt.Errorf("proposal %s: %w", id, ErrNotFound)
	}
	views, err := s.views(ctx, repo, []Proposal{proposal})
	if err != nil {
		return View{}, err
	}
	return views[0], nil
}

// RepositoryOf returns the repository a proposal belongs to.
func (s *Service) RepositoryOf(ctx context.Context, id string) (string, error) {
	proposal, err := s.store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return proposal.RepositoryID, nil
}

// views adds freshness, dynamic flags and issues. Shared repository reads
// run once per call, not once per proposal.
func (s *Service) views(ctx context.Context, repo Repo, proposals []Proposal) ([]View, error) {
	views := make([]View, 0, len(proposals))
	var shared *readContext
	for _, proposal := range proposals {
		view := View{Proposal: proposal}
		view.Issues = trailers.Validate(proposal.Message.Subject, proposal.Message.Body, proposal.Message.Trailers, trailers.ValidateOptions{AgentAuthored: !proposal.CreatedBy.IsHuman() && !proposal.Message.OperatorEdited})
		if proposal.State != StateOpen {
			view.Freshness = Freshness{State: FreshnessNotApplicable, CheckedAt: s.now()}
			views = append(views, view)
			continue
		}
		if shared == nil {
			loaded, err := s.loadReadContext(ctx, repo)
			if err != nil {
				return nil, err
			}
			shared = &loaded
		}
		view.Freshness = s.freshness(ctx, repo, proposal, *shared)
		view.Files = withDynamicFlags(proposal, *shared)
		views = append(views, view)
	}
	return views, nil
}

type readContext struct {
	head      string
	headErr   error
	staged    map[string]struct{}
	stagedErr error
	openPaths map[string][]string
}

func (s *Service) loadReadContext(ctx context.Context, repo Repo) (readContext, error) {
	var rc readContext
	rc.head, rc.headErr = repo.Git.Head(ctx)
	staged, err := repo.Git.StagedPaths(ctx)
	rc.stagedErr = err
	rc.staged = map[string]struct{}{}
	for _, path := range staged {
		rc.staged[path] = struct{}{}
	}
	open, err := s.store.List(ctx, ListFilter{RepositoryID: repo.ID, Limit: 200})
	if err != nil {
		return readContext{}, err
	}
	rc.openPaths = map[string][]string{}
	for _, proposal := range open {
		for _, file := range proposal.Files {
			rc.openPaths[file.Path] = append(rc.openPaths[file.Path], proposal.ID)
		}
	}
	return rc, nil
}

func (s *Service) freshness(ctx context.Context, repo Repo, proposal Proposal, rc readContext) Freshness {
	fresh := Freshness{State: FreshnessFresh, CurrentHead: rc.head, CheckedAt: s.now()}
	if rc.headErr != nil || rc.stagedErr != nil {
		fresh.State, fresh.Detail = FreshnessUnknown, "repository read failed: "+errors.Join(rc.headErr, rc.stagedErr).Error()
		return fresh
	}
	paths := proposal.Paths()
	contents, err := repo.Git.Hash(ctx, paths)
	if err != nil {
		fresh.State, fresh.Detail = FreshnessUnknown, "content hash failed: "+err.Error()
		return fresh
	}
	fresh.CurrentBlobs = map[string]string{}
	for _, file := range proposal.Files {
		current := contents[file.Path].ContentID()
		fresh.CurrentBlobs[file.Path] = current
		if current != file.ContentID() {
			fresh.Drifted = append(fresh.Drifted, file.Path)
		}
	}
	if rc.head != proposal.BaseHead {
		changed, err := repo.Git.ChangedPaths(ctx, proposal.BaseHead, rc.head, paths)
		if err != nil {
			fresh.State, fresh.Detail = FreshnessUnknown, "base comparison failed: "+err.Error()
			return fresh
		}
		fresh.BaseChanged = changed
	}
	inProposal := map[string]struct{}{}
	for _, path := range paths {
		inProposal[path] = struct{}{}
	}
	for path := range rc.staged {
		if _, ok := inProposal[path]; !ok {
			fresh.ForeignStaged = append(fresh.ForeignStaged, path)
		}
	}
	sort.Strings(fresh.ForeignStaged)
	switch {
	case len(fresh.Drifted) > 0:
		fresh.State = FreshnessDrifted
	case len(fresh.BaseChanged) > 0:
		fresh.State = FreshnessBaseMoved
	}
	return fresh
}

// withDynamicFlags replaces read-time flags: shared paths with other open
// proposals and paths already staged.
func withDynamicFlags(proposal Proposal, rc readContext) []File {
	files := make([]File, 0, len(proposal.Files))
	for _, file := range proposal.Files {
		copyFile := file
		copyFile.Flags = nil
		for _, flag := range file.Flags {
			if flag.Code != FlagOtherOpenProposal && flag.Code != FlagAlreadyStaged {
				copyFile.Flags = append(copyFile.Flags, flag)
			}
		}
		for _, other := range rc.openPaths[file.Path] {
			if other != proposal.ID {
				copyFile.Flags = append(copyFile.Flags, Flag{Code: FlagOtherOpenProposal, Detail: other})
			}
		}
		if _, staged := rc.staged[file.Path]; staged {
			copyFile.Flags = append(copyFile.Flags, Flag{Code: FlagAlreadyStaged})
		}
		files = append(files, copyFile)
	}
	return files
}

type EditRequest struct {
	ID               string
	ExpectedRevision int
	Subject          *string
	Body             *string
	Trailers         []trailers.Entry
	ReplaceTrailers  bool
	RemovePaths      []string
	Actor            Actor
}

// Edit changes the message or drops files and creates a new revision.
func (s *Service) Edit(ctx context.Context, repo Repo, req EditRequest) (View, []trailers.Issue, error) {
	proposal, err := s.openRevision(ctx, repo, req.ID, req.ExpectedRevision)
	if err != nil {
		return View{}, nil, err
	}
	previous := proposal.Revision
	message := proposal.Message
	var changes []string
	if req.Subject != nil && strings.TrimSpace(*req.Subject) != message.Subject {
		message.Subject, changes = *req.Subject, append(changes, "subject")
	}
	if req.Body != nil && trailers.CleanBody(*req.Body) != message.Body {
		message.Body, changes = *req.Body, append(changes, "body")
	}
	if req.ReplaceTrailers {
		var replaced []trailers.Entry
		for _, entry := range req.Trailers {
			if strings.TrimSpace(entry.Key) != "" {
				replaced = append(replaced, trailers.Entry{Key: entry.Key, Value: entry.Value})
			}
		}
		message.Trailers, changes = replaced, append(changes, "trailers")
	}
	if len(req.RemovePaths) > 0 {
		remove, err := cleanPaths(req.RemovePaths)
		if err != nil {
			return View{}, nil, err
		}
		var kept []File
		for _, file := range proposal.Files {
			if !contains(remove, file.Path) {
				kept = append(kept, file)
			}
		}
		if len(kept) == 0 {
			return View{}, nil, fmt.Errorf("%w: removing every file empties the proposal; withdraw it instead", ErrInvalid)
		}
		if len(kept) != len(proposal.Files) {
			proposal.Files, changes = kept, append(changes, fmt.Sprintf("removed %d files", len(proposal.Files)-len(kept)))
		}
	}
	if len(changes) == 0 {
		view, err := s.Get(ctx, repo, proposal.ID)
		return view, view.Issues, err
	}
	if req.Actor.IsHuman() {
		message.OperatorEdited = true
	}
	message = finishMessage(message)
	issues := trailers.Validate(message.Subject, message.Body, message.Trailers, trailers.ValidateOptions{AgentAuthored: !req.Actor.IsHuman()})
	if !req.Actor.IsHuman() && trailers.HasErrors(issues) {
		return View{}, issues, &ValidationError{Issues: issues}
	}
	now := s.now()
	proposal.Message = message
	proposal.Revision++
	proposal.UpdatedAt = now
	proposal.Digest = Digest(proposal.BaseHead, proposal.Files, message.Rendered)
	proposal.Events = append(proposal.Events, Event{Action: "edited", Actor: req.Actor, Revision: proposal.Revision, Detail: strings.Join(changes, ", "), Timestamp: now})
	if err := s.store.Update(ctx, proposal, previous, StateOpen); err != nil {
		return View{}, nil, err
	}
	view, err := s.Get(ctx, repo, proposal.ID)
	return view, issues, err
}

// Withdraw ends an open proposal. History is kept.
func (s *Service) Withdraw(ctx context.Context, repo Repo, id, reason string, actor Actor) (View, error) {
	proposal, err := s.store.Get(ctx, id)
	if err != nil {
		return View{}, err
	}
	if proposal.RepositoryID != repo.ID {
		return View{}, fmt.Errorf("proposal %s: %w", id, ErrNotFound)
	}
	if proposal.State != StateOpen {
		return View{}, fmt.Errorf("%w: proposal %s is %s", ErrNotOpen, id, proposal.State)
	}
	now := s.now()
	proposal.State, proposal.UpdatedAt = StateWithdrawn, now
	proposal.Events = append(proposal.Events, Event{Action: "withdrawn", Actor: actor, Revision: proposal.Revision, Detail: strings.TrimSpace(reason), Timestamp: now})
	if err := s.store.Update(ctx, proposal, proposal.Revision, StateOpen); err != nil {
		return View{}, err
	}
	return s.Get(ctx, repo, id)
}

// Refresh re-hashes current content into a new revision. Paths that became
// clean are dropped; when none remain the proposal is superseded. The
// message, including operator edits, is kept.
func (s *Service) Refresh(ctx context.Context, repo Repo, id string, expectedRevision int, actor Actor) (View, []string, error) {
	proposal, err := s.openRevision(ctx, repo, id, expectedRevision)
	if err != nil {
		return View{}, nil, err
	}
	previous := proposal.Revision
	snap, err := readSnapshot(ctx, repo)
	if err != nil {
		return View{}, nil, err
	}
	var anchor *Anchor
	if proposal.Evidence.AnchorID != "" {
		loaded, err := s.store.GetAnchor(ctx, proposal.Evidence.AnchorID)
		if err != nil {
			return View{}, nil, err
		}
		anchor = &loaded
	}
	files, excluded, err := buildFiles(ctx, repo, snap, proposal.Paths(), anchor, false)
	if err != nil {
		return View{}, nil, err
	}
	var dropped []string
	for _, exclusion := range excluded {
		if exclusion.Reason == "now_clean" || exclusion.Reason == "unchanged_since_anchor" {
			dropped = append(dropped, exclusion.Path)
		}
	}
	now := s.now()
	proposal.UpdatedAt = now
	if len(files) == 0 {
		proposal.State = StateSuperseded
		proposal.Excluded = excluded
		proposal.Events = append(proposal.Events, Event{Action: "superseded", Actor: actor, Revision: proposal.Revision, Detail: "every path is clean or unchanged since the anchor", Timestamp: now})
		if err := s.store.Update(ctx, proposal, previous, StateOpen); err != nil {
			return View{}, nil, err
		}
		view, err := s.Get(ctx, repo, id)
		return view, dropped, err
	}
	proposal.Files, proposal.Excluded = files, excluded
	proposal.BaseHead, proposal.Branch = snap.head, snap.branch
	proposal.Revision++
	proposal.Digest = Digest(proposal.BaseHead, proposal.Files, proposal.Message.Rendered)
	detail := "re-hashed current content"
	if len(dropped) > 0 {
		detail += "; dropped " + strings.Join(dropped, ", ")
	}
	proposal.Events = append(proposal.Events, Event{Action: "refreshed", Actor: actor, Revision: proposal.Revision, Detail: detail, Timestamp: now})
	if err := s.store.Update(ctx, proposal, previous, StateOpen); err != nil {
		return View{}, nil, err
	}
	view, err := s.Get(ctx, repo, id)
	return view, dropped, err
}

func (s *Service) openRevision(ctx context.Context, repo Repo, id string, expectedRevision int) (Proposal, error) {
	proposal, err := s.store.Get(ctx, id)
	if err != nil {
		return Proposal{}, err
	}
	if proposal.RepositoryID != repo.ID {
		return Proposal{}, fmt.Errorf("proposal %s: %w", id, ErrNotFound)
	}
	if proposal.State != StateOpen {
		return Proposal{}, fmt.Errorf("%w: proposal %s is %s", ErrNotOpen, id, proposal.State)
	}
	if expectedRevision > 0 && expectedRevision != proposal.Revision {
		return Proposal{}, fmt.Errorf("%w: expected revision %d, current %d", ErrConflict, expectedRevision, proposal.Revision)
	}
	return proposal, nil
}

// SubjectEvidence is the proposal part of an apply intent digest: the
// proposal digest and the current content of every proposal path. A change
// to either invalidates an issued intent.
func (s *Service) SubjectEvidence(ctx context.Context, repo Repo, subjectContext string) ([]byte, error) {
	id, revision, ok := ParseSubjectContext(subjectContext)
	if !ok {
		return nil, fmt.Errorf("%w: subject context must be proposal:<id>@<revision>", ErrInvalid)
	}
	proposal, err := s.openRevision(ctx, repo, id, revision)
	if err != nil {
		return nil, err
	}
	contents, err := repo.Git.Hash(ctx, proposal.Paths())
	if err != nil {
		return nil, err
	}
	var builder strings.Builder
	builder.WriteString(proposal.Digest)
	for _, path := range proposal.Paths() {
		builder.WriteString("\x00" + path + "=" + contents[path].ContentID())
	}
	return []byte(builder.String()), nil
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok || value == "" {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func subtract(values, remove []string) []string {
	var result []string
	for _, value := range values {
		if !contains(remove, value) {
			result = append(result, value)
		}
	}
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
