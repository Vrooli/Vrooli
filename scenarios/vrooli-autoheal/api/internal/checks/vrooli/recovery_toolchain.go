package vrooli

import (
	"os"
	"runtime"

	repocontract "github.com/vrooli/repo-contract-go"
	langrecover "vrooli-autoheal-langrecover"
)

// recoveryRunner is the command runner the API's dependency-drift repairs
// use: PATH first, then langrecover's per-OS toolchain table. The table's
// runtime-home bin entry comes from the repository contract here because
// langrecover is stdlib-only and cannot resolve it. Built per repair so a bin
// directory installed after the API started is still searched.
func recoveryRunner() langrecover.Runner {
	home, _ := os.UserHomeDir()
	return langrecover.NewRunner(recoveryToolchainEntries(runtime.GOOS, home))
}

// recoveryToolchainEntries is the table for goos and the account home, with
// the contract-resolved runtime-home bin. An unresolvable contract omits only
// that entry; the OS toolchain homes are still searched.
func recoveryToolchainEntries(goos, home string) []string {
	runtimeBin := ""
	if home != "" {
		if path, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin); err == nil {
			runtimeBin = path
		}
	}
	return langrecover.DefaultPathEntries(goos, home, runtimeBin)
}
