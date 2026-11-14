package database_test

import (
	"testing"

	"github.com/vrooli/api-core/databasetest"
	localdb "typescript-code-graph/internal/database"
)

func TestSystemSchema(t *testing.T) {
	databasetest.AssertEmptySystemSchema(t, localdb.SystemSchema)
}
