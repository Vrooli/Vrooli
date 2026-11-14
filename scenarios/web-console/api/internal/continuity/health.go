package continuity

import (
	"context"
	"fmt"
)

// HealthObservation reports current catalog-link health. It does not inventory
// every identity or read event bodies and is not a content-conservation receipt.
// Explicit Audit remains the owner of those stronger proofs.
type HealthObservation struct {
	UncatalogedConversations int64
	OrphanConversations      int64
	OrphanCheckpoints        int64
	OrphanWorkspacePanes     int64
}

func (a *IntegrityAuditor) ObserveHealth(ctx context.Context) (HealthObservation, error) {
	var observation HealthObservation
	for _, check := range []struct {
		name, query string
		destination *int64
	}{
		{"uncataloged_conversations", `SELECT COUNT(*) FROM conversation_sessions c LEFT JOIN sessions s ON s.id=c.session_id LEFT JOIN conversation_catalog k ON k.session_id=c.session_id WHERE s.id IS NULL AND k.session_id IS NULL`, &observation.UncatalogedConversations},
		{"orphan_conversations", `SELECT COUNT(*) FROM conversation_sessions c LEFT JOIN sessions s ON s.id=c.session_id WHERE s.id IS NULL`, &observation.OrphanConversations},
		{"orphan_checkpoints", `SELECT COUNT(*) FROM agent_transcript_checkpoints c LEFT JOIN sessions s ON s.id=c.web_console_session_id WHERE s.id IS NULL`, &observation.OrphanCheckpoints},
		{"orphan_workspace_panes", `SELECT COUNT(*) FROM workspace_panes p LEFT JOIN sessions s ON s.id=p.session_id WHERE s.id IS NULL`, &observation.OrphanWorkspacePanes},
	} {
		if err := a.db.QueryRowContext(ctx, check.query).Scan(check.destination); err != nil {
			return HealthObservation{}, fmt.Errorf("observe continuity %s: %w", check.name, err)
		}
	}
	return observation, nil
}
