package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

func (c *Controller) ExecutorScope(parent context.Context, req ExecutorScopeRequest) (ExecutorScopeReport, error) {
	report := ExecutorScopeReport{SchemaVersion: "executor-scope-v1", IdentityVersion: ExecutorScopeIdentityVersion, Complete: true, Executors: []ExecutorScopeEvidence{}}
	if len(req.Executors) < 1 {
		return report, fmt.Errorf("executor-scope requires at least one executor")
	}
	for i := range req.Executors {
		homes, err := normalizeExecutorHomePaths(req.Executors[i].HomePaths)
		if err != nil {
			return report, fmt.Errorf("executor-scope invalid exact home reference: %w", err)
		}
		req.Executors[i].HomePaths = homes
	}
	transcripts, err := normalizeExecutorTranscripts(req.Transcripts)
	if err != nil {
		return report, fmt.Errorf("executor-scope invalid transcript reference: %w", err)
	}
	req.Transcripts = transcripts
	requestBytes, err := json.Marshal(req)
	if err != nil {
		return report, fmt.Errorf("encode executor-scope request: %w", err)
	}
	if len(requestBytes) > ExecutorScopeMaxBytes {
		return report, fmt.Errorf("executor-scope request exceeds %d bytes", ExecutorScopeMaxBytes)
	}
	ctx, cancel := context.WithTimeout(parent, executorScopeObservationTimeout)
	defer cancel()
	report.Executors = make([]ExecutorScopeEvidence, 0, len(req.Executors))
	seen := map[string]bool{}
	for _, ref := range req.Executors {
		if strings.TrimSpace(ref.RunID) == "" || ref.Tag == "" || ref.LegacyTag == "" || ref.PID < 0 || ref.PGID < 0 ||
			len(ref.RunID) > 128 || len(ref.Tag) > 256 || len(ref.LegacyTag) > 256 || seen[ref.RunID] {
			return report, fmt.Errorf("executor-scope requires unique runId, current and legacy tags, and nonnegative pid/pgid")
		}
		if ref.HarnessSessionID != "" || ref.HarnessKind != "" {
			if strings.TrimSpace(ref.HarnessSessionID) == "" || strings.TrimSpace(ref.HarnessKind) == "" ||
				strings.TrimSpace(ref.HarnessSessionID) != ref.HarnessSessionID || strings.TrimSpace(ref.HarnessKind) != ref.HarnessKind ||
				len(ref.HarnessSessionID) > 512 || len(ref.HarnessKind) > 100 || (ref.PID <= 1 && !validExecutorScopeBindingProof(ref)) {
				return report, fmt.Errorf("executor-scope attached identity requires an exact harnessSessionId, harnessKind, and persisted or owner-proved pid")
			}
		}
		if ref.BindingProof != nil && !validExecutorScopeBindingProof(ref) {
			return report, fmt.Errorf("executor-scope attached owner proof does not match its exact run and session identity")
		}
		seen[ref.RunID] = true
		if ref.StartedAt != nil && ref.EndedAt != nil && ref.EndedAt.Before(*ref.StartedAt) {
			return report, fmt.Errorf("executor-scope execution interval is reversed")
		}
		report.Executors = append(report.Executors, ExecutorScopeEvidence{RunID: ref.RunID, State: "absent", PIDs: []int{}, Reasons: []string{}})
	}
	// Read exact lease keys before walking the host process table. On a busy
	// host, process identity inspection can consume the whole observation
	// budget; leaving the lease read until afterward turns that unrelated cost
	// into a misleading "registry unavailable" result for every run.
	leasePages := make([]scopeLeasePage, 0, (len(req.Executors)+ExecutorScopeBatchSize-1)/ExecutorScopeBatchSize)
	for pageStart := 0; pageStart < len(req.Executors); pageStart += ExecutorScopeBatchSize {
		pageEnd := min(pageStart+ExecutorScopeBatchSize, len(req.Executors))
		pageRefs := req.Executors[pageStart:pageEnd]
		leasePages = append(leasePages, loadScopeLeasePage(ctx, c.Home, pageRefs))
	}
	beforeBirths, beforeErr := scopeProcessBirths(ctx)
	entries, identities, identityErrors, err := observeScopeProcesses(ctx)
	afterBirths, afterErr := scopeProcessBirths(ctx)
	var inherited map[int]map[string]bool
	if beforeErr == nil && afterErr == nil && err == nil {
		// This is optional evidence for removing inherited matches, not a
		// prerequisite for the process scan. On lookup failure retain every
		// original match; unrelated absent references remain provably absent.
		// Bound optional work independently of the overall observation budget.
		proofCtx, cancelProof := context.WithTimeout(ctx, time.Second)
		var proofErr error
		inherited, proofErr = c.inheritedServiceTags(proofCtx, req.Executors, entries, identities, beforeBirths, afterBirths)
		cancelProof()
		if proofErr != nil {
			inherited = nil
		}
	}
	if ctx.Err() != nil {
		err = fmt.Errorf("scope observation deadline or cancellation")
	}
	// A different process incarnation across the observation is uncertainty,
	// not a reusable timestamp attached to an unrelated PID.
	bornOutside := func(pid int, ref ExecutorScopeRef) bool {
		before, beforeOK := beforeBirths[pid]
		after, afterOK := afterBirths[pid]
		if beforeErr != nil || afterErr != nil || !beforeOK || !afterOK || before.StartedAt.IsZero() || !before.StartedAt.Equal(after.StartedAt) {
			return false
		}
		const allowance = 2 * time.Second
		return (ref.StartedAt != nil && before.StartedAt.Before(ref.StartedAt.Add(-allowance))) ||
			(ref.EndedAt != nil && before.StartedAt.After(ref.EndedAt.Add(allowance)))
	}
	for pageIndex, pageStart := 0, 0; pageStart < len(req.Executors); pageIndex, pageStart = pageIndex+1, pageStart+ExecutorScopeBatchSize {
		pageEnd := min(pageStart+ExecutorScopeBatchSize, len(req.Executors))
		pageRefs := req.Executors[pageStart:pageEnd]
		attached, attachedUncertain, attachedReasons, attachedErr := c.attachedExecutorLeases(ctx, pageRefs, leasePages[pageIndex], entries, identities, identityErrors, beforeBirths, afterBirths, beforeErr, afterErr, err)
		sampleLimit := max(1, min(64, executorScopeSampleBudget/len(pageRefs)))
		index, indexErr := indexExecutorScopes(ctx, pageRefs, entries, identities, inherited, sampleLimit)
		pageErr := err
		if indexErr != nil {
			pageErr = fmt.Errorf("scope observation deadline or cancellation")
		}
		for pageIndex, ref := range pageRefs {
			evidence := &report.Executors[pageStart+pageIndex]
			matches := [][]int{index.runs[ref.RunID], index.tags[ref.Tag], index.tags[ref.LegacyTag]}
			for _, home := range ref.HomePaths {
				matches = append(matches, index.homes[home])
			}
			// Only ended executions can use service-label exclusion. A live or
			// undated run retains all labels until its owner supplies an interval.
			if ref.EndedAt == nil {
				matches = append(matches, index.serviceTags[ref.Tag], index.serviceTags[ref.LegacyTag])
			}
			for _, pids := range matches {
				for _, pid := range pids {
					if len(evidence.PIDs) < sampleLimit && !slices.Contains(evidence.PIDs, pid) {
						evidence.PIDs = append(evidence.PIDs, pid)
					}
				}
			}
			if len(evidence.PIDs) > 0 {
				evidence.State = "present"
			} else if pid := attached[ref.RunID]; pid > 0 {
				evidence.State = "present"
				evidence.PIDs = append(evidence.PIDs, pid)
			} else if attachedUncertain[ref.RunID] {
				evidence.State = "unknown"
				evidence.Reasons = append(evidence.Reasons, "attached-session binding lacks current process/scope proof")
			} else if attachedErr != nil {
				evidence.State = "unknown"
				evidence.Reasons = append(evidence.Reasons, attachedErr.Error())
			} else {
				ambiguous := 0
				if entry, exists := entries[ref.PID]; exists && ref.PID > 0 && !strings.HasPrefix(entry.State, "Z") && !bornOutside(ref.PID, ref) {
					ambiguous = ref.PID
				}
				// A later-born member does not prove the old group was replaced.
				if member := index.groups[ref.PGID]; member > 0 && !bornOutside(ref.PGID, ref) {
					ambiguous = member
				}
				if ambiguous > 0 {
					evidence.State = "unknown"
					evidence.Reasons = append(evidence.Reasons, fmt.Sprintf("pid/group %d lacks run identity or reuse proof", ambiguous))
				}
			}
			if len(ref.HomePaths) > 0 {
				unreadable := make([]int, 0)
				for pid := range identityErrors {
					entry, ok := entries[pid]
					if ok && executorHomeIdentityInScope(entry) {
						unreadable = append(unreadable, pid)
					}
				}
				if len(unreadable) > 0 {
					slices.Sort(unreadable)
					const unreadablePIDSampleLimit = 16
					sample := unreadable[:min(len(unreadable), unreadablePIDSampleLimit)]
					report.Complete = false
					if evidence.State != "present" {
						evidence.State = "unknown"
					}
					evidence.Reasons = append(evidence.Reasons, fmt.Sprintf("exact-home inventory has %d unreadable in-scope process identity reads; sample_pids=%v; omitted_pids=%d", len(unreadable), sample, max(0, len(unreadable)-len(sample))))
				}
			}
			if reason := attachedReasons[ref.RunID]; reason != "" {
				evidence.Reasons = append(evidence.Reasons, reason)
			}
			slices.Sort(evidence.PIDs)
			if pageErr != nil {
				report.Complete = false
				if evidence.State != "present" {
					evidence.State = "unknown"
				}
				evidence.Reasons = append(evidence.Reasons, pageErr.Error())
			}
			if evidence.State == "unknown" {
				report.Complete = false
			}
		}
	}
	// Positive-only evidence; it never changes a verdict or completeness.
	attachOpenTranscripts(ctx, req.Transcripts, report.Executors, entries)
	// The deadline covers indexing and matching too. Never emit a previously
	// complete-looking absence after that observation budget expired.
	if ctx.Err() != nil {
		report.Complete = false
		for i := range report.Executors {
			if report.Executors[i].State == "absent" {
				report.Executors[i].State = "unknown"
				report.Executors[i].Reasons = []string{"scope observation deadline or cancellation"}
			}
		}
	}
	return report, nil
}
