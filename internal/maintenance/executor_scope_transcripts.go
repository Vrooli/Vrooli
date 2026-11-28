package maintenance

import (
	"context"
	"fmt"
	"slices"
)

// Open-transcript evidence answers one narrow question for the caller: which
// of the harness transcript files it already knows (by exact path) are held
// open by a process of an executor this observation found present. A coding
// agent keeps its transcript open for the life of the conversation, so this is
// positive physical evidence that a transcript-only (imported) run is live
// inside an observed executor. Only requested paths are ever reported; no
// other descriptor, path, or process detail leaves the host read. An
// unreadable descriptor table yields no claim, never a negative.
const (
	// ExecutorScopeMaxTranscripts bounds the requested exact paths.
	ExecutorScopeMaxTranscripts = 4096
	// executorScopeTranscriptProcessBudget bounds descriptor-table reads per
	// request across all present executors and their descendants.
	executorScopeTranscriptProcessBudget = 16384
)

// scopeOpenFiles returns the resolved targets of a process's open file
// descriptors. Substituted only by isolated tests.
var scopeOpenFiles = readProcessOpenFiles

func normalizeExecutorTranscripts(paths []string) ([]string, error) {
	if len(paths) > ExecutorScopeMaxTranscripts {
		return nil, fmt.Errorf("executor transcript path limit exceeded")
	}
	normalized := make([]string, 0, len(paths))
	for _, candidate := range paths {
		value, err := normalizeExecutorHomePath(candidate)
		if err != nil {
			return nil, fmt.Errorf("executor transcript path must be a bounded absolute path")
		}
		if !slices.Contains(normalized, value) {
			normalized = append(normalized, value)
		}
	}
	slices.Sort(normalized)
	return normalized, nil
}

// attachOpenTranscripts sets OpenTranscripts on each present executor whose
// processes (sampled PIDs and their descendants) hold a requested path open.
func attachOpenTranscripts(ctx context.Context, transcripts []string, executors []ExecutorScopeEvidence, entries map[int]processTableEntry) {
	if len(transcripts) == 0 {
		return
	}
	wanted := make(map[string]struct{}, len(transcripts))
	for _, transcript := range transcripts {
		wanted[transcript] = struct{}{}
	}
	children := make(map[int][]int)
	for pid, entry := range entries {
		if entry.PPID > 0 && entry.PPID != pid {
			children[entry.PPID] = append(children[entry.PPID], pid)
		}
	}
	held := map[int][]string{}
	budget := executorScopeTranscriptProcessBudget
	for i := range executors {
		evidence := &executors[i]
		if evidence.State != "present" {
			continue
		}
		visited := map[int]bool{}
		queue := append([]int(nil), evidence.PIDs...)
		found := []string{}
		for len(queue) > 0 && ctx.Err() == nil {
			pid := queue[0]
			queue = queue[1:]
			if pid <= 1 || visited[pid] {
				continue
			}
			visited[pid] = true
			queue = append(queue, children[pid]...)
			paths, cached := held[pid]
			if !cached {
				if budget <= 0 {
					continue
				}
				budget--
				targets, err := scopeOpenFiles(pid)
				if err == nil {
					for _, target := range targets {
						if _, ok := wanted[target]; ok && !slices.Contains(paths, target) {
							paths = append(paths, target)
						}
					}
				}
				held[pid] = paths
			}
			for _, path := range paths {
				if !slices.Contains(found, path) {
					found = append(found, path)
				}
			}
		}
		if len(found) > 0 {
			slices.Sort(found)
			evidence.OpenTranscripts = found
		}
	}
}
