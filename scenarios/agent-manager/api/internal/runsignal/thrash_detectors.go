package runsignal

import (
	"fmt"
	"sort"
	"strings"

	"agent-manager/internal/domain"
)

// detectOscillations finds A...B...A...B cycles within a fixed recent window.
// The bounded backward scan is O(window*n), hence linear because window is a
// constant. A and B are fingerprints rather than command text.
func detectOscillations(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for end := 3; end < len(ctx.Facts); end++ {
		b := ctx.Facts[end].Fingerprint
		if !oscillatingFact(ctx.Facts[end]) {
			continue
		}
		limit := end - detectorWindow
		if limit < 0 {
			limit = 0
		}
		found := false
		for secondA := end - 1; secondA >= limit && !found; secondA-- {
			a := ctx.Facts[secondA].Fingerprint
			if !oscillatingFact(ctx.Facts[secondA]) || a == b {
				continue
			}
			for firstB := secondA - 1; firstB >= limit; firstB-- {
				if !oscillatingFact(ctx.Facts[firstB]) || ctx.Facts[firstB].Fingerprint != b {
					continue
				}
				for firstA := firstB - 1; firstA >= limit; firstA-- {
					if !oscillatingFact(ctx.Facts[firstA]) || ctx.Facts[firstA].Fingerprint != a {
						continue
					}
					e := newEpisode("oscillation", ctx.Facts[firstA], ctx.Facts[end], ctx.EventsByID, ctx.Events)
					e.price(ctx.Facts[firstA:end+1], ctx.EventsByID)
					e.CycleCount, e.RepeatedElement = 1, pairElement(a, b)
					out = append(out, e)
					found = true
					break
				}
			}
		}
	}
	return out
}

// oscillatingFact is a resolved command that is not a read-only status or
// listing, which is repeated by design.
func oscillatingFact(fact InvocationFact) bool {
	return comparableFingerprintFact(fact) && fact.Ownership == OwnershipResolved && !isPollingCommand(fact.CommandPath)
}

// detectEditReverts recognizes a file-change event whose path returns to a
// prior operation (add/delete or modify/revert) within the same bounded window.
// Codex rewrites a file as a delete followed by an add, so a path that was
// rewritten that way is never reverted by it.
func detectEditReverts(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	rewritten := rewrittenPaths(ctx.Events)
	for end, event := range ctx.Events {
		if prior, ok := swappedEdit(ctx.Events, end); ok {
			fact := InvocationFact{CallEventID: ctx.Events[prior].ID.String(), ResultEventID: ctx.Events[prior].ID.String()}
			endFact := InvocationFact{CallEventID: event.ID.String(), ResultEventID: event.ID.String()}
			e := newEpisode("edit-revert", fact, endFact, ctx.EventsByID, ctx.Events)
			e.CycleCount, e.RepeatedElement = 1, replacement(event).path
			out = append(out, e)
			continue
		}
		paths := changedPaths(event)
		if len(paths) == 0 {
			continue
		}
		start := max(end-detectorWindow, 0)
		for prior := end - 1; prior >= start; prior-- {
			for path, kind := range paths {
				if priorKind, ok := changedPaths(ctx.Events[prior])[path]; ok && isRevert(priorKind, kind) && !rewritten[path] {
					fact := InvocationFact{CallEventID: ctx.Events[prior].ID.String(), ResultEventID: ctx.Events[prior].ID.String()}
					endFact := InvocationFact{CallEventID: event.ID.String(), ResultEventID: event.ID.String()}
					e := newEpisode("edit-revert", fact, endFact, ctx.EventsByID, ctx.Events)
					e.CycleCount, e.RepeatedElement = 1, path
					out = append(out, e)
					goto next
				}
			}
		}
	next:
	}
	return out
}

type textReplacement struct{ path, old, new string }

// replacement reads a string-replacing edit call (a path, the text it replaces
// and its replacement).
func replacement(event *domain.RunEvent) textReplacement {
	call := toolCallData(event)
	if call == nil {
		return textReplacement{}
	}
	path, _ := call.Input["file_path"].(string)
	oldText, _ := call.Input["old_string"].(string)
	newText, _ := call.Input["new_string"].(string)
	return textReplacement{path, oldText, newText}
}

// swappedEdit finds, within the window before event end, an edit of the same
// file whose replaced and replacement text are this edit's, swapped: this edit
// puts back what the earlier one took out.
func swappedEdit(events []*domain.RunEvent, end int) (int, bool) {
	now := replacement(events[end])
	if now.path == "" || now.old == now.new || now.old == "" {
		return 0, false
	}
	for prior := end - 1; prior >= max(end-detectorWindow, 0); prior-- {
		if before := replacement(events[prior]); before.path == now.path && before.old == now.new && before.new == now.old {
			return prior, true
		}
	}
	return 0, false
}

// rewrittenPaths are the paths deleted and then added again with no message
// between: a rewrite, not an edit and its reversal.
func rewrittenPaths(events []*domain.RunEvent) map[string]bool {
	out := map[string]bool{}
	deleted := map[string]bool{}
	for _, event := range events {
		if _, ok := event.Data.(*domain.MessageEventData); ok {
			clear(deleted)
			continue
		}
		for path, kind := range changedPaths(event) {
			switch kind {
			case "delete":
				deleted[path] = true
			case "add":
				if deleted[path] {
					out[path] = true
				}
			}
		}
	}
	return out
}

func pairElement(a, b string) string {
	values := []string{a, b}
	sort.Strings(values)
	return fmt.Sprintf("%s:%s", values[0], values[1])
}

func changedPaths(event *domain.RunEvent) map[string]string {
	if event == nil {
		return nil
	}
	call, ok := event.Data.(*domain.ToolCallEventData)
	if !ok || call.ToolName != "file_change" {
		return nil
	}
	out := map[string]string{}
	switch files := call.Input["files"].(type) {
	case []map[string]string:
		for _, file := range files {
			if path := strings.TrimSpace(file["path"]); path != "" {
				out[path] = strings.ToLower(file["kind"])
			}
		}
	case []any:
		for _, raw := range files {
			file, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			path, _ := file["path"].(string)
			kind, _ := file["kind"].(string)
			if path = strings.TrimSpace(path); path != "" {
				out[path] = strings.ToLower(kind)
			}
		}
	}
	return out
}

func isRevert(previous, current string) bool {
	return (previous == "add" && current == "delete") || (previous == "delete" && current == "add") || (previous == "modify" && current == "revert") || (previous == "revert" && current == "modify")
}
