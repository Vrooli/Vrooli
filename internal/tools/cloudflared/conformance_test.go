package cloudflared

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.APTRepoToolCase(
		newTestHandler, testManifest, "apt-get install cloudflared", "cloudflared", "cloudflared version 2024.2.1\n",
		&hostreqkittest.APTRepoCase{
			Setup:          func() func() { return stubLookups(t) },
			SetKeyDownload: func(download func() ([]byte, error)) { KeyDownloadFn = download },
			MinCommands:    5,
		},
	))
}
