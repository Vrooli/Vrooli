package config

import "testing"

func TestMergeSPFMechanismPreservesExistingMechanisms(t *testing.T) {
	got, err := MergeSPFMechanism([]string{"v=spf1 include:mailgun.org ~all"}, "include:sendgrid.net")
	if err != nil {
		t.Fatal(err)
	}
	if got != "v=spf1 include:mailgun.org include:sendgrid.net ~all" {
		t.Fatalf("merged SPF = %q", got)
	}
}

func TestMergeSPFMechanismIsIdempotentAndRejectsDuplicates(t *testing.T) {
	got, err := MergeSPFMechanism([]string{"v=spf1 include:mailgun.org ~all"}, "include:mailgun.org")
	if err != nil || got != "v=spf1 include:mailgun.org ~all" {
		t.Fatalf("idempotent merge = %q, err=%v", got, err)
	}
	if _, err := MergeSPFMechanism([]string{"v=spf1 include:a ~all", "v=spf1 include:b ~all"}, "include:c"); err == nil {
		t.Fatal("expected duplicate SPF records to be rejected")
	}
}
