package conformance

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	controlruntime "github.com/vrooli/vrooli/internal/runtime"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunGenericToolSuite(t, "claude", controlruntime.NewGenericToolHandler)
}
