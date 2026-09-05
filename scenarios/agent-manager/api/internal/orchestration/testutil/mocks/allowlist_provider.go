// Package mocks provides controllable collaborators for orchestration tests.
package mocks

import (
	"context"

	"agent-manager/internal/domain"
)

// FakeAllowlistProvider returns a configurable investigation tag allowlist.
// A nil rule slice preserves the production default fallback behavior.
type FakeAllowlistProvider struct {
	Rules []domain.InvestigationTagRule
}

func NewFakeAllowlistProvider(rules ...domain.InvestigationTagRule) *FakeAllowlistProvider {
	return &FakeAllowlistProvider{Rules: append([]domain.InvestigationTagRule(nil), rules...)}
}

func NewDefaultAllowlistProvider() *FakeAllowlistProvider {
	return &FakeAllowlistProvider{}
}

func (p *FakeAllowlistProvider) GetAllowlist(context.Context) []domain.InvestigationTagRule {
	if p == nil || p.Rules == nil {
		return domain.DefaultInvestigationTagAllowlist()
	}
	return append([]domain.InvestigationTagRule(nil), p.Rules...)
}
