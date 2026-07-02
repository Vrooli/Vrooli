package authoring

import (
	"sort"
	"strings"

	"github.com/google/uuid"

	planmodel "plan-manager/internal/planmodel"
)

const (
	referenceTierShortlist = contextTierShortlist
	referenceTierLonglist  = contextTierLonglist

	referenceShortlistScore = contextSearchShortlistScore
	referenceHighConfidence = contextHighConfidenceScore
	referenceShortlistCap   = 8
)

func mergeReferenceDiscoveryBatch(sess *Session, candidates []ReferenceCandidate) DiscoveryBatch {
	batch := DiscoveryBatch{
		ID:         uuid.NewString(),
		Source:     "interactive",
		Status:     DiscoveryBatchPending,
		CreatedSeq: len(sess.ReferenceBatches) + 1,
	}
	for i := range sess.ReferenceBatches {
		if sess.ReferenceBatches[i].Status == DiscoveryBatchPending {
			sess.ReferenceBatches[i].Status = DiscoveryBatchSuperseded
		}
	}

	known := map[string]int{}
	for i := range sess.ReferenceCandidates {
		key := referenceDedupeKey(sess.ReferenceCandidates[i].Reference)
		if key != "" {
			known[key] = i
		}
	}

	for i := range sess.ReferenceCandidates {
		if sess.ReferenceCandidates[i].Status != ReferenceCandidatePending {
			continue
		}
		sess.ReferenceCandidates[i].BatchID = batch.ID
	}

	for _, incoming := range candidates {
		incoming = normalizeReferenceCandidate(incoming)
		incoming.BatchID = batch.ID
		key := referenceDedupeKey(incoming.Reference)
		if key == "" {
			continue
		}
		if pos, ok := known[key]; ok {
			switch sess.ReferenceCandidates[pos].Status {
			case ReferenceCandidatePending:
				merged := mergeReferenceCandidate(sess.ReferenceCandidates[pos], incoming)
				merged.ID = sess.ReferenceCandidates[pos].ID
				merged.Reference.ID = sess.ReferenceCandidates[pos].Reference.ID
				merged.Status = ReferenceCandidatePending
				merged.BatchID = batch.ID
				sess.ReferenceCandidates[pos] = merged
			default:
				batch.CurationStats.SuppressedDispositioned++
			}
			continue
		}
		known[key] = len(sess.ReferenceCandidates)
		sess.ReferenceCandidates = append(sess.ReferenceCandidates, incoming)
	}

	curatePendingReferenceBatch(sess, batch.ID, &batch)
	assignReferenceHandles(sess.ReferenceCandidates, batch.ID)
	sess.ReferenceBatches = append(sess.ReferenceBatches, batch)
	closeReferenceBatchIfResolved(sess, len(sess.ReferenceBatches)-1, "no shortlisted reference candidates")
	return sess.ReferenceBatches[len(sess.ReferenceBatches)-1]
}

func mergeReferenceCandidate(existing, incoming ReferenceCandidate) ReferenceCandidate {
	if incoming.Confidence > existing.Confidence {
		existing.Reference = incoming.Reference
		existing.Confidence = incoming.Confidence
		existing.Source = incoming.Source
		existing.Detail = incoming.Detail
	}
	existing.Corroboration = mergeReferenceProbeHits(existing.Corroboration, incoming.Corroboration)
	if len(existing.Corroboration) == 0 {
		existing.Corroboration = []ProbeHit{{Probe: firstNonEmpty(existing.Source, incoming.Source), Score: existing.Confidence}}
	}
	return existing
}

func mergeReferenceProbeHits(existing, incoming []ProbeHit) []ProbeHit {
	out := append([]ProbeHit(nil), existing...)
	seen := map[string]struct{}{}
	for _, hit := range out {
		seen[strings.TrimSpace(hit.Probe)+"\x00"+strings.TrimSpace(hit.Concept)] = struct{}{}
	}
	for _, hit := range incoming {
		key := strings.TrimSpace(hit.Probe) + "\x00" + strings.TrimSpace(hit.Concept)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, hit)
	}
	return out
}

func curatePendingReferenceBatch(sess *Session, batchID string, batch *DiscoveryBatch) {
	var indexes []int
	var pending []ReferenceCandidate
	for i := range sess.ReferenceCandidates {
		if sess.ReferenceCandidates[i].BatchID != batchID || sess.ReferenceCandidates[i].Status != ReferenceCandidatePending {
			continue
		}
		indexes = append(indexes, i)
		pending = append(pending, sess.ReferenceCandidates[i])
	}
	curated, stats := curateReferenceBatch(pending)
	batch.CurationStats.OmittedBelowThreshold += stats.OmittedBelowThreshold
	batch.CurationStats.OmittedByCap += stats.OmittedByCap
	for i, candidate := range curated {
		sess.ReferenceCandidates[indexes[i]] = candidate
	}
}

