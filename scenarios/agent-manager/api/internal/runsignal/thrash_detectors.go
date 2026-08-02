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
		if b == "" {
			continue
		}
		limit := end - detectorWindow
		if limit < 0 {
			limit = 0
		}
		for secondA := end - 1; secondA >= limit; secondA-- {
			a := ctx.Facts[secondA].Fingerprint
			if a == "" || a == b {
				continue
			}
			for firstB := secondA - 1; firstB >= limit; firstB-- {
				if ctx.Facts[firstB].Fingerprint != b {
					continue
				}
				for firstA := firstB - 1; firstA >= limit; firstA-- {
					if ctx.Facts[firstA].Fingerprint != a {
						continue
					}
					e := newEpisode("oscillation", ctx.Facts[firstA], ctx.Facts[end], ctx.EventsByID, ctx.Events)
					e.Turns, e.CycleCount, e.RepeatedElement = end-firstA+1, 1, pairElement(a, b)
					out = append(out, e)
					break
				}
				if len(out) > 0 && out[len(out)-1].EndEventID == ctx.Facts[end].ResultEventID {
					break
				}
				if len(out) > 0 && out[len(out)-1].EndEventID == ctx.Facts[end].ResultEventID {
					break
				}
			}
			if len(out) > 0 && out[len(out)-1].EndEventID == ctx.Facts[end].ResultEventID {
				break
			}
		}
	}
	return out
}

// detectEditReverts recognizes a file-change event whose path returns to a
// prior operation (add/delete or modify/revert) within the same bounded window.
func detectEditReverts(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for end, event := range ctx.Events {
		paths := changedPaths(event)
		if len(paths) == 0 {
			continue
		}
		start := end - detectorWindow
		if start < 0 {
			start = 0
		}
		for prior := end - 1; prior >= start; prior-- {
			for path, kind := range paths {
				if priorKind, ok := changedPaths(ctx.Events[prior])[path]; ok && isRevert(priorKind, kind) {
					fact := InvocationFact{CallEventID: ctx.Events[prior].ID.String(), ResultEventID: ctx.Events[prior].ID.String()}
					endFact := InvocationFact{CallEventID: event.ID.String(), ResultEventID: event.ID.String()}
					e := newEpisode("edit-revert", fact, endFact, ctx.EventsByID, ctx.Events)
					e.Turns, e.CycleCount, e.RepeatedElement = end-prior+1, 1, path
					out = append(out, e)
					goto next
				}
			}
		}
	next:
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
	files, ok := call.Input["files"].([]map[string]string)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for _, file := range files {
		if path := strings.TrimSpace(file["path"]); path != "" {
			out[path] = strings.ToLower(file["kind"])
		}
	}
	return out
}

func isRevert(previous, current string) bool {
	return (previous == "add" && current == "delete") || (previous == "delete" && current == "add") || (previous == "modify" && current == "revert") || (previous == "revert" && current == "modify")
}
