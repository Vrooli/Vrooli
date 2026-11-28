package main

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

// TestMain isolates storage: the server resolves its state and cache class
// directories under the scenario's storage root, which would otherwise be the
// operator's live one.
func TestMain(m *testing.M) { os.Exit(storagetest.RunWithIsolatedStorage(m)) }
