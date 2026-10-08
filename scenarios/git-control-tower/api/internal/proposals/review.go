package proposals

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"git-control-tower/internal/trailers"
)

func Captured(p Proposal) bool {
	if len(p.Files) == 0 {
		return false
	}
	for _, f := range p.Files {
		if !f.Captured {
			return false
		}
	}
	return true
}

func (s *Service) Revision(ctx context.Context, repo Repo, id string, revision int) (View, error) {
	if revision <= 0 {
		return s.Get(ctx, repo, id)
	}
	p, err := s.store.Revision(ctx, id, revision)
	if err != nil {
		return View{}, err
	}
	if p.RepositoryID != repo.ID {
		return View{}, ErrNotFound
	}
	return View{Proposal: p, Freshness: Freshness{State: FreshnessNotApplicable, Detail: "immutable captured revision", CheckedAt: s.now()}}, nil
}

func (s *Service) ReviewFile(ctx context.Context, repo Repo, id string, revision int, path string) (File, []byte, []byte, error) {
	view, err := s.Revision(ctx, repo, id, revision)
	if err != nil {
		return File{}, nil, nil, err
	}
	for _, f := range view.Files {
		if f.Path != path {
			continue
		}
		if !f.Captured {
			return File{}, nil, nil, fmt.Errorf("%w: this legacy revision has no captured contents; adopt current contents explicitly", ErrInvalid)
		}
		base, err := s.store.Blob(ctx, f.BaseBlobID)
		if err != nil {
			return File{}, nil, nil, err
		}
		data, err := s.store.Blob(ctx, f.ContentID())
		return f, base, data, err
	}
	return File{}, nil, nil, ErrNotFound
}

func (s *Service) PrepareCommit(ctx context.Context, repo Repo, id string, revision int, check CandidateCheck, skipHooks bool) (CommitOutcome, error) {
	p, err := s.openRevision(ctx, repo, id, revision)
	if err != nil {
		return CommitOutcome{}, err
	}
	return repo.Git.PrepareCommit(ctx, p, s.store.Blob, check, skipHooks)
}

type RevisionReference struct {
	ID       string
	Revision int
}
type CombineRequest struct {
	Sources       []RevisionReference
	Subject, Body string
	Actor         Actor
}

