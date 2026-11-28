package proposals

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"git-control-tower/internal/policygate"

	"github.com/vrooli/cli-core/cliutil"
)

type ApplyRequest struct {
	ID                   string
	Revision             int
	Actor                Actor
	AcknowledgePriorWork bool
	KeepLaterChanges     bool
}

// PrecommitOutcome is the configured pre-commit result recorded during apply.
type PrecommitOutcome struct {
	Status     string
	Command    string
	ExitCode   int
	Summary    string
	Stdout     string
	Stderr     string
	DurationMs int64
	Timestamp  time.Time
}

// CommitOutcome is what the commit step reports. A non-empty Failure means
// no commit was created.
type CommitOutcome struct {
	OID       string
	Failure   string
	Precommit *PrecommitOutcome
	// Candidate facts are inspected before Publish can advance the real ref.
	Facts   *CommitFacts
	Publish func(context.Context) (string, error)
	Cleanup func()
}

// CommitFunc prepares a checked private candidate. Apply verifies its facts
// before its Publish closure may advance the operator’s repository.
type CommitFunc func(ctx context.Context, message string) (CommitOutcome, error)

// Refusal explains why apply did nothing.
type Refusal struct {
	Code   string
	Paths  []string
	Detail string
}

type ApplyResult struct {
	Success   bool
	CommitOID string
	View      View
	Refusal   *Refusal
	Error     string
	Precommit *PrecommitOutcome
	Verified  bool
	Notes     []string
}

