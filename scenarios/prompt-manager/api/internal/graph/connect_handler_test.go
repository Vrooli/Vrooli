package graph

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph"
)

func sampleScores() []HealthScore {
	return []HealthScore{
		{
			NodeID:  "skill:scientific-debugging",
			Score:   0.82,
			Factors: map[string]float64{"freshness": 0.9, "usage": 0.7},
			Messages: []HealthMessage{
				{
					Key:            "stale-usage",
					Severity:       "warning",
					Factor:         "usage",
					Summary:        "Low recent usage",
					Detail:         "No invocations in 30 days",
					Recommendation: "Surface this skill in onboarding",
					MetricValue:    0.7,
					Target:         ">= 0.8",
				},
			},
		},
		{
			NodeID:  "action:report-bug",
			Score:   0.41,
			Factors: map[string]float64{},
		},
	}
}

func TestConnectGetHealthScores_Mapping(t *testing.T) {
	h := NewConnectHandler(&mockGraphIndexProvider{idx: testIndex(nil, nil, sampleScores())})

	resp, err := h.GetHealthScores(context.Background(), connect.NewRequest(&graphv1.GetHealthScoresRequest{}))
	if err != nil {
		t.Fatalf("GetHealthScores: %v", err)
	}
	got := resp.Msg.GetScores()
	if len(got) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(got))
	}

	first := got[0]
	if first.GetNodeId() != "skill:scientific-debugging" {
		t.Errorf("node_id: got %q", first.GetNodeId())
	}
	if first.GetScore() != 0.82 {
		t.Errorf("score: got %v", first.GetScore())
	}
	if first.GetFactors()["freshness"] != 0.9 || first.GetFactors()["usage"] != 0.7 {
		t.Errorf("factors: got %v", first.GetFactors())
	}
	if len(first.GetMessages()) != 1 {
		t.Fatalf("expected 1 message, got %d", len(first.GetMessages()))
	}
	m := first.GetMessages()[0]
	if m.GetKey() != "stale-usage" || m.GetSeverity() != "warning" || m.GetFactor() != "usage" {
		t.Errorf("message head mismatch: %+v", m)
	}
	if m.GetSummary() != "Low recent usage" || m.GetDetail() != "No invocations in 30 days" {
		t.Errorf("message body mismatch: %+v", m)
	}
	if m.GetRecommendation() != "Surface this skill in onboarding" || m.GetMetricValue() != 0.7 || m.GetTarget() != ">= 0.8" {
		t.Errorf("message tail mismatch: %+v", m)
	}

	// A score with an empty Factors map round-trips as an empty/absent map.
	if len(got[1].GetMessages()) != 0 {
		t.Errorf("expected no messages on second score, got %d", len(got[1].GetMessages()))
	}
}

func TestConnectGetHealthScores_Error(t *testing.T) {
	h := NewConnectHandler(&mockGraphIndexProvider{getErr: errors.New("index unavailable")})
	_, err := h.GetHealthScores(context.Background(), connect.NewRequest(&graphv1.GetHealthScoresRequest{}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("expected CodeInternal, got %v", connect.CodeOf(err))
	}
}

// TestConnectRESTWireParity asserts the Connect protojson serialization of one
// HealthScore matches the legacy REST JSON shape (camelCase keys from
// api/graph/models.go HealthScore/HealthMessage json tags). MoM consumes the
// typed Go message, but this guards against a silent wire-casing divergence
// between the additive Connect surface and the kept REST surface.
func TestConnectRESTWireParity(t *testing.T) {
	scores := sampleScores()
	h := NewConnectHandler(&mockGraphIndexProvider{idx: testIndex(nil, nil, scores)})
	resp, err := h.GetHealthScores(context.Background(), connect.NewRequest(&graphv1.GetHealthScoresRequest{}))
	if err != nil {
		t.Fatalf("GetHealthScores: %v", err)
	}

	// Connect side: protojson of the first proto HealthScore.
	connectJSON, err := protojson.Marshal(resp.Msg.GetScores()[0])
	if err != nil {
		t.Fatalf("protojson marshal: %v", err)
	}
	var connectMap map[string]any
	if err := json.Unmarshal(connectJSON, &connectMap); err != nil {
		t.Fatalf("connect json: %v", err)
	}

	// REST side: encoding/json of the domain HealthScore (what /graph/health emits).
	restJSON, err := json.Marshal(scores[0])
	if err != nil {
		t.Fatalf("rest marshal: %v", err)
	}
	var restMap map[string]any
	if err := json.Unmarshal(restJSON, &restMap); err != nil {
		t.Fatalf("rest json: %v", err)
	}

	for _, key := range []string{"nodeId", "score", "factors", "messages"} {
		if _, ok := restMap[key]; !ok {
			t.Errorf("REST JSON missing expected key %q", key)
		}
		if _, ok := connectMap[key]; !ok {
			t.Errorf("Connect protojson missing expected key %q (wire-casing divergence)", key)
		}
	}
}
