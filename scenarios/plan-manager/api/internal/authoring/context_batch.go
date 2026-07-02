package authoring

import (
	"strings"

	"github.com/google/uuid"
)

const (
	contextTierShortlist = "shortlist"
	contextTierLonglist  = "longlist"
)

func mergeContextDiscoveryBatch(sess *Session, concepts []string, complexity string, result ContextDiscoveryResult) DiscoveryBatch {
	return mergeContextDiscoveryBatchWithSource(sess, concepts, complexity, result, "interactive")
}

func mergeContextDiscoveryBatchWithSource(sess *Session, concepts []string, complexity string, result ContextDiscoveryResult, source string) DiscoveryBatch {
	source = strings.TrimSpace(source)
	if source == "" {
		source = "interactive"
	}
	batch := DiscoveryBatch{
		ID:         uuid.NewString(),
		Concepts:   normalizeConcepts(concepts, sess.Title),
		Complexity: strings.TrimSpace(complexity),
		Source:     source,
		ProbeNotes: normalizeProbeNotes(result.ProbeNotes),
		Status:     DiscoveryBatchPending,
		CreatedSeq: len(sess.DiscoveryBatches) + 1,
	}
	for i := range sess.DiscoveryBatches {
		if sess.DiscoveryBatches[i].Status == DiscoveryBatchPending {
			sess.DiscoveryBatches[i].Status = DiscoveryBatchSuperseded
		}
	}

	known := map[string]int{}
	for i := range sess.ContextCandidates {
		key := contextDedupeKey(sess.ContextCandidates[i].Item)
		if key != "" {
			known[key] = i
		}
	}

	for i := range sess.ContextCandidates {
		if sess.ContextCandidates[i].Status != ContextCandidatePending {
			continue
		}
		sess.ContextCandidates[i].BatchID = batch.ID
	}

	for _, incoming := range result.Candidates {
		incoming = normalizeContextCandidate(incoming)
		incoming.BatchID = batch.ID
		key := contextDedupeKey(incoming.Item)
		if key == "" {
			continue
		}
		if pos, ok := known[key]; ok {
			switch sess.ContextCandidates[pos].Status {
			case ContextCandidatePending:
				merged := mergeContextCandidate(sess.ContextCandidates[pos], incoming)
				merged.ID = sess.ContextCandidates[pos].ID
				merged.Item.ID = sess.ContextCandidates[pos].Item.ID
				merged.Status = ContextCandidatePending
				merged.BatchID = batch.ID
				sess.ContextCandidates[pos] = merged
			default:
				batch.CurationStats.SuppressedDispositioned++
			}
			continue
		}
		known[key] = len(sess.ContextCandidates)
		sess.ContextCandidates = append(sess.ContextCandidates, incoming)
	}

	curatePendingContextBatch(sess, batch.ID, &batch)
	assignContextHandles(sess.ContextCandidates, batch.ID)
	sess.DiscoveryBatches = append(sess.DiscoveryBatches, batch)
	closeContextBatchIfResolved(sess, len(sess.DiscoveryBatches)-1, "no shortlisted context candidates")
	batch = sess.DiscoveryBatches[len(sess.DiscoveryBatches)-1]
	return batch
}

func curatePendingContextBatch(sess *Session, batchID string, batch *DiscoveryBatch) {
	var indexes []int
	var pending []ContextCandidate
	for i := range sess.ContextCandidates {
		if sess.ContextCandidates[i].BatchID != batchID || sess.ContextCandidates[i].Status != ContextCandidatePending {
			continue
		}
		indexes = append(indexes, i)
		pending = append(pending, sess.ContextCandidates[i])
	}
	curated, stats := curateContextBatch(pending)
	batch.CurationStats.OmittedBelowThreshold += stats.OmittedBelowThreshold
	batch.CurationStats.OmittedTopicFiller += stats.OmittedTopicFiller
	batch.CurationStats.OmittedByCap += stats.OmittedByCap
	for i, candidate := range curated {
		sess.ContextCandidates[indexes[i]] = candidate
	}
}

func normalizeConcepts(concepts []string, fallback string) []string {
	out := make([]string, 0, len(concepts))
	for _, concept := range concepts {
		if concept = strings.TrimSpace(concept); concept != "" {
			out = append(out, concept)
		}
	}
	if len(out) == 0 {
		if fallback = strings.TrimSpace(fallback); fallback != "" {
			out = append(out, fallback)
		}
	}
	return out
}

func normalizeProbeNotes(notes []ProbeNote) []ProbeNote {
	out := make([]ProbeNote, 0, len(notes))
	for _, note := range notes {
		note.Probe = strings.TrimSpace(note.Probe)
		note.Concept = strings.TrimSpace(note.Concept)
		note.Detail = strings.TrimSpace(note.Detail)
		if note.Probe == "" && note.Concept == "" && note.Detail == "" {
			continue
		}
		out = append(out, note)
	}
	return out
}

