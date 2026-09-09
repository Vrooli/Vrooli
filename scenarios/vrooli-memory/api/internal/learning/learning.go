// Package learning interprets agent attempt records. Source Ledger remains the
// sole journal authority; this package owns no database or retrieval ranking.
package learning

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	source "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	"google.golang.org/protobuf/encoding/protojson"
)

const Prefix = "learning-attempt/v1 "
const ObservationPrefix = "learning-observation/v1 "

func ValidateObservation(o *pb.Observation) error {
	if o == nil {
		return fmt.Errorf("observation is required")
	}
	for key, value := range map[string]string{"observation_id": o.ObservationId, "attempt_id": o.AttemptId, "disposition": o.Disposition, "provenance": o.Provenance} {
		if strings.TrimSpace(value) == "" || len(value) > 1024 || strings.ContainsAny(value, "\r\n\t\x00") {
			return fmt.Errorf("%s must contain 1..1024 bytes", key)
		}
	}
	if !oneOf(o.Disposition, "supported", "contradicted", "insufficient", "unavailable", "unresolved", "unknown") {
		return fmt.Errorf("invalid observation disposition")
	}
	if !oneOf(o.Provenance, "operator", "test") {
		return fmt.Errorf("provenance must be operator or test")
	}
	if len(o.MethodRevision) > 256 || len(o.Correction) > 2048 {
		return fmt.Errorf("observation revision or correction is too large")
	}
	if err := refs(o.EvidenceRefs); err != nil {
		return err
	}
	if o.Disposition != "unknown" && len(o.EvidenceRefs) == 0 {
		return fmt.Errorf("assessed observation requires evidence references")
	}
	when, err := time.Parse(time.RFC3339Nano, o.ObservedAt)
	if err != nil || when.After(time.Now().UTC()) {
		return fmt.Errorf("observed_at must be a historical RFC3339 timestamp")
	}
	return nil
}

func EncodeObservation(o *pb.Observation) (string, error) {
	if err := ValidateObservation(o); err != nil {
		return "", err
	}
	b, err := protojson.Marshal(o)
	return ObservationPrefix + string(b), err
}

func DecodeObservation(body string) (*pb.Observation, error) {
	body = strings.TrimSpace(body)
	if !strings.HasPrefix(body, ObservationPrefix) {
		return nil, fmt.Errorf("not a learning observation")
	}
	o := &pb.Observation{}
	if err := protojson.Unmarshal([]byte(strings.TrimPrefix(body, ObservationPrefix)), o); err != nil {
		return nil, err
	}
	return o, ValidateObservation(o)
}

