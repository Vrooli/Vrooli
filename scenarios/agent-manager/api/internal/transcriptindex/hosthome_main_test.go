package transcriptindex_test

import (
	"os"
	"testing"

	"agent-manager/internal/testutil/hosthome"
)

// TestMain keeps every test in this package off the operator's real harness
// logins and fails the package if a test leaves a credential file behind.
func TestMain(m *testing.M) { os.Exit(hosthome.Main(m)) }
