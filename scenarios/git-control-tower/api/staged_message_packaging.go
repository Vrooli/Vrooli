package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"git-control-tower/internal/stagedmessages"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

const stagedMessageDetailLimit = 64

type stagedMessageCount struct {
	Label string `json:"label"`
	Files int    `json:"files"`
}

type stagedMessageDetail struct {
	ID            int    `json:"file_id"`
	Path          string `json:"path"`
	OldPath       string `json:"old_path,omitempty"`
	Patch         string `json:"patch"`
	CompletePatch bool   `json:"complete_patch"`
}

type stagedMessageSource struct {
	ID                  string   `json:"id"`
	Revision            string   `json:"revision"`
	Kind                string   `json:"kind"`
	Status              string   `json:"status"`
	Title               string   `json:"title"`
	Excerpt             string   `json:"excerpt"`
	ExcerptTruncated    bool     `json:"excerpt_truncated"`
	MatchedPathExamples int      `json:"matched_path_examples"`
	MatchedEdits        int32    `json:"matched_edits"`
	FullMatch           bool     `json:"full_match"`
	Unknowns            []string `json:"unknowns"`
}

type stagedMessageDraftContext struct {
	ContractVersion      int                   `json:"contract_version,omitempty"`
	EvidenceJobID        string                `json:"evidence_job_id,omitempty"`
	PreviousTitle        string                `json:"previous_title,omitempty"`
	PreviousBody         string                `json:"previous_body,omitempty"`
	PreviouslyCoveredIDs []int                 `json:"previously_covered_ids,omitempty"`
	SnapshotDigest       string                `json:"snapshot_digest"`
	RepositoryID         string                `json:"repository_id"`
	FileCount            int                   `json:"file_count"`
	ScopeCounts          []stagedMessageCount  `json:"scope_counts"`
	OmissionCounts       []stagedMessageCount  `json:"omission_counts"`
	Files                []stagedMessageDetail `json:"files"`
	Contexts             []stagedMessageSource `json:"contexts"`
	OperatorNote         string                `json:"operator_note"`
	Limitations          []string              `json:"limitations"`
}

type stagedMessageCompactInput struct {
	FormatVersion       int                       `json:"format_version"`
	DraftInputDigest    string                    `json:"draft_input_digest"`
	RepositoryDirectory string                    `json:"repository_directory"`
	DraftContext        stagedMessageDraftContext `json:"draft_context"`
}

