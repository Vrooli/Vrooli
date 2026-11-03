package main

import (
	"context"
	"errors"
	"landing-page-business-suite-api/internal/intelligence"
	"testing"
)

func TestAICompositionRequiresNoPersonalCreditAdapter(t *testing.T) {
	// Match main's deliberately uncomposed usage owner. Admission must refuse
	// before touching this nil owner, even when caller metadata names an app.
	service := intelligence.NewMeteredInferenceService(intelligence.MeteredInferenceServiceOptions{})
	for _, account := range []string{"", "acct-selected"} {
		_, err := service.ExecuteChat(context.Background(), "actor@example.test", intelligence.AIRequest{BusinessAccountID: account, Model: "openai/gpt-4o", Metadata: intelligence.AIMetadata{AppBundleKey: "caller-controlled"}})
		want := intelligence.ErrBillingScopeUnavailable
		if account == "" {
			want = intelligence.ErrBillingAccountRequired
		}
		if !errors.Is(err, want) {
			t.Fatalf("error=%v want=%v", err, want)
		}
	}
}
