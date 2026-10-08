package proposals

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"git-control-tower/internal/trailers"
)

// Fold reasons recorded on the combined proposal and in the folded
// proposal's history.
const (
	FoldSameEpoch = "same_epoch"
	FoldOverlap   = "overlapping_paths"
	FoldDrifted   = "drifted"
)

// maxRunTrailers matches the Vrooli-Run multiplicity in the trailer registry.
const maxRunTrailers = 20

// foldPlan is the set of open proposals of one effort that a new proposal
// replaces or absorbs. A slow review never blocks an acceptance: instead of
// refusing when an effort has many open proposals, a new proposal folds every
// open proposal of its effort that shares a path with it, that has drifted,
// or that shares a path with one already folded. Proposals that share
// nothing with the result and are still fresh stay separate.
type foldPlan struct {
	// replaced holds same-epoch proposals: their listed files give way to
	// the new list, and only the files they themselves folded are carried.
	replaced []Proposal
	// folded holds earlier epochs carried whole, oldest first.
	folded []Proposal
	// carried maps each carried path to the folded proposal file it comes
	// from. Listed paths are never carried.
	carried map[string]carriedFile
	// folds records every absorbed proposal, transitively, oldest first.
	folds []Fold
	// overlaps names, per folded proposal ID, the paths that triggered it.
	overlaps map[string][]string
}

type carriedFile struct {
	file   File
	fromID string
}

func (s *Service) planFold(ctx context.Context, repo Repo, work Work, listed []string) (foldPlan, error) {
	plan := foldPlan{carried: map[string]carriedFile{}, overlaps: map[string][]string{}}
	if work.EffortRef == "" {
		return plan, nil
	}
	open, err := s.store.List(ctx, ListFilter{RepositoryID: repo.ID, EffortRef: work.EffortRef, Limit: 200})
	if err != nil {
		return plan, err
	}
	if len(open) == 0 {
		return plan, nil
	}
	drifted, err := driftedProposals(ctx, repo, open)
	if err != nil {
		return plan, err
	}
	covered := map[string]struct{}{}
	for _, path := range listed {
		covered[path] = struct{}{}
	}
	cover := func(files []File) {
		for _, file := range files {
			covered[file.Path] = struct{}{}
		}
	}
	reasons := map[string]string{}
	pending := make([]Proposal, 0, len(open))
	for _, proposal := range open {
		if work.Epoch != "" && proposal.Work.Epoch == work.Epoch {
			plan.replaced = append(plan.replaced, proposal)
			reasons[proposal.ID] = FoldSameEpoch
			cover(foldedFiles(proposal))
			continue
		}
		pending = append(pending, proposal)
	}
	// Folding grows the covered set, which can make another open proposal
	// overlap, so repeat until nothing more joins.
	for changed := true; changed; {
		changed = false
		remaining := pending[:0]
		for _, proposal := range pending {
			var shared []string
			for _, file := range proposal.Files {
				if _, ok := covered[file.Path]; ok {
					shared = append(shared, file.Path)
				}
			}
			reason := ""
			switch {
			case len(shared) > 0:
				reason = FoldOverlap
			case drifted[proposal.ID]:
				reason = FoldDrifted
			}
			if reason == "" || len(covered)+len(proposal.Files)-len(shared) > maxProposalPaths {
				remaining = append(remaining, proposal)
				continue
			}
			plan.folded = append(plan.folded, proposal)
			reasons[proposal.ID], plan.overlaps[proposal.ID] = reason, shared
			cover(proposal.Files)
			changed = true
		}
		pending = remaining
	}
	sort.SliceStable(plan.folded, func(i, j int) bool { return plan.folded[i].CreatedAt.Before(plan.folded[j].CreatedAt) })

	listedSet := map[string]struct{}{}
	for _, path := range listed {
		listedSet[path] = struct{}{}
	}
	carry := func(proposal Proposal, files []File) {
		for _, file := range files {
			if _, isListed := listedSet[file.Path]; isListed {
				continue
			}
			// A newer proposal's file wins over an older one for the same
			// path; both carry current content after re-hashing.
			plan.carried[file.Path] = carriedFile{file: file, fromID: proposal.ID}
		}
	}
	seenFold := map[string]struct{}{}
	addFold := func(fold Fold) {
		if _, ok := seenFold[fold.ID]; !ok {
			seenFold[fold.ID] = struct{}{}
			plan.folds = append(plan.folds, fold)
		}
	}
	for _, proposal := range plan.replaced {
		for _, fold := range proposal.Folds {
			addFold(fold)
		}
		carry(proposal, foldedFiles(proposal))
	}
	for _, proposal := range plan.folded {
		for _, fold := range proposal.Folds {
			addFold(fold)
		}
		addFold(Fold{ID: proposal.ID, Epoch: proposal.Work.Epoch, Subject: proposal.Message.Subject, Reason: reasons[proposal.ID], Paths: proposal.Paths()})
		carry(proposal, proposal.Files)
	}
	return plan, nil
}

