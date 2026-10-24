package holders_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"token-economy/internal/holders"
)

type historySpy struct {
	holderID string
	subject  string
	history  holders.History
}

func (s *historySpy) Read(_ context.Context, holderID, subject string) (holders.History, error) {
	s.holderID = holderID
	s.subject = subject
	return s.history, nil
}

// [REQ:TKE-P0-012] The holder view returns the complete ordered history and
// reasons supplied by the scoped journal repository.
func TestServiceViewUsesAuthenticatedSubjectForCompleteHistory(t *testing.T) {
	repository := newRepository(t)
	holder := holders.Holder{
		ID: "holder-sam", DisplayName: "Sam", AuthenticatorSubject: "auth:sam",
		CreatedAt: time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
	}
	_, err := repository.Create(context.Background(), holder)
	require.NoError(t, err)
	history := holders.History{
		Events: []holders.HistoryEvent{
			{ID: "grant-1", TokenTypeID: "chores", Amount: 5, Kind: "credit", Reason: "weekly chores", CreatedAt: holder.CreatedAt.Add(time.Minute)},
			{ID: "redeem-1", TokenTypeID: "chores", Amount: 2, Kind: "debit", Reason: "movie reward", CreatedAt: holder.CreatedAt.Add(2 * time.Minute)},
		},
		Balances: []holders.Balance{{TokenTypeID: "chores", Amount: 3}},
	}
	spy := &historySpy{history: history}
	view, err := holders.NewService(repository, spy).View(context.Background(), holder.AuthenticatorSubject)
	require.NoError(t, err)
	require.Equal(t, holder, view.Holder)
	require.Equal(t, history, view.History)
	require.Equal(t, holder.ID, spy.holderID)
	require.Equal(t, holder.AuthenticatorSubject, spy.subject)
}
