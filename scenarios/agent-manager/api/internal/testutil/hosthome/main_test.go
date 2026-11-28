package hosthome_test

import (
	"os"
	"testing"

	"agent-manager/internal/testutil/hosthome"
)

func TestMain(m *testing.M) { os.Exit(hosthome.Main(m)) }
