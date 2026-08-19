package book_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"treasury/internal/book"
)

// [REQ:TRS-P0-010] The real schema admits many books for the operator but no
// second beneficiary, even when a future caller bypasses an HTTP boundary.
func TestSQLiteRepositoryEnforcesOneOperatorBeneficiary(t *testing.T) {
	databaseHandle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), databaseHandle, database.SchemaProviderFunc(book.Schema)))
	service := book.NewService(book.NewSQLiteRepository(databaseHandle), schedule.NewFake(time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)))

	_, err := service.Create(context.Background(), book.CreateInput{ID: "personal", Name: "Personal", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	_, err = service.Create(context.Background(), book.CreateInput{ID: "business", Name: "Business", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err, "multiple books may separate contexts for the same beneficiary")
	_, err = service.Create(context.Background(), book.CreateInput{ID: "third-party", Name: "Third party", BeneficiaryIdentity: "customer:2"})
	require.ErrorIs(t, err, book.ErrBeneficiaryConflict)
}
