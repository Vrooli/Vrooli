package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
)

type recordingPublisher struct{ events []eventbus.DomainEvent }

func (r *recordingPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	r.events = append(r.events, event)
	return nil
}

func TestProposalDigestSendsAtMostOnePhoneNoticePerDay(t *testing.T) {
	p := newProposalTransport(t)
	proposal := p.createViaAgent(t)
	publisher := &recordingPublisher{}
	ctx := context.Background()
	created := proposal.GetCreatedAt().AsTime()

	if sent, err := p.server.sendProposalDigest(ctx, publisher, 12*time.Hour, created.Add(time.Hour)); err != nil || sent {
		t.Fatalf("a proposal one hour old was announced: sent=%v err=%v", sent, err)
	}
	later := created.Add(13 * time.Hour)
	if sent, err := p.server.sendProposalDigest(ctx, publisher, 12*time.Hour, later); err != nil || !sent {
		t.Fatalf("first digest: sent=%v err=%v", sent, err)
	}
	if sent, err := p.server.sendProposalDigest(ctx, publisher, 12*time.Hour, later.Add(time.Hour)); err != nil || sent {
		t.Fatalf("second digest on the same day: sent=%v err=%v", sent, err)
	}
	if sent, err := p.server.sendProposalDigest(ctx, publisher, 12*time.Hour, later.Add(24*time.Hour)); err != nil || !sent {
		t.Fatalf("next day's digest: sent=%v err=%v", sent, err)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("events = %d", len(publisher.events))
	}
	event := publisher.events[0]
	if event.EventType != proposalDigestEventType || event.Source != "git-control-tower" {
		t.Fatalf("event = %#v", event)
	}
	message, _ := event.Payload["message"].(string)
	if !strings.HasPrefix(message, "1 commit proposal waits for review in Git Control Tower (oldest 13h). bas: E27") {
		t.Fatalf("message = %q", message)
	}
	for _, producerCopy := range []string{"title", "body", "sensitivity_label"} {
		if _, ok := event.Payload[producerCopy]; ok {
			t.Fatalf("payload carries hub-owned copy %q", producerCopy)
		}
	}
	if ids, _ := event.Payload["proposal_ids"].([]any); len(ids) != 1 || ids[0] != proposal.GetId() {
		t.Fatalf("proposal_ids = %v", event.Payload["proposal_ids"])
	}
}

func TestProposalDigestIsOffByDefault(t *testing.T) {
	t.Setenv(proposalDigestEnv, "")
	if proposalDigestEnabled() {
		t.Fatal("digest enabled without the switch")
	}
	t.Setenv(proposalDigestEnv, "true")
	if !proposalDigestEnabled() {
		t.Fatal("digest disabled with the switch on")
	}
}