// Combine uses only the explicitly named captured revisions. It never reads
// live bytes or silently chooses one of two different versions of a file.
func (s *Service) Combine(ctx context.Context, repo Repo, req CombineRequest) (View, error) {
	if len(req.Sources) < 2 || len(req.Sources) > 20 {
		return View{}, fmt.Errorf("%w: select between 2 and 20 proposals", ErrInvalid)
	}
	seen := map[string]bool{}
	byPath := map[string]File{}
	var sources []Proposal
	var entries []trailers.Entry
	var folds []Fold
	for _, ref := range req.Sources {
		if ref.Revision <= 0 || seen[ref.ID] {
			return View{}, fmt.Errorf("%w: unique IDs and exact source revisions are required", ErrInvalid)
		}
		seen[ref.ID] = true
		p, err := s.openRevision(ctx, repo, ref.ID, ref.Revision)
		if err != nil {
			return View{}, err
		}
		if !Captured(p) {
			return View{}, fmt.Errorf("%w: %s has no retained capture; explicitly adopt its current content first", ErrInvalid, ref.ID)
		}
		if len(sources) > 0 && (p.BaseHead != sources[0].BaseHead || p.Branch != sources[0].Branch) {
			return View{}, fmt.Errorf("%w: source bases/destinations differ; review current-content revisions first", ErrInvalid)
		}
		for _, f := range p.Files {
			if old, ok := byPath[f.Path]; ok {
				if old.ContentID() != f.ContentID() || old.Mode != f.Mode || old.BaseBlobID != f.BaseBlobID || old.BaseMode != f.BaseMode {
					return View{}, fmt.Errorf("%w: different captured versions of %s; compare and refresh explicitly", ErrConflict, f.Path)
				}
				old.Flags = mergeFlags(old.Flags, f.Flags)
				byPath[f.Path] = old
			} else {
				byPath[f.Path] = f
			}
		}
		for _, e := range p.Message.Trailers {
			if !strings.EqualFold(e.Key, "Vrooli-Proposal") {
				entries = append(entries, e)
			}
		}
		folds = append(folds, Fold{ID: p.ID, Revision: p.Revision, Epoch: p.Work.Epoch, EffortRef: p.Work.EffortRef, Subject: p.Message.Subject, Evidence: p.Evidence, Reason: "explicit_combination", Paths: p.Paths()})
		sources = append(sources, p)
	}
	if len(byPath) > maxProposalPaths {
		return View{}, fmt.Errorf("%w: combined proposal exceeds path limit", ErrInvalid)
	}
	files := make([]File, 0, len(byPath))
	for _, f := range byPath {
		files = append(files, f)
	}
	if size, err := s.store.captureSize(ctx, files); err != nil { return View{}, err } else if size > MaxCaptureBytes { return View{}, fmt.Errorf("%w: combined capture exceeds 64 MiB; split the package", ErrInvalid) }
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	now := s.now()
	p := Proposal{ID: s.newID("gctp-"), RepositoryID: repo.ID, State: StateOpen, Revision: 1, CreatedBy: req.Actor, BaseHead: sources[0].BaseHead, Branch: sources[0].Branch, Files: files, Folds: folds, CreatedAt: now, UpdatedAt: now}
	p.Work.EffortRef = sources[0].Work.EffortRef
	for _, source := range sources {
		if source.Work.EffortRef != sources[0].Work.EffortRef {
			p.Work.EffortRef = ""
		}
		p.Work.RunIDs = append(p.Work.RunIDs, source.Work.RunIDs...)
		p.Work.Plans = append(p.Work.Plans, source.Work.Plans...)
	}
	p.Work.RunIDs = uniqueSorted(p.Work.RunIDs)
	p.Work.Plans = uniqueSorted(p.Work.Plans)
	entries = append(entries, trailers.Entry{Key: "Vrooli-Proposal", Value: p.ID})
	p.Message = finishMessage(Message{Subject: req.Subject, Body: req.Body, Trailers: trailers.Normalize(entries), OperatorEdited: req.Actor.IsHuman()})
	issues := trailers.Validate(p.Message.Subject, p.Message.Body, p.Message.Trailers, trailers.ValidateOptions{AgentAuthored: !req.Actor.IsHuman()})
	if trailers.HasErrors(issues) {
		return View{}, &ValidationError{Issues: issues}
	}
	p.Digest = Digest(p.BaseHead, p.Files, p.Message.Rendered)
	p.Events = []Event{{Action: "combined", Actor: req.Actor, Revision: 1, Detail: fmt.Sprintf("explicit union of %d source revisions; acceptance evidence remains source-bound", len(sources)), Timestamp: now}}
	if err := s.store.Combine(ctx, p, sources, req.Actor); err != nil {
		return View{}, err
	}
	return s.Get(ctx, repo, p.ID)
}

func mergeFlags(a, b []Flag) []Flag {
	result := append([]Flag(nil), a...)
	for _, f := range b {
		found := false
		for _, old := range result {
			if old.Code == f.Code && old.Detail == f.Detail {
				found = true
				break
			}
		}
		if !found {
			result = append(result, f)
		}
	}
	return result
}

func UncertainPaths(p Proposal) []string {
	var result []string
	for _, f := range p.Files {
		for _, flag := range f.Flags {
			if flag.Code == FlagMixedPrior || flag.Code == FlagNoAnchor || flag.Code == FlagOutsideAnchorScope {
				result = append(result, f.Path)
				break
			}
		}
	}
	return result
}

func (s *Service) AvailableRevisions(ctx context.Context, repo Repo, id string) ([]int,error) {
	p,err:=s.store.Get(ctx,id)
	if err!=nil {return nil,err}
	if p.RepositoryID != repo.ID {return nil,ErrNotFound}
	return s.store.AvailableRevisions(ctx,id)
}