// foldedFiles returns the files a proposal holds for proposals it folded,
// including folded paths it also listed itself.
func foldedFiles(proposal Proposal) []File {
	foldedPaths := map[string]struct{}{}
	for _, fold := range proposal.Folds {
		for _, path := range fold.Paths {
			foldedPaths[path] = struct{}{}
		}
	}
	var files []File
	for _, file := range proposal.Files {
		_, folded := foldedPaths[file.Path]
		if folded || strings.HasPrefix(file.Source, sourceFolded) {
			files = append(files, file)
		}
	}
	return files
}

const sourceFolded = "folded:"

// driftedProposals reports which proposals no longer match current content.
// One hash call covers every proposal.
func driftedProposals(ctx context.Context, repo Repo, open []Proposal) (map[string]bool, error) {
	var paths []string
	for _, proposal := range open {
		paths = append(paths, proposal.Paths()...)
	}
	contents, err := repo.Git.Hash(ctx, uniqueSorted(paths))
	if err != nil {
		return nil, err
	}
	drifted := map[string]bool{}
	for _, proposal := range open {
		for _, file := range proposal.Files {
			if contents[file.Path].ContentID() != file.ContentID() {
				drifted[proposal.ID] = true
				break
			}
		}
	}
	return drifted, nil
}

// carriedFiles re-hashes carried paths at current content. A path that
// became clean or is now a directory is excluded with a reason, never
// silently dropped. Stored flags travel with the file.
func carriedFiles(ctx context.Context, repo Repo, snap snapshot, carried map[string]carriedFile) ([]File, []Exclusion, error) {
	if len(carried) == 0 {
		return nil, nil, nil
	}
	var dirty []string
	var excluded []Exclusion
	for path, item := range carried {
		if _, ok := snap.xy[path]; ok {
			dirty = append(dirty, path)
			continue
		}
		excluded = append(excluded, Exclusion{Path: path, Reason: "now_clean", Detail: "folded from " + item.fromID + "; no uncommitted change remains"})
	}
	dirty = uniqueSorted(dirty)
	contents, err := repo.Git.Hash(ctx, dirty)
	if err != nil {
		return nil, nil, err
	}
	files := make([]File, 0, len(dirty))
	for _, path := range dirty {
		item, content := carried[path], contents[path]
		if content.BlobID == DirectoryBlob {
			excluded = append(excluded, Exclusion{Path: path, Reason: "directory", Detail: "folded from " + item.fromID + "; directories and submodules are not supported"})
			continue
		}
		file := File{Path: path, Kind: changeKind(snap.xy[path], content), SHA256: content.SHA256, BlobID: content.BlobID, Deleted: content.Deleted, Source: sourceFolded + item.fromID}
		for _, flag := range item.file.Flags {
			if flag.Code != FlagOtherOpenProposal && flag.Code != FlagAlreadyStaged {
				file.Flags = append(file.Flags, flag)
			}
		}
		if file.Deleted {
			file.SHA256, file.BlobID = "", ""
		}
		files = append(files, file)
	}
	return files, excluded, nil
}

