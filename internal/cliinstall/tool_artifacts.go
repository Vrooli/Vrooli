package cliinstall

// RecordGoToolInstallArtifacts records the binary and PATH link created by a
// Go-backed per-user tool installer.
func RecordGoToolInstallArtifacts(home, source, prefix, link string) error {
	return RecordToolArtifacts(home,
		InstallEntry{Scope: ScopeRuntime, Kind: EntryBinary, Path: source, Prefix: prefix},
		InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: link, Prefix: home},
	)
}

// RecordNPMToolInstallArtifacts records the npm prefix, shim, and PATH link
// created by an npm-backed per-user tool installer.
func RecordNPMToolInstallArtifacts(home, source, prefix, link string) error {
	return RecordToolArtifacts(home,
		InstallEntry{Scope: ScopeRuntime, Kind: EntryDirectory, Path: prefix, Prefix: prefix},
		// npm's bin shim is a symlink into the package tree under the same
		// prefix (.bin/x -> ../@scope/pkg/cli.js), so it belongs to the prefix.
		InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: source, Prefix: prefix},
		InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: link, Prefix: home},
	)
}
