package stagedmessages

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

const (
	maxProviderOmissions   = 32
	maxProviderReasonBytes = 256
)

var providerSecretPattern = regexp.MustCompile(`(?i)(api[_ -]?key|access[_ -]?token|password|secret|authorization)\s*[:=]\s*[^\s,;]+|\bbearer\s+[^\s,;]+|\bsk-[A-Za-z0-9_-]{8,}`)

// searchDraftContexts adapts advisory provider output to the immutable job
// contract. Keep this limit aligned with the Agent Manager Start boundary so
// admission cannot persist a job that the provider will reject immediately.
func (s *Service) searchDraftContexts(ctx context.Context, snapshot *v1.StagedMessageSnapshot) ([]*v1.StagedMessageContext, []*v1.Omission) {
	if s.provider == nil {
		return nil, []*v1.Omission{{Reason: "Agent Manager context unavailable"}}
	}
	contexts, omissions, err := s.provider.Search(ctx, snapshot)
	out := make([]*v1.StagedMessageContext, 0, min(len(contexts), MaxSelectedContexts))
	dropped := 0
	for _, candidate := range contexts {
		if candidate == nil {
			continue
		}
		if len(out) == MaxSelectedContexts {
			dropped++
			continue
		}
		out = append(out, candidate)
	}
	bounded := make([]*v1.Omission, 0, min(len(omissions)+2, maxProviderOmissions))
	add := func(o *v1.Omission) {
		if len(bounded) >= maxProviderOmissions || o == nil {
			return
		}
		reason := safeProviderReason(o.Reason)
		path := safeProviderReason(o.Path)
		if reason != "" || path != "" {
			bounded = append(bounded, &v1.Omission{Path: path, Reason: reason})
		}
	}
	for _, omission := range omissions {
		add(omission)
	}
	if dropped > 0 {
		add(&v1.Omission{Reason: fmt.Sprintf("additional Agent Manager contexts omitted by the %d-context workflow input bound", MaxSelectedContexts)})
	}
	if err != nil {
		add(&v1.Omission{Reason: "Agent Manager context unavailable: " + safeProviderError(err)})
	}
	return out, bounded
}

func safeProviderError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, err.Error())), " ")
	text = providerSecretPattern.ReplaceAllString(text, "$1=[redacted]")
	if text == "" {
		return "provider search failed"
	}
	return boundedText(text, maxProviderReasonBytes)
}

func safeProviderReason(text string) string {
	text = strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)), " ")
	return boundedText(text, maxProviderReasonBytes)
}
