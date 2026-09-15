package stripe

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.APTRepoToolCase(
		newTestHandler, testManifest, "apt-get install stripe", "stripe", "stripe version 1.29.0\n",
		&hostreqkittest.APTRepoCase{
			Setup:          func() func() { return stubLookups(t) },
			SetKeyDownload: func(download func() ([]byte, error)) { KeyDownloadFn = download },
			MinCommands:    5,
			SudoMode:       "ask",
		},
		"apply_not_applicable_returns_early",
	))
}
