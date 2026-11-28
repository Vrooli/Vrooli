package database

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

// TestMain runs the package under a private HOME. The DSN tests resolve SQLite
// paths through an injected environment, whose defaults would otherwise be the
// operator's live databases, which the storage resolver refuses under test.
func TestMain(m *testing.M) { os.Exit(storagetest.RunWithPrivateHome(m)) }
