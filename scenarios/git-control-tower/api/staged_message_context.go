package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	advisoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	stagedMessageContextBytes = 12000
	stagedMessageContextLimit = 20
)

var stagedMessageIdentifier = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]{11,}`)

type stagedMessageClue struct {
	query      string
	path       string
	start, end time.Time
	reason     string
}

// Search collects policy-visible retained evidence only. It performs no
// inference, workflow starts, transcript imports, index repairs, or mutations.
func (p *stagedMessageAgent) Search(ctx context.Context, snapshot *advisoryv1.StagedMessageSnapshot) ([]*advisoryv1.StagedMessageContext, []*advisoryv1.Omission, error) {
	if snapshot == nil || snapshot.GetDigest() == "" {
		return nil, nil, fmt.Errorf("staged snapshot is required")
	}
	callerCtx := ctx
	budget := p.searchBudget
	if budget <= 0 || budget > 5*time.Second {
		budget = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	base, err := p.resolve(ctx)
	if err != nil {
		if callerCtx.Err() != nil {
			return nil, nil, callerCtx.Err()
		}
		return nil, []*advisoryv1.Omission{{Reason: "Agent Manager context unavailable; no external sessions searched"}}, nil
	}
	client := p.search(base)
	clues, omissions := stagedMessageClues(snapshot)
	seen := map[string]bool{}
	var contexts []*advisoryv1.StagedMessageContext
	for _, clue := range clues {
		if err := ctx.Err(); err != nil {
			if callerCtx.Err() != nil {
				return nil, nil, callerCtx.Err()
			}
			omissions = append(omissions, &advisoryv1.Omission{Reason: "Session context search wall-time budget exhausted; retained matches are partial"})
			break
		}
		filters := &ampb.ConversationSearchFilters{IncludeToolEvents: true}
		order := ampb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_RELEVANCE
		if !clue.start.IsZero() {
			filters.OccurredAfter = timestamppb.New(clue.start)
			filters.OccurredBefore = timestamppb.New(clue.end)
			order = ampb.ConversationSearchSort_CONVERSATION_SEARCH_SORT_NEWEST
		}
		response, err := client.SearchConversations(ctx, stagedMessageRequest(ctx, &ampb.SearchConversationsRequest{Query: clue.query, Mode: ampb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_TEXT, Sort: order, Filters: filters, PageSize: 10}))
		if err != nil {
			if callerCtx.Err() != nil {
				return nil, nil, callerCtx.Err()
			}
			if ctx.Err() != nil {
				omissions = append(omissions, &advisoryv1.Omission{Reason: "Session context search wall-time budget exhausted; retained matches are partial"})
				break
			}
			omissions = append(omissions, &advisoryv1.Omission{Reason: "Agent Manager context search unavailable (" + connect.CodeOf(err).String() + "); retained matches are incomplete"})
			break // An owner denial/outage never triggers a different credential or repeated search.
		}
		if response.Msg.GetNextPageCursor() != "" {
			omissions = append(omissions, &advisoryv1.Omission{Path: clue.path, Reason: "Additional matching session events omitted by bounded retrieval"})
		}
		for _, degradation := range response.Msg.GetDegradations() {
			omissions = append(omissions, &advisoryv1.Omission{Reason: "Agent Manager reported partial context: " + degradation.GetReason().String()})
		}
		for _, hit := range response.Msg.GetHits() {
			if hit == nil || hit.GetRunId() == "" || hit.GetEventId() == "" {
				continue
			}
			key := hit.GetRunId() + "/" + hit.GetEventId()
			if seen[key] {
				continue
			}
			seen[key] = true
			snippet := stagedMessageBoundedText(hit.GetSnippet(), 1200)
			if snippet == "" {
				continue
			}
			provenance := hit.GetProvenance()
			unknowns := []string{"Possible relevance only; authorship and exact edit coverage are unverified", "Event timestamp origin is unavailable: native versus synthesized/import time is unknown"}
			scope := "Repository scope unknown"
			if provenance.GetProjectScope() != "" {
				scope = "Recorded project scope: " + stagedMessageBoundedText(provenance.GetProjectScope(), 512)
			}
			if provenance.GetCwdScope() != "" {
				scope += "; recorded cwd: " + stagedMessageBoundedText(provenance.GetCwdScope(), 512)
			}
			if provenance.GetImporter() != "" {
				unknowns = append(unknowns, "Imported transcript tail/resumption freshness is unknown")
			}
			unknowns = append(unknowns, "Context freshness beyond the visible search projection is unknown")
			if coverage := response.Msg.GetCoverage(); coverage != nil && (coverage.GetPendingDocuments() > 0 || coverage.GetFreshnessLagMs() > 0) {
				unknowns = append(unknowns, "Search projection has pending or lagging content")
			}
			var matched []string
			for _, file := range snapshot.GetFiles() {
				if strings.Contains(snippet, file.GetPath()) || file.GetOldPath() != "" && strings.Contains(snippet, file.GetOldPath()) {
					matched = append(matched, file.GetPath())
				}
			}
			if len(matched) == 0 && clue.path != "" {
				matched = []string{clue.path}
				unknowns = append(unknowns, "Path is a retrieval clue; excerpt does not establish matching file content")
			}
			activity := ""
			if stamp := hit.GetOccurredAt(); stamp != nil && stamp.IsValid() {
				activity = stamp.AsTime().UTC().Format(time.RFC3339Nano)
			}
			title := stagedMessageBoundedText(hit.GetRun().GetLabel(), 180)
			if title == "" {
				title = "Related session event"
			}
			revisionBytes := []byte(key + "\x00" + hit.GetChunkId() + "\x00" + activity + "\x00" + scope + "\x00" + snippet)
			digest := sha256.Sum256(revisionBytes)
			revision := hex.EncodeToString(digest[:])
			detail := clue.reason + "; " + scope
			if session := provenance.GetSourceSessionId(); session != "" {
				detail += "; session: " + stagedMessageBoundedText(session, 128)
			}
			contexts = append(contexts, &advisoryv1.StagedMessageContext{Id: "am:" + revision[:24], Kind: "session_event", Reference: key, Revision: revision, Title: title, Body: snippet, Status: "possibly_related", Detail: detail, MatchedPaths: matched, Unknowns: unknowns, ActivityStart: activity, ActivityEnd: activity})
		}
	}
	// Query priority is changed identifiers/patch text, paths, then activity.
	// Search every bounded leg before trimming output, so a path-heavy result
	// cannot prevent modification-time discovery from running at all.
	remaining := stagedMessageContextBytes
	retained := make([]*advisoryv1.StagedMessageContext, 0, min(len(contexts), stagedMessageContextLimit))
	for _, candidate := range contexts {
		if len(retained) >= stagedMessageContextLimit || remaining <= 0 {
			omissions = append(omissions, &advisoryv1.Omission{Reason: "Session context candidate or excerpt byte limit reached"})
			break
		}
		if len(candidate.Body) > remaining {
			omissions = append(omissions, &advisoryv1.Omission{Reason: "Additional excerpt omitted by aggregate context byte bound"})
			break
		}
		remaining -= len(candidate.Body)
		retained = append(retained, candidate)
	}
	return retained, omissions, nil
}

func stagedMessageClues(snapshot *advisoryv1.StagedMessageSnapshot) ([]stagedMessageClue, []*advisoryv1.Omission) {
	var clues, pathClues []stagedMessageClue
	var omissions []*advisoryv1.Omission
	seenPaths := map[string]bool{}
	symbols := map[string]string{}
	fragments := map[string]string{}
	var windows []stagedMessageClue
	for _, file := range snapshot.GetFiles() {
		path := file.GetPath()
		if path != "" && !seenPaths[path] {
			seenPaths[path] = true
			if len(pathClues) < 2 {
				pathClues = append(pathClues, stagedMessageClue{query: `"` + strings.ReplaceAll(stagedMessageBoundedText(path, 512), `"`, ``) + `"`, path: path, reason: "Path overlap"})
			} else {
				omissions = append(omissions, &advisoryv1.Omission{Path: path, Reason: "Path-specific session query omitted by retrieval budget; other clues may still find related events"})
			}
		}
		for _, line := range strings.Split(file.GetPatch(), "\n") {
			if !(strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-")) || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
				continue
			}
			for _, symbol := range stagedMessageIdentifier.FindAllString(line, -1) {
				if len(symbol) <= 128 {
					symbols[symbol] = path
				}
			}
			fragment := strings.TrimSpace(line[1:])
			if len(fragment) >= 24 && len(fragment) <= 160 && len(strings.Fields(fragment)) >= 3 {
				fragments[fragment] = path
			}
		}
		for _, hint := range file.GetTimingHints() {
			stamp, err := time.Parse(time.RFC3339Nano, hint.GetTimestamp())
			if err != nil || stamp.IsZero() {
				continue
			}
			reason := "Retained event near file modification hint (" + stagedMessageBoundedText(hint.GetSource(), 80) + "); timing does not establish authorship"
			if !hint.GetContentMatchesIndex() {
				reason += "; observed file version differs from the staged version, so its time is only a weak retrieval clue"
			}
			windows = append(windows, stagedMessageClue{path: path, start: stamp.Add(-2 * time.Minute), end: stamp.Add(2 * time.Minute), reason: reason})
		}
	}
	identifiers := make([]string, 0, len(symbols))
	for symbol := range symbols {
		identifiers = append(identifiers, symbol)
	}
	sort.Slice(identifiers, func(i, j int) bool {
		if len(identifiers[i]) != len(identifiers[j]) {
			return len(identifiers[i]) > len(identifiers[j])
		}
		return identifiers[i] < identifiers[j]
	})
	identifierLimit := 2
	if len(fragments) > 0 {
		identifierLimit = 1
	}
	for _, symbol := range identifiers[:min(identifierLimit, len(identifiers))] {
		clues = append(clues, stagedMessageClue{query: symbol, path: symbols[symbol], reason: "Distinctive changed identifier; rarity is uncalibrated"})
	}
	if len(fragments) > 0 {
		values := make([]string, 0, len(fragments))
		for fragment := range fragments {
			values = append(values, fragment)
		}
		sort.Slice(values, func(i, j int) bool {
			if len(values[i]) != len(values[j]) {
				return len(values[i]) > len(values[j])
			}
			return values[i] < values[j]
		})
		fragment := values[0]
		clues = append(clues, stagedMessageClue{query: `"` + strings.ReplaceAll(fragment, `"`, ``) + `"`, path: fragments[fragment], reason: "Changed patch fragment; matching quotes or attempts do not prove authorship"})
	}
	clues = append(clues, pathClues...)
	sort.Slice(windows, func(i, j int) bool { return windows[i].start.Before(windows[j].start) })
	var merged []stagedMessageClue
	for _, window := range windows {
		if len(merged) > 0 && !window.start.After(merged[len(merged)-1].end) {
			last := &merged[len(merged)-1]
			if window.end.After(last.end) {
				last.end = window.end
			}
			last.path = ""
			last.reason = "Retained event near overlapping file modification hints; timing does not establish authorship"
		} else {
			merged = append(merged, window)
		}
	}
	for _, window := range merged[:min(2, len(merged))] {
		// Avoid a burst spanning hours becoming an effectively unbounded browse.
		if window.end.Sub(window.start) > 10*time.Minute {
			window.end = window.start.Add(10 * time.Minute)
			omissions = append(omissions, &advisoryv1.Omission{Reason: "Merged modification-time activity window bounded to ten minutes"})
		}
		clues = append(clues, window)
	}
	if len(merged) > 2 {
		omissions = append(omissions, &advisoryv1.Omission{Reason: "Additional file-modification activity windows omitted by retrieval budget"})
	}
	return clues, omissions
}

func stagedMessageBoundedText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	if limit <= 0 {
		return ""
	}
	text = text[:limit]
	for !utf8.ValidString(text) {
		text = text[:len(text)-1]
	}
	return text
}
