package main

import (
	"testing"

	"landing-page-business-suite-api/internal/commerce"
)

func TestNewReceiptValidatorsRegistersAppleAndGoogle(t *testing.T) {
	validators := newReceiptValidators(nil, nil)
	if len(validators) != 2 {
		t.Fatalf("receipt validator registry has %d entries, want Apple and Google", len(validators))
	}
	if _, ok := validators["apple"].(commerce.AppleSignedTransactionValidator); !ok {
		t.Fatalf("apple validator type = %T", validators["apple"])
	}
	if _, ok := validators["google"].(commerce.GooglePlayDeveloperValidator); !ok {
		t.Fatalf("google validator type = %T", validators["google"])
	}
}
