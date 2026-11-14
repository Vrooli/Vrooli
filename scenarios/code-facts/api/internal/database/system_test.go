package database_test

import (
	"testing"

	localdb "code-facts/internal/database"
	"github.com/vrooli/api-core/databasetest"
)

func TestSystemSchema(t *testing.T) {
	databasetest.AssertEmptySystemSchema(t, localdb.SystemSchema)
}
