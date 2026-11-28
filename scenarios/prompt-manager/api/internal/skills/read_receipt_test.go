package skills

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A reminder must reach an agent that has not read the governing skill, and
// must stay quiet for one that has. These tests pin the four receipt states.
func TestSkillReadReceiptReportsTheVerifiedRunsOwnRead(t *testing.T) {
	reads := newSkillReadService(t)
	h := &Handlers{readRecorder: NewReadRecorder(reads, nil, nil), identityVerifier: fixedIdentityVerifier{identity: &VerifiedWorkflowIdentity{RunID: "run-a"}}}

	before, err := h.SkillReadReceipt(context.Background(), "token-a", "notification-hub")
	if err != nil || before.Status != ReceiptNotRead || before.RunID != "run-a" {
		t.Fatalf("unread skill: got %+v err=%v, want not_read for run-a", before, err)
	}

	h.readRecorder.RecordWithRunID(httptest.NewRequest(http.MethodPost, "/skills/read", nil), []string{"notification-hub"}, "run-a")
	h.readRecorder.RecordWithRunID(httptest.NewRequest(http.MethodPost, "/skills/read", nil), []string{"notification-hub"}, "run-a")
	after, err := h.SkillReadReceipt(context.Background(), "token-a", "notification-hub")
	if err != nil || after.Status != ReceiptRead || after.ReadCount != 2 || after.FirstReadAt.IsZero() || after.LastReadAt.Before(after.FirstReadAt) {
		t.Fatalf("read skill: got %+v err=%v, want read twice", after, err)
	}
}

func TestSkillReadReceiptIgnoresOtherRunsAndUnverifiedReads(t *testing.T) {
	reads := newSkillReadService(t)
	recorder := NewReadRecorder(reads, nil, nil)
	recorder.RecordWithRunID(httptest.NewRequest(http.MethodPost, "/skills/read", nil), []string{"notification-hub"}, "run-other")
	// An unverified read carries no run, so it can never become a receipt.
	recorder.Record(httptest.NewRequest(http.MethodPost, "/skills/read", nil), []string{"notification-hub"})
	h := &Handlers{readRecorder: recorder, identityVerifier: fixedIdentityVerifier{identity: &VerifiedWorkflowIdentity{RunID: "run-a"}}}

	got, err := h.SkillReadReceipt(context.Background(), "token-a", "notification-hub")
	if err != nil || got.Status != ReceiptNotRead {
		t.Fatalf("got %+v err=%v, want not_read: another run's read is not this run's read", got, err)
	}
}

func TestSkillReadReceiptStaysFailOpenWithoutVerifiedIdentity(t *testing.T) {
	reads := newSkillReadService(t)
	cases := map[string]struct {
		h     *Handlers
		token string
	}{
		"no token":           {&Handlers{readRecorder: NewReadRecorder(reads, nil, nil), identityVerifier: fixedIdentityVerifier{identity: &VerifiedWorkflowIdentity{RunID: "run-a"}}}, ""},
		"verifier missing":   {&Handlers{readRecorder: NewReadRecorder(reads, nil, nil)}, "token"},
		"verification fails": {&Handlers{readRecorder: NewReadRecorder(reads, nil, nil), identityVerifier: fixedIdentityVerifier{err: errors.New("agent-manager down")}}, "token"},
	}
	for name, tc := range cases {
		got, err := tc.h.SkillReadReceipt(context.Background(), tc.token, "notification-hub")
		if err != nil || got.Status != ReceiptUnverified || got.Reason == "" || got.RunID != "" {
			t.Fatalf("%s: got %+v err=%v, want unverified with a reason", name, got, err)
		}
	}

	noStore := &Handlers{identityVerifier: fixedIdentityVerifier{identity: &VerifiedWorkflowIdentity{RunID: "run-a"}}}
	got, err := noStore.SkillReadReceipt(context.Background(), "token", "notification-hub")
	if err != nil || got.Status != ReceiptUnavailable {
		t.Fatalf("no store: got %+v err=%v, want unavailable", got, err)
	}
	if _, err := noStore.SkillReadReceipt(context.Background(), "token", " "); !errors.Is(err, ErrReceiptSkillRequired) {
		t.Fatalf("blank skill: err=%v, want ErrReceiptSkillRequired", err)
	}
}
