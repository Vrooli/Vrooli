package cliinstall

import "path/filepath"

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
		InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: source, Prefix: filepath.Dir(source)},
		InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: link, Prefix: home},
	)
}
