package backlog

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"

	"swarm-manager/internal/agentmanager"
)

// multipartClarificationRequest builds a multipart/form-data CreateClarification
// request with the given round_number string, exercising the same parse path the
// web-console file-upload UI uses.
func multipartClarificationRequest(t *testing.T, kind, name, roundNumber string) *httptest.ResponseRecorder {
	t.Helper()
	agent := &mockAgentService{result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"}}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	seedTwoInitiativesThreeItems(t, rootDir)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for field, value := range map[string]string{
		"round_number": roundNumber,
		"item_id":      "d2",
		"message":      "why option B?",
	} {
		if err := mw.WriteField(field, value); err != nil {
			t.Fatalf("write multipart field %s: %v", field, err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/backlog/"+kind+"/"+name+"/workshop/clarification", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = mux.SetURLVars(req, map[string]string{"kind": kind, "name": name})

	rec := httptest.NewRecorder()
	h.CreateClarification(rec, req)
	return rec
}

// TestCreateClarification_Multipart_RoundNumberBounds guards the round_number
// parse in the multipart path: it must reject values that cannot fit in int32,
// so the int32(rn) conversion never silently wraps to a bogus (possibly
// negative) round number.
func TestCreateClarification_Multipart_RoundNumberBounds(t *testing.T) {
	overInt32 := strconv.FormatInt(int64(1)<<32+5, 10) // 4294967301 > math.MaxInt32

	cases := []struct {
		name       string
		round      string
		wantReject bool
	}{
		{name: "zero rejected", round: "0", wantReject: true},
		{name: "negative rejected", round: "-1", wantReject: true},
		{name: "non-numeric rejected", round: "abc", wantReject: true},
		{name: "overflows int32 rejected", round: overInt32, wantReject: true},
		{name: "valid round accepted", round: "1", wantReject: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := multipartClarificationRequest(t, "idea", "gamma", tc.round)
			if tc.wantReject {
				if rec.Code != 400 {
					t.Fatalf("round_number %q: expected 400, got %d: %s", tc.round, rec.Code, rec.Body.String())
				}
				return
			}
			if rec.Code != 201 {
				t.Fatalf("round_number %q: expected 201, got %d: %s", tc.round, rec.Code, rec.Body.String())
			}
		})
	}
}
