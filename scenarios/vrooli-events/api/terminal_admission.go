package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/convert"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
	"google.golang.org/protobuf/types/known/structpb"
)

// Wire contract owned by Agent Manager runmodel.NewTerminalReceipt. It is
// deliberately distinct from the best-effort run notification / wake selector.
const terminalIntentEventType = "agent_manager.run.terminal.v1"

var terminalScopeToken = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
var terminalRunIdentity = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var terminalObservationDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)

type terminalAdmissionStore interface {
	CheckTerminalAdmissionTx(context.Context, *sql.Tx, store.Event, string) (*store.TerminalAdmission, error)
	InsertTerminalAdmissionTx(context.Context, *sql.Tx, store.TerminalAdmission) error
}

// acceptTerminalEventIntent is a dormant, trusted owner-to-owner seam, not an
// HTTP route or an authorization boundary. No production caller is wired.
// It accepts only the exact retained Agent Manager terminal intent and returns
// an acknowledgment after event, matching subscriptions, and replay receipt
// commit together. Errors (including uncertain commits) leave the sender's
// intent pending; retrying the identical bytes recovers the original receipt.
func (s *Server) acceptTerminalEventIntent(ctx context.Context, intentJSON, eventID string) (eventbus.DomainEventAcknowledgment, error) {
	var zero eventbus.DomainEventAcknowledgment
	if s == nil || ctx == nil || s.db == nil {
		return zero, errors.New("durable terminal admission requires context and database")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	receipts, ok := s.store.(terminalAdmissionStore)
	if !ok {
		return zero, errors.New("event store does not support terminal admission receipts")
	}
	env, canonical, err := eventbus.RetainedDomainEventEnvelope(intentJSON, eventID)
	if err != nil {
		return zero, err
	}
	if env.EventType != terminalIntentEventType || env.GetSource().GetScenario() != "agent-manager" {
		return zero, errors.New("terminal admission requires an Agent Manager terminal event")
	}
	var facts structpb.Struct
	if env.Data == nil || env.Data.UnmarshalTo(&facts) != nil {
		return zero, errors.New("terminal admission requires terminal facts")
	}
	if err := validateTerminalIntentFacts(&facts); err != nil {
		return zero, err
	}
	event, err := convert.EnvelopeToEvent(env)
	if err != nil {
		return zero, err
	}
	// EnvelopeToEvent's general protobuf marshaler is not deterministic. The
	// acknowledgment must name exactly the shared retained-contract bytes.
	event.Payload = canonical
	digest := sha256.Sum256(canonical)
	receipt := store.TerminalAdmission{EventID: eventID, IntentJSON: intentJSON, CanonicalPayload: canonical, CanonicalSHA256: hex.EncodeToString(digest[:])}
	id, err := s.withEventAdmissionTx(ctx, func(tx *sql.Tx, writer transactionalEventStore, queue subscription.AtomicQueueStore) (int64, error) {
		prior, err := receipts.CheckTerminalAdmissionTx(ctx, tx, event, intentJSON)
		if err != nil {
			return 0, err
		}
		if prior != nil {
			return prior.StoreID, nil
		}
		id, err := s.admitEventTx(ctx, tx, writer, queue, event)
		if err != nil {
			return 0, err
		}
		receipt.StoreID = id
		if err := receipts.InsertTerminalAdmissionTx(ctx, tx, receipt); err != nil {
			return 0, err
		}
		return id, nil
	})
	if err != nil {
		return zero, fmt.Errorf("accept terminal event intent: %w", err)
	}
	return eventbus.DomainEventAcknowledgment{Schema: eventbus.DomainEventAcknowledgmentSchema, EventID: eventID, CanonicalSHA256: receipt.CanonicalSHA256, StoreID: id}, nil
}

// All six fields are retained facts, not an instruction, work-acceptance claim,
// source-wide completeness proof, or authority to publish/resolve anything.
func validateTerminalIntentFacts(facts *structpb.Struct) error {
	fields := facts.GetFields()
	value := func(key string) string { return fields[key].GetStringValue() }
	version, err := strconv.ParseInt(value("lifecycle_version"), 10, 64)
	if len(fields) != 6 || !terminalScopeToken.MatchString(value("source_instance")) || !terminalScopeToken.MatchString(value("source_epoch")) || !terminalRunIdentity.MatchString(value("run_id")) || value("run_id") == "00000000-0000-0000-0000-000000000000" || !terminalObservationDigest.MatchString(value("terminal_observation_sha256")) || err != nil || version < 1 || strconv.FormatInt(version, 10) != value("lifecycle_version") {
		return errors.New("invalid native terminal intent facts")
	}
	switch value("status") {
	case "complete", "failed", "cancelled", "unknown":
		return nil
	default:
		return errors.New("terminal admission requires a terminal status")
	}
}
