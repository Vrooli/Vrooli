package main

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireFreshActionAuthentication(t *testing.T) {
	tests := []struct {
		name string
		auth ownerAuthContext
		want bool
	}{
		{name: "recent authenticator session", auth: ownerAuthContext{Role: "owner", AuthenticatedAt: time.Now().UTC().Add(-time.Minute)}, want: true},
		{name: "expired authenticator session", auth: ownerAuthContext{Role: "owner", AuthenticatedAt: time.Now().UTC().Add(-11 * time.Minute)}, want: false},
		{name: "static owner token", auth: ownerAuthContext{Role: "owner"}, want: false},
		{name: "agent cannot mint human assurance", auth: ownerAuthContext{Role: "agent", AuthenticatedAt: time.Now().UTC()}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/assurance", nil)
			setOwnerAuthContext(req, tt.auth)
			err := requireFreshActionAuthentication(req, false)
			if (err == nil) != tt.want {
				t.Fatalf("error = %v, want success=%t", err, tt.want)
			}
			if !tt.want && !errors.Is(err, errAssuranceRequired) {
				t.Fatalf("error = %v, want assurance requirement", err)
			}
		})
	}
	if err := requireFreshActionAuthentication(httptest.NewRequest("POST", "/", nil), true); err != nil {
		t.Fatalf("memory test seam rejected action: %v", err)
	}
}