// foldWork adds the folded epochs, runs and plans to the new work, newest
// epoch's runs first so the run cap drops the oldest. It returns how many
// runs the cap left out of the trailers.
func foldWork(work Work, plan foldPlan) (Work, int) {
	if len(plan.folds) == 0 {
		return work, 0
	}
	for _, fold := range plan.folds {
		if fold.Epoch != "" && fold.Epoch != work.Epoch && !contains(work.FoldedEpochs, fold.Epoch) {
			work.FoldedEpochs = append(work.FoldedEpochs, fold.Epoch)
		}
	}
	omitted := 0
	addRun := func(run string) {
		if contains(work.RunIDs, run) {
			return
		}
		if len(work.RunIDs) >= maxRunTrailers {
			omitted++
			return
		}
		work.RunIDs = append(work.RunIDs, run)
	}
	sources := append(append([]Proposal(nil), plan.replaced...), plan.folded...)
	for i := len(sources) - 1; i >= 0; i-- {
		for _, run := range sources[i].Work.RunIDs {
			addRun(run)
		}
		for _, plan := range sources[i].Work.Plans {
			if !contains(work.Plans, plan) {
				work.Plans = append(work.Plans, plan)
			}
		}
		for _, epoch := range sources[i].Work.FoldedEpochs {
			if epoch != work.Epoch && !contains(work.FoldedEpochs, epoch) {
				work.FoldedEpochs = append(work.FoldedEpochs, epoch)
			}
		}
	}
	sort.SliceStable(work.FoldedEpochs, func(i, j int) bool { return epochNumber(work.FoldedEpochs[i]) < epochNumber(work.FoldedEpochs[j]) })
	return work, omitted
}

func epochNumber(epoch string) int {
	var n int
	_, _ = fmt.Sscanf(strings.TrimPrefix(epoch, "E"), "%d", &n)
	return n
}

// foldedTrailers carries the non-generated Vrooli links of folded proposals
// (backlog items, generic work references) into the combined message.
func foldedTrailers(plan foldPlan) []trailers.Entry {
	var entries []trailers.Entry
	for _, proposal := range append(append([]Proposal(nil), plan.replaced...), plan.folded...) {
		for _, entry := range proposal.Message.Trailers {
			switch strings.ToLower(entry.Key) {
			case "vrooli-backlog", "vrooli-work":
				entries = append(entries, entry)
			}
		}
	}
	return entries
}

// foldSummary is the body paragraph that names every folded epoch, so the
// commit says what it carries.
func foldSummary(plan foldPlan, omittedRuns int) string {
	if len(plan.folds) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("Also carries earlier accepted epochs of this effort, folded from their open proposals:")
	for _, fold := range plan.folds {
		label := fold.Epoch
		if label == "" {
			label = "no epoch"
		}
		builder.WriteString(fmt.Sprintf("\n- %s (%s): %s", label, fold.ID, fold.Subject))
	}
	if omittedRuns > 0 {
		builder.WriteString(fmt.Sprintf("\n%d more runs are recorded on the folded proposals.", omittedRuns))
	}
	return builder.String()
}

// foldDetail describes the folds for the created event.
func foldDetail(plan foldPlan) string {
	var parts []string
	for _, proposal := range plan.folded {
		reason := FoldDrifted
		if shared := plan.overlaps[proposal.ID]; len(shared) > 0 {
			reason = fmt.Sprintf("%s: %s", FoldOverlap, strings.Join(shared, ", "))
		}
		parts = append(parts, fmt.Sprintf("%s (%s, %s)", proposal.ID, proposal.Work.Epoch, reason))
	}
	if len(parts) == 0 {
		return ""
	}
	return "folded " + strings.Join(parts, "; ")
}

// supersedeDetail is the history line on a folded or replaced proposal.
func supersedeDetail(proposal Proposal, plan foldPlan, newID, newEpoch string) string {
	if shared := plan.overlaps[proposal.ID]; len(shared) > 0 {
		return fmt.Sprintf("folded into %s (%s): shares %s", newID, newEpoch, strings.Join(shared, ", "))
	}
	for _, folded := range plan.folded {
		if folded.ID == proposal.ID {
			return fmt.Sprintf("folded into %s (%s): content drifted; carried forward at current content", newID, newEpoch)
		}
	}
	return "replaced by " + newID
}

// withoutPaths drops exclusions for paths the proposal now holds, such as an
// unlisted anchor-delta path that a folded proposal carries, and sorts the
// rest by path.
func withoutPaths(excluded []Exclusion, files []File) []Exclusion {
	held := map[string]struct{}{}
	for _, file := range files {
		held[file.Path] = struct{}{}
	}
	kept := excluded[:0]
	for _, exclusion := range excluded {
		if _, ok := held[exclusion.Path]; !ok {
			kept = append(kept, exclusion)
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].Path < kept[j].Path })
	return kept
}