func Validate(a *pb.Attempt) error {
	if a == nil {
		return fmt.Errorf("attempt is required")
	}
	for key, value := range map[string]string{"attempt_id": a.AttemptId, "task_id": a.TaskId, "operation": a.Operation, "context_key": a.ContextKey, "trigger": a.Trigger, "approach": a.Approach} {
		if strings.TrimSpace(value) == "" || len(value) > 1024 || strings.ContainsAny(value, "\r\n\t\x00") {
			return fmt.Errorf("%s must contain 1..1024 bytes", key)
		}
	}
	start, e1 := time.Parse(time.RFC3339Nano, a.StartedAt)
	end, e2 := time.Parse(time.RFC3339Nano, a.FinishedAt)
	if e1 != nil || e2 != nil || end.Before(start) {
		return fmt.Errorf("ordered RFC3339 started_at and finished_at are required")
	}
	taskStart, e3 := time.Parse(time.RFC3339Nano, a.TaskStartedAt)
	if e3 != nil || taskStart.After(start) || a.AttemptNumber < 1 {
		return fmt.Errorf("task_started_at must precede started_at; attempt_number must be positive")
	}
	if a.FirstActionAt != nil {
		action, err := time.Parse(time.RFC3339Nano, *a.FirstActionAt)
		if err != nil || action.Before(start) || action.After(end) {
			return fmt.Errorf("first_action_at must be inside the attempt interval")
		}
	}
	for _, count := range []*int32{a.ToolRoundTrips, a.VisualReasoningCalls} {
		if count != nil && (*count < 0 || *count > 100000) {
			return fmt.Errorf("effort counts must be between zero and 100000")
		}
	}
	if !oneOf(a.Outcome, "verified_success", "failed", "unavailable", "unknown") {
		return fmt.Errorf("invalid outcome")
	}
	if !oneOf(a.Provenance, "operator", "test") {
		return fmt.Errorf("provenance must be operator or test")
	}
	if !oneOf(a.RecallStatus, "matched", "no_match", "unavailable") {
		return fmt.Errorf("invalid recall_status")
	}
	if a.Outcome == "failed" && strings.TrimSpace(a.FailureFingerprint) == "" {
		return fmt.Errorf("failed outcome requires failure_fingerprint")
	}
	if a.Outcome != "failed" && a.FailureFingerprint != "" {
		return fmt.Errorf("only failed outcomes have failure_fingerprint")
	}
	if len(a.FailureFingerprint) > 256 {
		return fmt.Errorf("failure_fingerprint exceeds 256 bytes")
	}
	if err := refs(a.EvidenceRefs); err != nil {
		return err
	}
	if a.Outcome == "verified_success" && len(a.EvidenceRefs) == 0 {
		return fmt.Errorf("verified_success requires evidence references")
	}
	if len(a.Advice) > 10 {
		return fmt.Errorf("at most 10 advice uses")
	}
	if (a.RecallStatus == "matched") != (len(a.Advice) > 0) {
		return fmt.Errorf("matched recall requires advice uses; no_match/unavailable require none")
	}
	seen := map[string]bool{}
	for _, u := range a.Advice {
		if u == nil || strings.TrimSpace(u.EntryId) == "" || len(u.EntryId) > 128 || seen[u.EntryId] {
			return fmt.Errorf("advice entry IDs must be present and unique")
		}
		seen[u.EntryId] = true
		if !oneOf(u.Decision, "applied", "rejected", "unassessed") || !oneOf(u.Verdict, "supported", "contradicted", "unknown") {
			return fmt.Errorf("invalid advice decision or verdict")
		}
		if u.Decision == "unassessed" {
			if u.DecisionChange != "" || u.Verdict != "unknown" || len(u.EvidenceRefs) != 0 {
				return fmt.Errorf("unassessed advice records exposure only")
			}
		} else if strings.TrimSpace(u.DecisionChange) == "" || len(u.DecisionChange) > 1024 {
			return fmt.Errorf("advice requires bounded decision_change")
		}
		if err := refs(u.EvidenceRefs); err != nil {
			return err
		}
		if u.Verdict != "unknown" && len(u.EvidenceRefs) == 0 {
			return fmt.Errorf("assessed advice requires evidence references")
		}
	}
	return nil
}

func refs(values []string) error {
	if len(values) > 20 {
		return fmt.Errorf("at most 20 evidence references")
	}
	for _, v := range values {
		if strings.TrimSpace(v) == "" || len(v) > 512 {
			return fmt.Errorf("invalid evidence reference")
		}
	}
	return nil
}

func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}

func Encode(a *pb.Attempt) (string, error) {
	if err := Validate(a); err != nil {
		return "", err
	}
	b, err := protojson.Marshal(a)
	if err != nil {
		return "", err
	}
	var compact any
	if err = json.Unmarshal(b, &compact); err != nil {
		return "", err
	}
	b, err = json.Marshal(compact)
	return "Learning: " + a.Operation + " | " + a.Outcome + " | " + a.ContextKey + "\n" + Prefix + string(b), err
}

func Decode(body string) (*pb.Attempt, error) {
	body = payload(body)
	if body == "" {
		return nil, fmt.Errorf("not a learning record")
	}
	a := &pb.Attempt{}
	if err := protojson.Unmarshal([]byte(strings.TrimPrefix(body, Prefix)), a); err != nil {
		return nil, err
	}
	return a, Validate(a)
}

type (
	task   struct{ attempts []*pb.Attempt }
	cohort struct {
		out      *pb.Cohort
		tasks    map[string]*task
		failures map[string]int
	}
)