func curateReferenceBatch(candidates []ReferenceCandidate) ([]ReferenceCandidate, CurationStats) {
	out := append([]ReferenceCandidate(nil), candidates...)
	sortReferenceCandidates(out)
	var stats CurationStats
	shortlisted := 0
	for i := range out {
		out[i].Tier = referenceTierLonglist
		out[i].HighConfidence = referenceCorroborationCount(out[i]) >= 2 || out[i].Confidence >= referenceHighConfidence
		if referenceCorroborationCount(out[i]) < 2 && out[i].Confidence != 0 && out[i].Confidence < referenceShortlistScore {
			stats.OmittedBelowThreshold++
			continue
		}
		if shortlisted >= referenceShortlistCap {
			stats.OmittedByCap++
			continue
		}
		shortlisted++
		out[i].Tier = referenceTierShortlist
	}
	return out, stats
}

func sortReferenceCandidates(candidates []ReferenceCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		aCorroborated := referenceCorroborationCount(a) >= 2
		bCorroborated := referenceCorroborationCount(b) >= 2
		if aCorroborated != bCorroborated {
			return aCorroborated
		}
		if a.Confidence != b.Confidence {
			return a.Confidence > b.Confidence
		}
		if a.Reference.Kind != b.Reference.Kind {
			return string(a.Reference.Kind) < string(b.Reference.Kind)
		}
		return strings.ToLower(a.Reference.Target) < strings.ToLower(b.Reference.Target)
	})
}

func assignReferenceHandles(candidates []ReferenceCandidate, batchID string) {
	short, long := 0, 0
	for i := range candidates {
		if candidates[i].BatchID != batchID || candidates[i].Status != ReferenceCandidatePending {
			continue
		}
		switch candidates[i].Tier {
		case referenceTierLonglist:
			long++
			candidates[i].Handle = "y" + itoa(long)
		default:
			short++
			candidates[i].Tier = referenceTierShortlist
			candidates[i].Handle = "r" + itoa(short)
		}
	}
}

func referenceCandidatesForBatch(candidates []ReferenceCandidate, batchID string) []ReferenceCandidate {
	out := make([]ReferenceCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.BatchID == batchID && candidate.Status == ReferenceCandidatePending && candidate.Tier == referenceTierShortlist {
			out = append(out, candidate)
		}
	}
	return out
}

func referenceDedupeKey(ref planmodel.Reference) string {
	target := strings.ToLower(strings.TrimSpace(ref.Target))
	if target == "" {
		return ""
	}
	return string(ref.Kind) + "\x00" + target
}

func targetReferenceBatchIndex(sess Session, batchID string) int {
	if strings.TrimSpace(batchID) != "" {
		return indexOfDiscoveryBatch(sess.ReferenceBatches, batchID)
	}
	for i := len(sess.ReferenceBatches) - 1; i >= 0; i-- {
		if sess.ReferenceBatches[i].Status == DiscoveryBatchPending {
			return i
		}
	}
	return -1
}

func latestPendingReferenceBatch(sess Session) (DiscoveryBatch, bool) {
	for i := len(sess.ReferenceBatches) - 1; i >= 0; i-- {
		if sess.ReferenceBatches[i].Status == DiscoveryBatchPending {
			return sess.ReferenceBatches[i], true
		}
	}
	return DiscoveryBatch{}, false
}

func pendingShortlistReferencesForBatch(candidates []ReferenceCandidate, batchID string) []int {
	var out []int
	for i := range candidates {
		if candidates[i].BatchID == batchID && candidates[i].Status == ReferenceCandidatePending && candidates[i].Tier == referenceTierShortlist {
			out = append(out, i)
		}
	}
	return out
}

func pendingReferencesForBatch(candidates []ReferenceCandidate, batchID string) []int {
	var out []int
	for i := range candidates {
		if candidates[i].BatchID == batchID && candidates[i].Status == ReferenceCandidatePending {
			out = append(out, i)
		}
	}
	return out
}

func closeReferenceBatchIfResolved(sess *Session, batchIdx int, note string) {
	if batchIdx < 0 || batchIdx >= len(sess.ReferenceBatches) {
		return
	}
	batchID := sess.ReferenceBatches[batchIdx].ID
	if len(pendingShortlistReferencesForBatch(sess.ReferenceCandidates, batchID)) > 0 {
		return
	}
	for _, idx := range pendingReferencesForBatch(sess.ReferenceCandidates, batchID) {
		candidate := sess.ReferenceCandidates[idx]
		candidate.Status = ReferenceCandidateRejected
		candidate.RejectionReason = "swept from longlist by reference-apply"
		sess.ReferenceCandidates[idx] = candidate
	}
	sess.ReferenceBatches[batchIdx].Status = DiscoveryBatchApplied
	if strings.TrimSpace(note) != "" {
		sess.ReferenceBatches[batchIdx].AppliedNote = strings.TrimSpace(note)
	}
}

func referenceCorroborationCount(candidate ReferenceCandidate) int {
	if len(candidate.Corroboration) == 0 {
		return 1
	}
	seen := map[string]struct{}{}
	for _, hit := range candidate.Corroboration {
		key := strings.TrimSpace(hit.Probe) + "\x00" + strings.TrimSpace(hit.Concept)
		if key == "\x00" {
			continue
		}
		seen[key] = struct{}{}
	}
	if len(seen) == 0 {
		return 1
	}
	return len(seen)
}