func assignContextHandles(candidates []ContextCandidate, batchID string) {
	short, long := 0, 0
	for i := range candidates {
		if candidates[i].BatchID != batchID || candidates[i].Status != ContextCandidatePending {
			continue
		}
		switch candidates[i].Tier {
		case contextTierLonglist:
			long++
			candidates[i].Handle = "x" + itoa(long)
		default:
			short++
			candidates[i].Tier = contextTierShortlist
			candidates[i].Handle = "c" + itoa(short)
		}
	}
}

func itoa(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	return string(buf[i:])
}

func contextCandidatesForBatch(candidates []ContextCandidate, batchID string) []ContextCandidate {
	out := make([]ContextCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.BatchID == batchID && candidate.Status == ContextCandidatePending && candidate.Tier == contextTierShortlist {
			out = append(out, candidate)
		}
	}
	return out
}

func latestPendingDiscoveryBatch(sess Session) (DiscoveryBatch, bool) {
	for i := len(sess.DiscoveryBatches) - 1; i >= 0; i-- {
		if sess.DiscoveryBatches[i].Status == DiscoveryBatchPending {
			return sess.DiscoveryBatches[i], true
		}
	}
	return DiscoveryBatch{}, false
}

func LatestPendingDiscoveryBatch(sess Session) DiscoveryBatch {
	batch, _ := latestPendingDiscoveryBatch(sess)
	return batch
}

func LatestDiscoveryBatch(sess Session) DiscoveryBatch {
	if len(sess.DiscoveryBatches) == 0 {
		return DiscoveryBatch{}
	}
	return sess.DiscoveryBatches[len(sess.DiscoveryBatches)-1]
}

func latestAppliedDiscoveryBatch(sess Session) (DiscoveryBatch, bool) {
	for i := len(sess.DiscoveryBatches) - 1; i >= 0; i-- {
		if sess.DiscoveryBatches[i].Status == DiscoveryBatchApplied {
			return sess.DiscoveryBatches[i], true
		}
	}
	return DiscoveryBatch{}, false
}

func indexOfDiscoveryBatch(batches []DiscoveryBatch, id string) int {
	id = strings.TrimSpace(id)
	for i := range batches {
		if batches[i].ID == id {
			return i
		}
	}
	return -1
}

func targetContextBatchIndex(sess Session, batchID string) int {
	if strings.TrimSpace(batchID) != "" {
		return indexOfDiscoveryBatch(sess.DiscoveryBatches, batchID)
	}
	for i := len(sess.DiscoveryBatches) - 1; i >= 0; i-- {
		if sess.DiscoveryBatches[i].Status == DiscoveryBatchPending {
			return i
		}
	}
	return -1
}

func pendingShortlistCandidatesForBatch(candidates []ContextCandidate, batchID string) []int {
	var out []int
	for i := range candidates {
		if candidates[i].BatchID == batchID && candidates[i].Status == ContextCandidatePending && candidates[i].Tier == contextTierShortlist {
			out = append(out, i)
		}
	}
	return out
}

func pendingCandidatesForBatch(candidates []ContextCandidate, batchID string) []int {
	var out []int
	for i := range candidates {
		if candidates[i].BatchID == batchID && candidates[i].Status == ContextCandidatePending {
			out = append(out, i)
		}
	}
	return out
}

func indexOfContextCandidateInBatch(candidates []ContextCandidate, batchID, id string) int {
	id = strings.TrimSpace(id)
	if id == "" {
		return -1
	}
	for i := range candidates {
		if candidates[i].BatchID != batchID {
			continue
		}
		if candidates[i].ID == id || candidates[i].Handle == id {
			return i
		}
	}
	return -1
}

func closeContextBatchIfResolved(sess *Session, batchIdx int, note string) {
	if batchIdx < 0 || batchIdx >= len(sess.DiscoveryBatches) {
		return
	}
	batchID := sess.DiscoveryBatches[batchIdx].ID
	if len(pendingShortlistCandidatesForBatch(sess.ContextCandidates, batchID)) > 0 {
		return
	}
	for _, idx := range pendingCandidatesForBatch(sess.ContextCandidates, batchID) {
		candidate := sess.ContextCandidates[idx]
		candidate.Status = ContextCandidateRejected
		candidate.RejectionReason = "swept from longlist by batch apply"
		sess.ContextCandidates[idx] = candidate
	}
	sess.DiscoveryBatches[batchIdx].Status = DiscoveryBatchApplied
	if strings.TrimSpace(note) != "" {
		sess.DiscoveryBatches[batchIdx].AppliedNote = strings.TrimSpace(note)
	}
}