// Measure keeps comparison contexts separate and exposes missing denominators.
// Entries are journal snapshots selected by the caller with a fixed upper time.
func Measure(entries []*source.Entry, scope string, from, to time.Time, operation, contextKey string, truncated bool) *pb.MeasureLearningResponse {
	out := &pb.MeasureLearningResponse{Scope: scope, From: from.Format(time.RFC3339Nano), To: to.Format(time.RFC3339Nano), ScannedEntries: int32(len(entries)), Truncated: truncated, Interpretation: "Evidence-linked caller reports; operator/test is caller-declared. No causal or release-approval claim. Task timing covers observed attempts only; compare identical contexts and windows."}
	feedback := map[string]string{}
	for _, e := range entries {
		if e == nil || e.Kind != "attempt-observation" {
			continue
		}
		o, err := DecodeObservation(e.Body)
		if err != nil {
			out.InvalidRecords++
			continue
		}
		if previous, exists := feedback[o.AttemptId]; !exists || o.ObservedAt > previous {
			feedback[o.AttemptId] = o.Disposition
		}
	}
	groups := map[string]*cohort{}
	seen := map[string]string{}
	for _, e := range entries {
		if e == nil || e.CreatedAt == nil {
			out.InvalidRecords++
			continue
		}
		if !e.CreatedAt.AsTime().Before(to) {
			continue
		}
		if e.Kind != "task-record" {
			continue
		}
		if payload(e.Body) == "" {
			if !e.CreatedAt.AsTime().Before(from) {
				out.LegacyTaskRecords++
			}
			continue
		}
		a, err := Decode(e.Body)
		if err != nil {
			out.InvalidRecords++
			continue
		}
		finish, _ := time.Parse(time.RFC3339Nano, a.FinishedAt)
		if finish.Before(from) || !finish.Before(to) {
			continue
		}
		if operation != "" && a.Operation != operation || contextKey != "" && a.ContextKey != contextKey {
			continue
		}
		if a.Provenance == "test" {
			out.ExcludedTestAttempts++
			continue
		}
		canonical, _ := Encode(a)
		if previous, exists := seen[a.AttemptId]; exists {
			out.DuplicateAttempts++
			if previous != canonical {
				out.InvalidRecords++
			}
			continue
		}
		seen[a.AttemptId] = canonical
		out.EligibleAttempts++
		if len(out.EvidenceRefs) < 10 {
			out.EvidenceRefs = append(out.EvidenceRefs, e.Id)
		}
		key := a.Operation + "\x00" + a.ContextKey
		g := groups[key]
		if g == nil {
			g = &cohort{out: &pb.Cohort{Operation: a.Operation, ContextKey: a.ContextKey}, tasks: map[string]*task{}, failures: map[string]int{}}
			groups[key] = g
		}
		c := g.out
		c.Attempts++
		switch a.Outcome {
		case "verified_success":
			c.VerifiedSuccesses++
		case "failed":
			c.Failed++
			g.failures[a.FailureFingerprint]++
		case "unavailable":
			c.Unavailable++
		default:
			c.Unknown++
		}
		switch feedback[a.AttemptId] {
		case "supported":
			c.SupportedFeedback++
		case "contradicted":
			c.ContradictedFeedback++
		case "unresolved":
			c.UnresolvedFeedback++
		}
		switch a.RecallStatus {
		case "no_match":
			c.NoMatch++
		case "unavailable":
			c.RecallUnavailable++
		}
		for _, u := range a.Advice {
			if u.Decision == "applied" {
				c.AppliedAdvice++
			} else if u.Decision == "rejected" {
				c.RejectedAdvice++
			}
			switch u.Verdict {
			case "supported":
				c.SupportedAdvice++
			case "contradicted":
				c.ContradictedAdvice++
			default:
				c.UnassessedAdvice++
			}
		}
		t := g.tasks[a.TaskId]
		if t == nil {
			t = &task{}
			g.tasks[a.TaskId] = t
		}
		t.attempts = append(t.attempts, a)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		g := groups[key]
		c := g.out
		c.Tasks = int32(len(g.tasks))
		var counts, seconds, firstActions, toolCalls, visionCalls []float64
		reused := 0
		for _, task := range g.tasks {
			// First-action latency has one unambiguous first-attempt sample
			// per task. Retry-only or conflicting histories do not establish it.
			var initial *pb.Attempt
			initialCount := 0
			for _, a := range task.attempts {
				if a.AttemptNumber == 1 {
					initial = a
					initialCount++
				}
			}
			if initialCount == 1 && initial.FirstActionAt != nil {
				action, _ := time.Parse(time.RFC3339Nano, *initial.FirstActionAt)
				start, _ := time.Parse(time.RFC3339Nano, initial.TaskStartedAt)
				consistent := !start.Before(from)
				for _, a := range task.attempts {
					taskStart, _ := time.Parse(time.RFC3339Nano, a.TaskStartedAt)
					consistent = consistent && taskStart.Equal(start)
				}
				if consistent {
					firstActions = append(firstActions, action.Sub(start).Seconds())
				}
			}

			for _, a := range task.attempts {

				if a.ToolRoundTrips != nil {
					toolCalls = append(toolCalls, float64(*a.ToolRoundTrips))
				}
				if a.VisualReasoningCalls != nil {
					visionCalls = append(visionCalls, float64(*a.VisualReasoningCalls))
				}
				if a.ReusedWorkflow != nil {
					c.ReuseSamples++
					if *a.ReusedWorkflow {
						reused++
					}
				}
			}
		}
		c.FirstActionSamples = int32(len(firstActions))
		c.ToolRoundTripSamples = int32(len(toolCalls))
		c.VisualReasoningSamples = int32(len(visionCalls))
		c.MedianSecondsToFirstAction = median(firstActions)
		c.MedianToolRoundTrips = median(toolCalls)
		c.MedianVisualReasoningCalls = median(visionCalls)
		if c.ReuseSamples > 0 {
			rate := float64(reused) / float64(c.ReuseSamples)
			c.WorkflowReuseRate = &rate
		}
		for _, n := range g.failures {
			if n > 1 {
				c.RecurringFailureFingerprints++
				c.RepeatedFailures += int32(n - 1)
			}
		}
		for _, t := range g.tasks {
			sort.Slice(t.attempts, func(i, j int) bool {
				x, _ := time.Parse(time.RFC3339Nano, t.attempts[i].FinishedAt)
				y, _ := time.Parse(time.RFC3339Nano, t.attempts[j].FinishedAt)
				if x.Equal(y) {
					return t.attempts[i].AttemptId < t.attempts[j].AttemptId
				}
				return x.Before(y)
			})
			first, _ := time.Parse(time.RFC3339Nano, t.attempts[0].TaskStartedAt)
			incomplete := false
			ordinals := map[int32]bool{}
			for _, a := range t.attempts {
				s, _ := time.Parse(time.RFC3339Nano, a.TaskStartedAt)
				if !s.Equal(first) || ordinals[a.AttemptNumber] {
					incomplete = true
				}
				ordinals[a.AttemptNumber] = true
			}
			censored := first.Before(from) || incomplete
			completed := false
			for i, a := range t.attempts {
				if a.Outcome == "verified_success" {
					completed = true
					c.CompletedTasks++
					if a.AttemptNumber != int32(i+1) {
						censored = true
					}
					for n := int32(1); n <= a.AttemptNumber && n <= 1000; n++ {
						if !ordinals[n] {
							censored = true
						}
					}
					if a.AttemptNumber > 1000 {
						censored = true
					}
					if !censored {
						end, _ := time.Parse(time.RFC3339Nano, a.FinishedAt)
						counts = append(counts, float64(i+1))
						seconds = append(seconds, end.Sub(first).Seconds())
					}
					break
				}
			}
			if censored {
				c.LeftCensoredTasks++
			}
			if !completed {
				c.UnresolvedTasks++
			}
		}
		c.MedianAttemptsToSuccess = median(counts)
		c.MedianSecondsToSuccess = median(seconds)
		assessed := c.SupportedAdvice + c.ContradictedAdvice
		if assessed > 0 {
			r := float64(c.ContradictedAdvice) / float64(assessed)
			c.ContradictionRate = &r
		}
		out.Cohorts = append(out.Cohorts, c)
	}
	switch {
	case truncated:
		out.Reason = "unreliable:scan_limit"
	case out.InvalidRecords > 0:
		out.Reason = "unreliable:invalid_records"
	case out.LegacyTaskRecords > 0:
		out.Reason = "unreliable:legacy_task_records"
	case out.EligibleAttempts == 0:
		out.Reason = "unreliable:no_eligible_attempts"
	default:
		out.Reliable = true
	}
	return out
}

func median(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	sort.Float64s(values)
	n := len(values)
	v := values[n/2]
	if n%2 == 0 {
		v = (v + values[n/2-1]) / 2
	}
	return &v
}

func payload(body string) string {
	if strings.HasPrefix(body, Prefix) {
		return body
	}
	if strings.HasPrefix(body, "Learning: ") {
		_, rest, ok := strings.Cut(body, "\n")
		if ok && strings.HasPrefix(rest, Prefix) {
			return rest
		}
	}
	return ""
}