// Apply stages exactly the proposal paths and commits the rendered message.
// The caller holds the repository lock and has consumed a single-use intent
// for OperationApply. Strict mode refuses working-content drift. Preserved-files
// mode stages retained contents and refuses different staged versions. Both
// refuse commits since the base touching paths and foreign staging. On any
// failure after staging, it restores the index entries of the proposal paths.
func (s *Service) Apply(ctx context.Context, repo Repo, req ApplyRequest, commit CommitFunc) (ApplyResult, error) {
	if err := requireApplyAuthority(ctx); err != nil {
		return ApplyResult{}, err
	}
	if commit == nil {
		return ApplyResult{}, fmt.Errorf("commit step is required")
	}
	proposal, err := s.store.Get(ctx, req.ID)
	if err != nil {
		return ApplyResult{}, err
	}
	if proposal.RepositoryID != repo.ID {
		return ApplyResult{}, fmt.Errorf("proposal %s: %w", req.ID, ErrNotFound)
	}
	if proposal.State != StateOpen {
		return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: RefuseState, Detail: fmt.Sprintf("proposal is %s; a proposal commits at most once", proposal.State)}, false)
	}
	if proposal.Revision != req.Revision {
		return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: RefuseRevision, Detail: fmt.Sprintf("approved revision %d, current revision %d; review it again", req.Revision, proposal.Revision)}, true)
	}
	if !Captured(proposal) {
		return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: "capture_unavailable", Detail: "This legacy proposal has no retained content. Explicitly adopt and review current contents first."}, true)
	}
	if paths := UncertainPaths(proposal); len(paths) > 0 && !req.AcknowledgePriorWork {
		return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: "prior_work_acknowledgment", Paths: paths, Detail: "Explicitly acknowledge that these whole files may include prior or unattributed work."}, true)
	}
	branch, err := repo.Git.Branch(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	if proposal.Branch == "" {
		return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: "destination_branch_required", Detail: "Check out a named destination branch and adopt a new revision before approval."}, true)
	}
	if branch != proposal.Branch {
		return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: "branch_changed", Detail: "The destination branch changed; review a new revision."}, true)
	}
	paths := proposal.Paths()

	if !req.KeepLaterChanges {
		contents, err := repo.Git.Hash(ctx, paths)
		if err != nil {
			return ApplyResult{}, err
		}
		var drifted []string
		for _, file := range proposal.Files {
			if contents[file.Path].ContentID() != file.ContentID() || contents[file.Path].SHA256 != file.SHA256 || (!file.Deleted && contents[file.Path].Mode != file.Mode) {
				drifted = append(drifted, file.Path)
			}
		}
		if len(drifted) > 0 {
			return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: RefuseContentDrift, Paths: drifted, Detail: "content changed since the proposal; refresh it or remove the files"}, true)
		}
	}

	head, err := repo.Git.Head(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	if head != proposal.BaseHead {
		changed, err := repo.Git.ChangedPaths(ctx, proposal.BaseHead, head, paths)
		if err != nil {
			return ApplyResult{}, err
		}
		if len(changed) > 0 {
			return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: RefuseBaseMoved, Paths: changed, Detail: "commits since the proposal base touch these paths; refresh the proposal"}, true)
		}
	}
	staged, err := repo.Git.StagedPaths(ctx)
	if err != nil {
		return ApplyResult{}, err
	}
	if foreign := subtract(uniqueSorted(staged), paths); len(foreign) > 0 {
		return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: RefuseForeignStaged, Paths: foreign, Detail: "other paths are staged; unstage them first so the commit holds only the proposal"}, true)
	}

	prior, err := repo.Git.IndexEntries(ctx, paths)
	if err != nil {
		return ApplyResult{}, err
	}
	restore := func(cause string) []string {
		restoreCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		if restoreErr := repo.Git.RestoreIndex(restoreCtx, paths, prior); restoreErr != nil {
			return []string{cause, "index restore failed: " + restoreErr.Error()}
		}
		return []string{cause, "index was restored"}
	}
	if req.KeepLaterChanges {
		stagedSet := map[string]bool{}
		for _, path := range staged {
			stagedSet[path] = true
		}
		var different []string
		for _, file := range proposal.Files {
			if !stagedSet[file.Path] {
				continue
			}
			entry, exists := prior[file.Path]
			if (file.Deleted && exists) || (!file.Deleted && (!exists || entry.BlobID != file.BlobID || entry.Mode != file.Mode)) {
				different = append(different, file.Path)
			}
		}
		if len(different) > 0 {
			return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: "staged_version_differs", Paths: different, Detail: "Different versions of these files are staged. Review or unstage them first; their staged contents were left unchanged."}, true)
		}
	}
	stage := func() error {
		if req.KeepLaterChanges {
			return repo.Git.StageCaptured(ctx, proposal.Files, s.store.Blob)
		}
		return repo.Git.Stage(ctx, paths)
	}
	if err := stage(); err != nil {
		notes := restore("stage failed: " + err.Error())
		return ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Error: strings.Join(notes, "; ")}, nil
	}
	if mismatch, err := verifyIndex(ctx, repo.Git, proposal); err != nil || len(mismatch) > 0 {
		detail := "staged index differs from the proposal"
		if err != nil {
			detail = "staged index could not be verified: " + err.Error()
		}
		notes := restore(detail)
		return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: RefuseIndexMismatch, Paths: mismatch, Detail: strings.Join(notes, "; ")}, true)
	}

	outcome, err := commit(ctx, proposal.Message.Rendered)
	if outcome.Cleanup != nil {
		defer outcome.Cleanup()
	}
	if err != nil || outcome.Failure != "" {
		cause := outcome.Failure
		if err != nil {
			cause = err.Error()
		}
		notes := restore(cause)
		result := ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Precommit: outcome.Precommit, Error: strings.Join(notes, "; ")}
		if outcome.Precommit != nil && outcome.Precommit.Status != "passed" {
			result.Refusal = &Refusal{Code: RefusePrecommit, Detail: outcome.Precommit.Summary}
		}
		return result, nil
	}
	if outcome.Facts == nil || outcome.Publish == nil {
		notes := restore("candidate publisher did not provide verifiable immutable facts")
		return ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Error: strings.Join(notes, "; ")}, nil
	}
	if verified, notes := verifyFacts(proposal, *outcome.Facts); !verified {
		notes = append(notes, restore("candidate changed during checks; no repository ref was published")...)
		return ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Precommit: outcome.Precommit, Refusal: &Refusal{Code: RefuseIndexMismatch, Detail: strings.Join(notes, "; ")}}, nil
	}
	// Checks can take minutes. Recheck the shared inputs immediately before
	// publication; the isolated candidate cannot carry foreign index entries.
	currentHead, headErr := repo.Git.Head(ctx)
	currentBranch, branchErr := repo.Git.Branch(ctx)
	if headErr != nil || branchErr != nil || currentHead != head || currentBranch != proposal.Branch {
		notes := restore("repository destination changed during checks; review again")
		return ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Refusal: &Refusal{Code: "destination_changed", Detail: strings.Join(notes, "; ")}}, nil
	}
	if !req.KeepLaterChanges {
		current, hashErr := repo.Git.Hash(ctx, paths)
		if hashErr != nil {
			return ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Error: strings.Join(restore(hashErr.Error()), "; ")}, nil
		}
		for _, file := range proposal.Files {
			if current[file.Path].ContentID() != file.ContentID() || current[file.Path].SHA256 != file.SHA256 || (!file.Deleted && current[file.Path].Mode != file.Mode) {
				notes := restore("working content changed during checks; no commit was published")
				return ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Refusal: &Refusal{Code: RefuseContentDrift, Paths: []string{file.Path}, Detail: strings.Join(notes, "; ")}}, nil
			}
		}
	}

	if mismatch, err := verifyIndex(ctx, repo.Git, proposal); err != nil || len(mismatch) > 0 {
		notes := restore("index changed during checks; no commit was published")
		return ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Refusal: &Refusal{Code: RefuseIndexMismatch, Paths: mismatch, Detail: strings.Join(notes, "; ")}}, nil
	}
	oid, committed, err := s.store.Publish(ctx, proposal, req.Actor, outcome.Publish, s.now())
	if err != nil {
		if oid != "" {
			return ApplyResult{CommitOID: oid, View: s.viewOrBare(ctx, repo, committed), Error: "Commit outcome needs recovery review: " + err.Error()}, nil
		}
		return ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Error: strings.Join(restore("publish refused: "+err.Error()), "; ")}, nil
	}
	outcome.OID = oid

	// The commit exists from here on; the index is never restored over it.
	if strings.TrimSpace(outcome.OID) == "" {
		outcome.OID = "unresolved"
	}
	verified, notes := verifyCommit(ctx, repo.Git, proposal, outcome.OID)
	view := s.viewOrBare(ctx, repo, committed)
	if !verified {
		return ApplyResult{CommitOID: outcome.OID, View: view, Precommit: outcome.Precommit, Error: "A commit was created but did not match the approved proposal; inspect it before continuing.", Notes: notes}, nil
	}
	if req.KeepLaterChanges {
		notes = append(notes, "Committed the proposal version. Working files were left unchanged; later edits remain uncommitted.")
	}
	return ApplyResult{Success: true, CommitOID: outcome.OID, View: view, Precommit: outcome.Precommit, Verified: verified, Notes: notes}, nil
}

