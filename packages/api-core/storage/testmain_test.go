package storage

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

// TestMain runs the package under a private HOME. Many tests here resolve
// default class roots through an injected environment; with the account home
// those defaults are the operator's live storage, which the resolver refuses
// under test.
func TestMain(m *testing.M) { os.Exit(storagetest.RunWithPrivateHome(m)) }
