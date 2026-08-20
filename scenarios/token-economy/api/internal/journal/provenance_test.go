package journal_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/provenance"

	"token-economy/internal/journal"
)

// [REQ:TKE-P0-011] Every verification outcome uses the shared runtime-
// attribution vocabulary and is durably stamped at the only append seam.
func TestJournalProvenanceStatusMatrix(t *testing.T) {
	cases := []struct {
		status       string
		provenance   provenance.Provenance
		wantIdentity string
		wantKind     string
		wantRunID    string
	}{
		{
			status: journal.VerificationVerified,
			provenance: provenance.Provenance{
				Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
				Subject: "agent:chore-bot", ProfileKey: "household/chore-bot", RunID: "run-verified",
			},
			wantIdentity: "agent:chore-bot", wantKind: journal.ActorKindAgent, wantRunID: "run-verified",
		},
		{
			status: journal.VerificationUnavailable,
			provenance: provenance.Provenance{
				Actor: provenance.ActorOperator, VerificationStatus: provenance.VerificationUnavailable, Subject: "parent:alex",
			},
			wantIdentity: "parent:alex", wantKind: journal.ActorKindOperator,
		},
		{
			status: journal.VerificationInvalid,
			provenance: provenance.Provenance{
				Actor: provenance.ActorOperator, VerificationStatus: provenance.VerificationInvalid, Subject: "parent:alex",
			},
			wantIdentity: "parent:alex", wantKind: journal.ActorKindOperator,
		},
		{
			status: journal.VerificationAbsent,
			provenance: provenance.Provenance{
				Actor: provenance.ActorOperator, VerificationStatus: provenance.VerificationAbsent, Subject: "parent:alex",
			},
			wantIdentity: "parent:alex", wantKind: journal.ActorKindOperator,
		},
	}

	for index, testCase := range cases {
		t.Run(testCase.status, func(t *testing.T) {
			repo, db := newJournalRepository(t)
			seedTokenType(t, db, "chores")
			ctx := provenance.NewContext(context.Background(), testCase.provenance)
			at := time.Date(2026, 8, 19, 17, 0, index, 0, time.UTC)
			created, err := repo.Append(ctx, journal.Event{
				ID: fmt.Sprintf("event-%d", index), TokenTypeID: "chores", HolderID: "child:sam",
				Amount: 1, Kind: journal.EventKindCredit, CauseReference: "grant:matrix", CreatedAt: at,
			})
			require.NoError(t, err)
			require.Equal(t, testCase.wantIdentity, created.ActorIdentity)
			require.Equal(t, testCase.wantKind, created.ActorKind)
			require.Equal(t, testCase.status, created.ActorVerificationStatus)
			require.Equal(t, testCase.wantRunID, created.ActorRunID)

			stored, err := repo.Read(ctx, "child:sam", "chores")
			require.NoError(t, err)
			require.Equal(t, created, stored[0])
		})
	}
}