// requireApplyAuthority is the domain guard: a verified human principal and
// a consumed intent for OperationApply, independent of transport.
func requireApplyAuthority(ctx context.Context) error {
	principal, ok := policygate.PrincipalFromContext(ctx)
	if !ok || principal.Kind != cliutil.CallerKindHuman {
		return ErrForbidden
	}
	intent, ok := policygate.ConsumedIntentFromContext(ctx)
	if !ok || intent.Operation != OperationApply {
		return ErrForbidden
	}
	return nil
}

// verifyIndex closes the time-of-check gap after staging: the staged set must
// be exactly the proposal paths and each staged blob must be the proposal's.
func verifyIndex(ctx context.Context, git Git, proposal Proposal) ([]string, error) {
	paths := proposal.Paths()
	staged, err := git.StagedPaths(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := git.IndexEntries(ctx, paths)
	if err != nil {
		return nil, err
	}
	stagedSet := map[string]struct{}{}
	var mismatch []string
	for _, path := range staged {
		stagedSet[path] = struct{}{}
		if !contains(paths, path) {
			mismatch = append(mismatch, path)
		}
	}
	for _, file := range proposal.Files {
		entry, inIndex := entries[file.Path]
		_, isStaged := stagedSet[file.Path]
		switch {
		case !isStaged:
			mismatch = append(mismatch, file.Path)
		case file.Deleted && inIndex:
			mismatch = append(mismatch, file.Path)
		case !file.Deleted && (!inIndex || entry.BlobID != file.BlobID || entry.Mode != file.Mode):
			mismatch = append(mismatch, file.Path)
		}
	}
	return uniqueSorted(mismatch), nil
}

// verifyCommit compares the created commit with the proposal. A hook that
// rewrote a file or the message shows up here; the commit stands, and the
// mismatch is reported rather than hidden.
func verifyCommit(ctx context.Context, git Git, proposal Proposal, oid string) (bool, []string) {
	facts, err := git.CommitFacts(ctx, oid)
	if err != nil {
		return false, []string{"post-commit verification unavailable: " + err.Error()}
	}
	return verifyFacts(proposal, facts)
}

func verifyFacts(proposal Proposal, facts CommitFacts) (bool, []string) {
	var notes []string
	expected := map[string]string{}
	for _, file := range proposal.Files {
		expected[file.Path] = file.ContentID()
	}
	var paths []string
	for path := range facts.Blobs {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		want, ok := expected[path]
		switch {
		case !ok:
			notes = append(notes, "commit contains a path outside the proposal: "+path)
		case want != facts.Blobs[path]:
			notes = append(notes, "committed content differs from the proposal: "+path)
		case facts.Modes[path] != fileMode(proposal, path):
			notes = append(notes, "committed mode differs from the proposal: "+path)
		}
	}
	for _, path := range proposal.Paths() {
		if _, ok := facts.Blobs[path]; !ok {
			notes = append(notes, "proposal path missing from the commit: "+path)
		}
	}
	if facts.Message != proposal.Message.Rendered {
		notes = append(notes, "committed message differs from the rendered message")
	}
	return len(notes) == 0, notes
}

func fileMode(p Proposal, path string) string {
	for _, f := range p.Files {
		if f.Path == path {
			return f.Mode
		}
	}
	return ""
}

// refuse records the refusal in the proposal history (open proposals only)
// and returns it. Nothing was staged or committed.
func (s *Service) refuse(ctx context.Context, repo Repo, proposal Proposal, actor Actor, refusal Refusal, record bool) (ApplyResult, error) {
	if record && proposal.State == StateOpen {
		now := s.now()
		updated := proposal
		updated.UpdatedAt = now
		detail := refusal.Code
		if len(refusal.Paths) > 0 {
			detail += ": " + strings.Join(refusal.Paths, ", ")
		}
		updated.Events = append(append([]Event(nil), proposal.Events...), Event{Action: "apply_refused", Actor: actor, Revision: proposal.Revision, Detail: detail, Timestamp: now})
		if err := s.store.Update(ctx, updated, proposal.Revision, StateOpen); err == nil {
			proposal = updated
		}
	}
	return ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Refusal: &refusal}, nil
}

func (s *Service) viewOrBare(ctx context.Context, repo Repo, proposal Proposal) View {
	if proposal.ID == "" {
		return View{}
	}
	views, err := s.views(ctx, repo, []Proposal{proposal})
	if err != nil || len(views) == 0 {
		return View{Proposal: proposal, Freshness: Freshness{State: FreshnessUnknown, CheckedAt: s.now()}}
	}
	return views[0]
}
