package metrics

import "testing"

func TestSnapshotReadsAllCounters(t *testing.T) {
	m := New()
	m.SessionsCreated.Store(1)
	m.SessionsDeleted.Store(2)
	m.ActiveSessions.Store(3)
	m.ConnectionsTotal.Store(4)
	m.ActiveConnections.Store(5)
	m.WSMessagesSent.Store(6)
	m.WSMessagesReceived.Store(7)
	m.ResizeCount.Store(8)
	m.ReattachAttempts.Store(9)
	m.ReattachSuccesses.Store(10)
	m.ReattachFailures.Store(11)
	m.RecoveryRecovered.Store(12)
	m.RecoveryOrphanedMeta.Store(13)
	m.RecoveryOrphanedTmux.Store(14)
	m.RecoveryAttachRetries.Store(15)
	m.RecoveryPreservedForNow.Store(16)
	m.AIGenerations.Store(17)
	m.AISuggestions.Store(18)
	m.VoiceSkipVerificationTotal.Store(19)
	s := m.Snapshot()
	if s.Sessions.Created != 1 || s.Recovery.PreservedForNow != 16 || s.VoiceSkipVerificationTotal != 19 {
		t.Fatalf("snapshot = %#v", s)
	}
}
