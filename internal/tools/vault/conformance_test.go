package vault

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.APTRepoToolCase(
		newTestHandler, testManifest, "apt-get install vault", "vault", "Vault v1.15.4\n",
		&hostreqkittest.APTRepoCase{
			Setup:          func() func() { return stubLookups(t) },
			SetKeyDownload: func(download func() ([]byte, error)) { KeyDownloadFn = download },
			MinCommands:    5,
		},
	))
}
