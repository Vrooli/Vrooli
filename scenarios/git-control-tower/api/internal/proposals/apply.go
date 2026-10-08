package proposals

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"git-control-tower/internal/policygate"

	"github.com/vrooli/cli-core/cliutil"
)

type ApplyRequest struct {
	ID       string
	Revision int
	Actor    Actor
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
}

// CommitOutcome is what the commit step reports. A non-empty Failure means
// no commit was created.
type CommitOutcome struct {
	OID       string
	Failure   string
	Precommit *PrecommitOutcome
}

// CommitFunc commits the current index with an exact message. Production runs
// the configured pre-commit check and git commit as the repository's own
// identity; the operator is the author.
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
// for OperationApply. Apply refuses on content drift, on commits since the
// base that touch the paths, and on staged paths outside the proposal. On any
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
	paths := proposal.Paths()

	contents, err := repo.Git.Hash(ctx, paths)
	if err != nil {
		return ApplyResult{}, err
	}
	var drifted []string
	for _, file := range proposal.Files {
		if contents[file.Path].ContentID() != file.ContentID() {
			drifted = append(drifted, file.Path)
		}
	}
	if len(drifted) > 0 {
		return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: RefuseContentDrift, Paths: drifted, Detail: "content changed since the proposal; refresh it or remove the files"}, true)
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
		if restoreErr := repo.Git.RestoreIndex(context.WithoutCancel(ctx), paths, prior); restoreErr != nil {
			return []string{cause, "index restore failed: " + restoreErr.Error()}
		}
		return []string{cause}
	}
	if err := repo.Git.Stage(ctx, paths); err != nil {
		notes := restore("stage failed: " + err.Error())
		return ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Error: strings.Join(notes, "; ")}, nil
	}
	if mismatch, err := verifyIndex(ctx, repo.Git, proposal); err != nil || len(mismatch) > 0 {
		detail := "staged index differs from the proposal"
		if err != nil {
			detail = "staged index could not be verified: " + err.Error()
		}
		restore(detail)
		return s.refuse(ctx, repo, proposal, req.Actor, Refusal{Code: RefuseIndexMismatch, Paths: mismatch, Detail: detail + "; the index was restored"}, true)
	}

	outcome, err := commit(ctx, proposal.Message.Rendered)
	if err != nil || outcome.Failure != "" {
		cause := outcome.Failure
		if err != nil {
			cause = err.Error()
		}
		notes := restore(cause)
		result := ApplyResult{View: s.viewOrBare(ctx, repo, proposal), Precommit: outcome.Precommit, Error: strings.Join(notes, "; ") + "; the index was restored"}
		if outcome.Precommit != nil && outcome.Precommit.Status != "passed" {
			result.Refusal = &Refusal{Code: RefusePrecommit, Detail: outcome.Precommit.Summary}
		}
		return result, nil
	}

	// The commit exists from here on; the index is never restored over it.
	if strings.TrimSpace(outcome.OID) == "" {
		outcome.OID = "unresolved"
	}
	verified, notes := verifyCommit(ctx, repo.Git, proposal, outcome.OID)
	committed, err := s.markCommitted(ctx, proposal.ID, outcome.OID, req.Actor, verified, notes)
	if err != nil {
		notes = append(notes, "commit created but the proposal record was not updated: "+err.Error())
	}
	view := s.viewOrBare(ctx, repo, committed)
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
		case !file.Deleted && (!inIndex || entry.BlobID != file.BlobID):
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
		}
	}
	for _, path := range proposal.Paths() {
		if _, ok := facts.Blobs[path]; !ok {
			notes = append(notes, "proposal path missing from the commit: "+path)
		}
	}
	if strings.TrimSpace(facts.Message) != strings.TrimSpace(proposal.Message.Rendered) {
		notes = append(notes, "committed message differs from the rendered message")
	}
	return len(notes) == 0, notes
}

func (s *Service) markCommitted(ctx context.Context, id, oid string, actor Actor, verified bool, notes []string) (Proposal, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		current, err := s.store.Get(context.WithoutCancel(ctx), id)
		if err != nil {
			return Proposal{}, err
		}
		if current.State != StateOpen {
			return current, fmt.Errorf("proposal became %s during apply", current.State)
		}
		now := s.now()
		committedAt := now
		current.State, current.CommitOID, current.CommittedAt, current.UpdatedAt = StateCommitted, oid, &committedAt, now
		detail := "commit " + oid
		if !verified {
			detail += "; verification: " + strings.Join(notes, "; ")
		}
		current.Events = append(current.Events, Event{Action: "committed", Actor: actor, Revision: current.Revision, Detail: detail, Timestamp: now})
		if lastErr = s.store.Update(context.WithoutCancel(ctx), current, current.Revision, StateOpen); lastErr == nil {
			return current, nil
		}
		if !errors.Is(lastErr, ErrConflict) {
			break
		}
	}
	return Proposal{}, lastErr
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
