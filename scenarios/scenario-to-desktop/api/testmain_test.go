package main

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

// TestMain isolates storage: the server opens its database and artifact roots
// under the scenario's storage root, which would otherwise be the operator's.
func TestMain(m *testing.M) { os.Exit(storagetest.RunWithIsolatedStorage(m)) }