func stagedMessageInputDigest(in stagedmessages.DraftInput) (string, error) {
	if in.Contexts == nil {
		in.Contexts = []*v1.StagedMessageContext{}
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Counts represent the entire immutable selection, including files whose paths
// and patches cannot fit in the model context. The last row retains the exact
// count of groups beyond the bounded display, rather than dropping their files.
func stagedMessageCounts(counts map[string]int, limit int) []stagedMessageCount {
	labels := make([]string, 0, len(counts))
	for label := range counts {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	rows := make([]stagedMessageCount, 0, limit+1)
	remaining := 0
	for i, label := range labels {
		if i < limit {
			rows = append(rows, stagedMessageCount{Label: stagedMessageBoundedText(label, 160), Files: counts[label]})
		} else {
			remaining += counts[label]
		}
	}
	if remaining > 0 {
		rows = append(rows, stagedMessageCount{Label: "additional groups (summarized)", Files: remaining})
	}
	return rows
}

func stagedMessageScope(path string) string {
	parts := strings.Split(path, "/")
	return strings.Join(parts[:min(2, len(parts))], "/")
}

// Prefer cheaper complete patches across scope groups, then make another pass
// through those groups. A large lexical edge file cannot crowd out all other
// details or force the selection to become empty.
func stagedMessageDetailOrder(candidates []stagedMessageDetail) []stagedMessageDetail {
	groups := map[string][]stagedMessageDetail{}
	costs := map[int]int{}
	for _, detail := range candidates {
		raw, _ := json.Marshal(detail)
		costs[detail.ID] = len(raw)
		group := stagedMessageScope(detail.Path)
		groups[group] = append(groups[group], detail)
	}
	keys := make([]string, 0, len(groups))
	for key, group := range groups {
		sort.SliceStable(group, func(i, j int) bool { return costs[group[i].ID] < costs[group[j].ID] })
		groups[key] = group
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := costs[groups[keys[i]][0].ID], costs[groups[keys[j]][0].ID]
		if a != b {
			return a < b
		}
		return keys[i] < keys[j]
	})
	ordered := make([]stagedMessageDetail, 0, len(candidates))
	for row := 0; len(ordered) < len(candidates); row++ {
		for _, key := range keys {
			if row < len(groups[key]) {
				ordered = append(ordered, groups[key][row])
			}
		}
	}
	return ordered
}

func packageStagedMessageInput(in stagedmessages.DraftInput, directory string) (any, error) {
	if in.FormatVersion == 0 {
		// Preserve the exact original input shape and owner key on a lost legacy
		// acknowledgement. AM checks an existing idempotency key before the active
		// revision's input schema. A new v2 revision will truthfully reject a legacy
		// admission with no prior execution; it is never silently reinterpreted.
		if in.Contexts == nil {
			in.Contexts = []*v1.StagedMessageContext{}
		}
		raw, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		var legacy map[string]any
		if err := json.Unmarshal(raw, &legacy); err != nil {
			return nil, err
		}
		legacy["repository_directory"] = directory
		return legacy, nil
	}
	if (in.FormatVersion != 2 && in.FormatVersion != 3 && in.FormatVersion != 4) || in.Snapshot == nil || len(in.Snapshot.Files) > stagedmessages.MaxManifestFiles {
		return nil, errors.New("unsupported immutable draft format or file count")
	}
	digest, err := stagedMessageInputDigest(in)
	if err != nil {
		return nil, err
	}
	input := stagedMessageCompactInput{FormatVersion: in.FormatVersion, DraftInputDigest: digest, RepositoryDirectory: directory}
	context := &input.DraftContext
	if in.FormatVersion == 4 {
		context.ContractVersion = 4
	}
	context.SnapshotDigest, context.RepositoryID = in.Snapshot.Digest, in.Snapshot.RepositoryId
	context.FileCount, context.OperatorNote = len(in.Snapshot.Files), in.OperatorNote
	priorCovered := map[string]bool{}
	if in.FormatVersion >= 3 {
		if in.JobID == "" || in.PriorResult == nil || in.ContinuationOfJobID == "" {
			return nil, errors.New("deep drafting requires its durable parent and evidence job")
		}
		context.EvidenceJobID = in.JobID
		context.PreviousTitle = in.PriorResult.Title
		context.PreviousBody = in.PriorResult.Body
		for _, path := range in.PriorResult.GetCoverage().GetCoveredPaths() {
			priorCovered[path] = true
		}
	}
	context.Files, context.Contexts = []stagedMessageDetail{}, []stagedMessageSource{}
	scopes, omissions := map[string]int{}, map[string]int{}
	var candidates []stagedMessageDetail
	paths := map[string]bool{}
	for i, file := range in.Snapshot.Files {
		if file == nil || file.Path == "" || paths[file.Path] {
			return nil, errors.New("immutable file manifest contains an absent path")
		}
		paths[file.Path] = true
		if priorCovered[file.Path] {
			context.PreviouslyCoveredIDs = append(context.PreviouslyCoveredIDs, i+1)
		}
		scopes[stagedMessageScope(file.Path)]++
		reason := file.OmissionReason
		if file.Binary {
			reason = "binary content unavailable"
		} else if reason == "" && file.Patch == "" {
			reason = "no staged patch supplied"
		}
		if reason != "" {
			omissions[reason]++
		}
		if file.Patch != "" && in.FormatVersion < 3 {
			candidates = append(candidates, stagedMessageDetail{ID: i + 1, Path: file.Path, OldPath: file.OldPath, Patch: file.Patch, CompletePatch: reason == ""})
		}
	}
	context.ScopeCounts, context.OmissionCounts = stagedMessageCounts(scopes, 24), stagedMessageCounts(omissions, 12)
	for _, source := range in.Contexts {
		if source == nil || len(source.Id) > 256 || len(source.Revision) > 512 {
			return nil, errors.New("selected source identity exceeds supported bounds")
		}
		unknowns := []string{"Bounded source excerpt and scope counts; source relevance does not establish authorship or complete staged coverage."}
		for _, unknown := range source.Unknowns {
			if len(unknowns) == 4 {
				break
			}
			unknowns = append(unknowns, stagedMessageBoundedText(unknown, 160))
		}
		context.Contexts = append(context.Contexts, stagedMessageSource{ID: source.Id, Revision: source.Revision, Kind: stagedMessageBoundedText(source.Kind, 64), Status: stagedMessageBoundedText(source.Status, 96), Title: stagedMessageBoundedText(source.Title, 160), Excerpt: stagedMessageBoundedText(source.Body, 384), ExcerptTruncated: len(source.Body) > 384, MatchedPathExamples: len(source.MatchedPaths), MatchedEdits: source.MatchedEdits, FullMatch: source.FullMatch, Unknowns: unknowns})
	}
	candidates = stagedMessageDetailOrder(candidates)
	fit := func() ([]byte, bool) {
		context.Limitations = []string{
			fmt.Sprintf("The subject includes all %d staged files. Detailed representative patches are supplied for %d files; other files remain in scope but cannot be claimed fully covered.", context.FileCount, len(context.Files)),
			"Whole captured patches are preferred across scope groups within a byte budget. Any fallback excerpt is explicitly incomplete. Group counts summarize the complete selection, not proof of every edit.",
			"Only complete_patch=true file IDs may be claimed covered. All other and unspecified file IDs are expanded as uncovered by the server.",
		}
		if in.FormatVersion >= 3 {
			context.Limitations = []string{"This is deeper investigation of the SAME complete immutable staged selection. Read remaining whole patches through the scoped evidence CLI; do not repeat the representative sample.", "Only previously covered IDs and complete patches actually supplied by evidence pages may be covered. Unread/binary/oversized/unavailable changes remain uncovered; previous text must be reconciled with newly examined changes."}
		}
		if in.FormatVersion == 4 {
			context.Limitations = []string{"Message scope and detailed patch inspection are separate. Obtain the complete immutable metadata inventory, then choose important or representative stable file IDs for targeted patch reads; reading every patch is not required.", "Inventory edit_count is known only when positive; absent or zero means unmeasured, not no changes. Do not treat unsampled files as small.", "Report whole_selection only after receiving the complete inventory and explaining the synthesis basis. Use limited with substantive unknown themes or a clear no-progress explanation. Covered IDs still mean complete inspected patches, never metadata-only reads."}
		}
		prompt, _ := json.Marshal(context)
		packaged, _ := json.Marshal(input)
		return packaged, len(prompt) <= stagedMessageMaxPromptBytes && len(packaged) <= stagedMessageMaxWorkflowInputBytes
	}
	for trim := 0; trim < 3; trim++ {
		context.Files = context.Files[:0]
		if _, fits := fit(); fits {
			included := map[int]bool{}
			// A complete whole patch always gets a chance before an excerpt.
			for _, detail := range candidates {
				if !detail.CompletePatch || len(context.Files) == stagedMessageDetailLimit {
					continue
				}
				context.Files = append(context.Files, detail)
				if _, fits := fit(); !fits {
					context.Files = context.Files[:len(context.Files)-1]
				} else {
					included[detail.ID] = true
				}
			}
			for _, detail := range candidates {
				if included[detail.ID] || len(context.Files) == stagedMessageDetailLimit {
					continue
				}
				detail.Patch = stagedMessageBoundedText(detail.Patch, 2048)
				detail.CompletePatch = false
				context.Files = append(context.Files, detail)
				if _, fits := fit(); !fits {
					context.Files = context.Files[:len(context.Files)-1]
				}
			}
			packaged, _ := fit()
			var value any
			err = json.Unmarshal(packaged, &value)
			return value, err
		}
		// Preserve every selected ID/revision while reducing only explanatory
		// excerpts. Large selected-source path lists were already replaced by counts.
		for i := range context.Contexts {
			source := &context.Contexts[i]
			bound := 128
			if trim > 0 {
				bound = 0
			}
			if len(source.Excerpt) > bound {
				source.ExcerptTruncated = true
			}
			source.Excerpt = stagedMessageBoundedText(source.Excerpt, bound)
			source.Title = stagedMessageBoundedText(source.Title, 64)
			source.Unknowns = source.Unknowns[:1]
		}
		context.ScopeCounts, context.OmissionCounts = stagedMessageCounts(scopes, 8), stagedMessageCounts(omissions, 4)
	}
	return nil, errors.New("bounded source identities and operator note exceed the workflow envelope")
}

type stagedMessageCompactResult struct {
	MessageScope            string   `json:"message_scope,omitempty"`
	MessageScopeExplanation string   `json:"message_scope_explanation,omitempty"`
	SnapshotDigest          string   `json:"snapshot_digest"`
	Title                   string   `json:"title"`
	Body                    string   `json:"body"`
	Trailers                []any    `json:"trailers"`
	CoveredFileIDs          []int    `json:"covered_file_ids"`
	SourceIDs               []string `json:"source_ids"`
	Unknowns                []string `json:"unknowns"`
}

func decodeBoundStagedMessageResult(execution *ampb.WorkflowExecution, canonical stagedmessages.DraftInput) (*v1.StagedMessageSuggestion, error) {
	return decodeStagedMessageResultWithEvidence(execution, canonical, nil)
}

func decodeStagedMessageResultWithEvidence(execution *ampb.WorkflowExecution, canonical stagedmessages.DraftInput, evidence map[int]bool) (*v1.StagedMessageSuggestion, error) {
	return decodeStagedMessageSynthesisResult(execution, canonical, evidence, nil)
}
func decodeStagedMessageSynthesisResult(execution *ampb.WorkflowExecution, canonical stagedmessages.DraftInput, evidence, inventory map[int]bool) (*v1.StagedMessageSuggestion, error) {
	raw, err := json.Marshal(execution.GetInput().AsInterface())
	if err != nil {
		return nil, err
	}
	var input stagedMessageCompactInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, err
	}
	if input.FormatVersion == 0 {
		if canonical.FormatVersion != 0 || canonical.Snapshot == nil {
			return nil, errors.New("legacy workflow is not bound to this immutable draft")
		}
		var legacy map[string]any
		if err := json.Unmarshal(raw, &legacy); err != nil {
			return nil, err
		}
		delete(legacy, "repository_directory")
		var retained stagedmessages.DraftInput
		retainedRaw, _ := json.Marshal(legacy)
		if err := json.Unmarshal(retainedRaw, &retained); err != nil {
			return nil, err
		}
		expected, _ := stagedMessageInputDigest(canonical)
		actual, _ := stagedMessageInputDigest(retained)
		if expected != actual {
			return nil, errors.New("legacy workflow subject differs from the canonical draft")
		}
		return decodeStagedMessageResult(execution)
	}
	if (input.FormatVersion != 2 && input.FormatVersion != 3 && input.FormatVersion != 4) || canonical.FormatVersion != input.FormatVersion || canonical.Snapshot == nil || len(canonical.Snapshot.Files) == 0 || len(canonical.Snapshot.Files) > stagedmessages.MaxManifestFiles {
		return nil, errors.New("unsupported or mismatched workflow input format")
	}
	expected, err := stagedMessageInputDigest(canonical)
	if err != nil || input.DraftInputDigest != expected || input.DraftContext.SnapshotDigest != canonical.Snapshot.Digest || input.DraftContext.RepositoryID != canonical.Snapshot.RepositoryId || input.DraftContext.FileCount != len(canonical.Snapshot.Files) {
		return nil, errors.New("compact workflow subject differs from the canonical draft")
	}
	eligible := map[int]bool{}
	if canonical.FormatVersion >= 3 {
		if len(input.DraftContext.Files) != 0 || input.DraftContext.EvidenceJobID != canonical.JobID || canonical.PriorResult == nil || input.DraftContext.PreviousTitle != canonical.PriorResult.Title || input.DraftContext.PreviousBody != canonical.PriorResult.Body {
			return nil, errors.New("deep workflow is not bound to its durable prior draft")
		}
		prior := map[string]bool{}
		for _, path := range canonical.PriorResult.GetCoverage().GetCoveredPaths() {
			prior[path] = true
		}
		for i, file := range canonical.Snapshot.Files {
			if prior[file.Path] || evidence[i+1] {
				eligible[i+1] = true
			}
		}
		if len(input.DraftContext.PreviouslyCoveredIDs) != len(prior) {
			return nil, errors.New("deep prior coverage differs from durable draft")
		}
		seenPrior := map[int]bool{}
		for _, id := range input.DraftContext.PreviouslyCoveredIDs {
			if id < 1 || id > len(canonical.Snapshot.Files) || seenPrior[id] || !prior[canonical.Snapshot.Files[id-1].Path] {
				return nil, errors.New("deep prior coverage contains an unsupported file")
			}
			seenPrior[id] = true
		}
	}
	seen := map[int]bool{}
	paths := map[string]bool{}
	for _, file := range canonical.Snapshot.Files {
		if file == nil || file.Path == "" || paths[file.Path] {
			return nil, errors.New("canonical file manifest contains an absent or duplicate path")
		}
		paths[file.Path] = true
	}
	for _, detail := range input.DraftContext.Files {
		if detail.ID < 1 || detail.ID > len(canonical.Snapshot.Files) || seen[detail.ID] {
			return nil, errors.New("unknown or duplicated detailed file ID")
		}
		seen[detail.ID] = true
		file := canonical.Snapshot.Files[detail.ID-1]
		if detail.Path != file.Path || detail.OldPath != file.OldPath || !strings.HasPrefix(file.Patch, detail.Patch) || (detail.CompletePatch && (file.Patch == "" || detail.Patch != file.Patch || file.Binary || file.OmissionReason != "")) {
			return nil, errors.New("detailed file evidence differs from the immutable staged patch")
		}
		eligible[detail.ID] = detail.CompletePatch
	}
	sources := map[string]*v1.StagedMessageContext{}
	for _, source := range canonical.Contexts {
		if source == nil || source.Id == "" {
			return nil, errors.New("canonical selected source identity is absent")
		}
		sources[source.Id] = source
	}
	available := map[string]bool{}
	for _, source := range input.DraftContext.Contexts {
		canonical, ok := sources[source.ID]
		if !ok || canonical.Revision != source.Revision || available[source.ID] {
			return nil, errors.New("workflow source identity or revision differs from selected context")
		}
		if !strings.HasPrefix(canonical.Body, source.Excerpt) || !strings.HasPrefix(canonical.Title, source.Title) || !strings.HasPrefix(canonical.Kind, source.Kind) || !strings.HasPrefix(canonical.Status, source.Status) || source.MatchedEdits != canonical.MatchedEdits || source.FullMatch != canonical.FullMatch || source.MatchedPathExamples != len(canonical.MatchedPaths) || (!source.ExcerptTruncated && source.Excerpt != canonical.Body) {
			return nil, errors.New("workflow source excerpt or scope differs from selected context")
		}
		available[source.ID] = true
	}
	if len(available) != len(sources) {
		return nil, errors.New("workflow omitted a selected source identity")
	}
	output, ok := execution.GetOutput().AsInterface().(map[string]any)
	if !ok {
		return nil, errors.New("output unavailable")
	}
	resultRaw, err := json.Marshal(output["result"])
	if err != nil || len(resultRaw) > 60<<10 {
		return nil, errors.New("compact message output exceeds owner bounds")
	}
	var result stagedMessageCompactResult
	if err := decodeStrictStagedMessageJSON(resultRaw, &result); err != nil {
		return nil, err
	}
	coverageLimit := stagedMessageDetailLimit
	maxBody, maxUnknowns, maxUnknownBytes := 8000, 31, 1000
	if canonical.FormatVersion >= 3 {
		coverageLimit = stagedmessages.MaxManifestFiles
		maxBody, maxUnknowns, maxUnknownBytes = 6000, 16, 500
	}
	if result.SnapshotDigest != canonical.Snapshot.Digest || strings.TrimSpace(result.Title) == "" || len(result.Title) > 200 || strings.ContainsAny(result.Title, "\r\n") || len(result.Body) > maxBody || len(result.Trailers) != 0 || len(result.CoveredFileIDs) > coverageLimit || len(result.SourceIDs) > 20 || len(result.Unknowns) > maxUnknowns {
		return nil, errors.New("message fields exceed compact contract or subject binding")
	}
	scope := v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_UNSPECIFIED
	if canonical.FormatVersion == 4 {
		if input.DraftContext.ContractVersion != 4 || strings.TrimSpace(result.MessageScopeExplanation) == "" || len(result.MessageScopeExplanation) > 1000 {
			return nil, errors.New("synthesis requires its explicit contract and bounded scope explanation")
		}
		switch result.MessageScope {
		case "whole_selection":
			scope = v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_WHOLE_SELECTION
			for i := range canonical.Snapshot.Files {
				if !inventory[i+1] {
					return nil, errors.New("whole-selection synthesis requires the complete immutable inventory")
				}
			}
		case "limited":
			scope = v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_LIMITED
		default:
			return nil, errors.New("invalid synthesis message scope")
		}
	} else if result.MessageScope != "" || result.MessageScopeExplanation != "" {
		return nil, errors.New("legacy output cannot reinterpret message scope")
	}
	covered := map[int]bool{}
	for _, id := range result.CoveredFileIDs {
		if !eligible[id] || covered[id] {
			return nil, errors.New("unknown, duplicate or incompletely supplied covered file ID")
		}
		covered[id] = true
	}
	usedSources := map[string]bool{}
	for _, id := range result.SourceIDs {
		if !available[id] || usedSources[id] {
			return nil, errors.New("unknown or duplicate selected source")
		}
		usedSources[id] = true
	}
	for _, unknown := range result.Unknowns {
		if len(unknown) > maxUnknownBytes {
			return nil, errors.New("unknown disclosure exceeds bounds")
		}
	}
	coverage := &v1.StagedMessageCoverage{}
	for i, file := range canonical.Snapshot.Files {
		edits := max(file.EditCount, 1)
		coverage.TotalEdits += edits
		if covered[i+1] {
			coverage.CoveredPaths = append(coverage.CoveredPaths, file.Path)
			coverage.MatchedEdits += edits
		} else {
			coverage.UncoveredPaths = append(coverage.UncoveredPaths, file.Path)
		}
	}
	coverage.Partial = len(coverage.UncoveredPaths) > 0
	unknowns := []string{fmt.Sprintf("Drafting received representative file details for %d of %d staged files; %d files are not fully described. The immutable subject includes the entire selection.", len(input.DraftContext.Files), len(canonical.Snapshot.Files), len(coverage.UncoveredPaths))}
	if canonical.FormatVersion >= 3 {
		unknowns = []string{fmt.Sprintf("Deeper investigation supplied whole staged patches for %d files, in addition to previous verified draft coverage; %d of %d staged files remain not fully described.", len(evidence), len(coverage.UncoveredPaths), len(canonical.Snapshot.Files))}
	}
	if canonical.FormatVersion == 4 {
		unknowns = nil
	}
	unknowns = append(unknowns, result.Unknowns...)
	return &v1.StagedMessageSuggestion{MessageScope: scope, MessageScopeExplanation: result.MessageScopeExplanation, Id: execution.Id, Title: result.Title, Body: result.Body, Coverage: coverage, SourceIds: result.SourceIDs, Unknowns: unknowns}, nil
}
