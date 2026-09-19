package runtimeapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/maintenance"
)

func TestExecutorScopeRejectsIncompleteAndOversizedInput(t *testing.T) {
	for _, body := range []string{"", "not-json", `{}`, `{"executors":[]}`, `{"executors":[{"runId":"one"}]}`, strings.Repeat("x", maintenance.ExecutorScopeMaxBytes+1)} {
		var out bytes.Buffer
		if err := (&Service{}).ExecutorScope(t.Context(), strings.NewReader(body), &out); err == nil || out.Len() != 0 {
			t.Fatalf("invalid input produced usable evidence: err=%v out=%q", err, out.String())
		}
	}
}

func TestExecutorScopeWireCapacityRetainsAllManagedHistory(t *testing.T) {
	for _, count := range []int{6000, maintenance.ExecutorScopeMaxReferences} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			request := maintenance.ExecutorScopeRequest{Executors: make([]maintenance.ExecutorScopeRef, count)}
			report := maintenance.ExecutorScopeReport{SchemaVersion: "executor-scope-v1", Complete: true, Executors: make([]maintenance.ExecutorScopeEvidence, count)}
			for i := range request.Executors {
				id := fmt.Sprintf("%08d-0000-0000-0000-000000000000", i)
				request.Executors[i] = maintenance.ExecutorScopeRef{RunID: id, Tag: id, LegacyTag: fmt.Sprintf("opencode-continue-%08d", i), PID: i + 100000, PGID: i + 100000}
				report.Executors[i] = maintenance.ExecutorScopeEvidence{RunID: id, State: "absent", PIDs: []int{}, Reasons: []string{}}
			}
			body, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := decodeExecutorScopeRequest(bytes.NewReader(body))
			if err != nil || len(decoded.Executors) != count {
				t.Fatalf("wire request truncated normal history: count=%d err=%v", len(decoded.Executors), err)
			}
			var out bytes.Buffer
			if err := writeExecutorScopeReport(t.Context(), &out, report); err != nil {
				t.Fatal(err)
			}
			var actual maintenance.ExecutorScopeReport
			if err := json.Unmarshal(out.Bytes(), &actual); err != nil || len(actual.Executors) != count {
				t.Fatalf("wire response lost identities: count=%d err=%v", len(actual.Executors), err)
			}
			for i := range report.Executors {
				if actual.Executors[i].RunID != request.Executors[i].RunID {
					t.Fatalf("wrong identity at %d", i)
				}
			}
			t.Logf("%d references: input=%d bytes output=%d bytes", count, len(body), out.Len())
		})
	}
}

func TestExecutorScopeWireRefusesOverflowOrExpiredProofWithoutPartialJSON(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprint(expired), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			report := maintenance.ExecutorScopeReport{SchemaVersion: "executor-scope-v1", Complete: true}
			if expired {
				cancel()
			} else {
				report.Executors = []maintenance.ExecutorScopeEvidence{{RunID: strings.Repeat("x", maintenance.ExecutorScopeMaxBytes), State: "absent"}}
			}
			var out bytes.Buffer
			if err := writeExecutorScopeReport(ctx, &out, report); err == nil || out.Len() != 0 {
				t.Fatalf("unusable proof emitted: err=%v bytes=%d", err, out.Len())
			}
		})
	}
}
