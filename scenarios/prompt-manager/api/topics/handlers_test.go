package topics

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMatch_WithoutMatchFn_ReturnsEmpty(t *testing.T) {
	h := &Handlers{}

	body, _ := json.Marshal(MatchRequest{
		Queries: []string{"test"},
		Limit:   5,
	})
	req := httptest.NewRequest("POST", "/topics/match", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Match(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp MatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Method != "none" {
		t.Errorf("expected method 'none', got %q", resp.Method)
	}
	if len(resp.Topics) != 0 {
		t.Errorf("expected 0 topics, got %d", len(resp.Topics))
	}
	if len(resp.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(resp.Skills))
	}
}

func TestMatch_WithMatchFn_ReturnsResults(t *testing.T) {
	parentID := "parent"
	h := &Handlers{
		topicMatchFn: func(_ context.Context, queries []string, limit int) ([]MatchedTopic, []string, string, error) {
			return []MatchedTopic{
				{
					ID:            "topic-1",
					Name:          "Testing",
					Description:   "Testing practices",
					ParentTopicID: &parentID,
					Score:         0.85,
					ScorePercent:  85,
				},
			}, []string{"skill-a", "skill-b"}, "ai", nil
		},
	}

	body, _ := json.Marshal(MatchRequest{
		Queries: []string{"testing"},
		Limit:   5,
	})
	req := httptest.NewRequest("POST", "/topics/match", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Match(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp MatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Method != "ai" {
		t.Errorf("expected method 'ai', got %q", resp.Method)
	}
	if len(resp.Topics) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(resp.Topics))
	}
	if resp.Topics[0].ID != "topic-1" {
		t.Errorf("expected topic ID 'topic-1', got %q", resp.Topics[0].ID)
	}
	if len(resp.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(resp.Skills))
	}
}

func TestMatch_MissingQueries_Returns400(t *testing.T) {
	h := &Handlers{}

	body, _ := json.Marshal(MatchRequest{
		Queries: []string{},
	})
	req := httptest.NewRequest("POST", "/topics/match", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Match(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}
