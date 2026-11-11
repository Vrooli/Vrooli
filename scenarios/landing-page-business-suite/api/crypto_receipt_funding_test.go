package main

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
)

func TestAcceptedReceiptDatabaseContextRoutesTestModeAndRejectsOtherValues(t *testing.T) {
	plain := httptest.NewRequest("POST", "/api/v1/internal/crypto-funding/accepted-receipts", nil)
	ctx, err := acceptedReceiptDatabaseContext(plain)
	require.NoError(t, err)
	require.False(t, database.IsTestMode(ctx))

	testMode := httptest.NewRequest("POST", "/api/v1/internal/crypto-funding/accepted-receipts", nil)
	testMode.Header.Set("X-Vrooli-Test-Mode", "1")
	ctx, err = acceptedReceiptDatabaseContext(testMode)
	require.NoError(t, err)
	require.True(t, database.IsTestMode(ctx))

	invalid := httptest.NewRequest("POST", "/api/v1/internal/crypto-funding/accepted-receipts", nil)
	invalid.Header.Set("X-Vrooli-Test-Mode", "true")
	_, err = acceptedReceiptDatabaseContext(invalid)
	require.Error(t, err)
}
