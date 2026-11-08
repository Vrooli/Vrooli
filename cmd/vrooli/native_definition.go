package main

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
)

// The candidate exports its own compiled native definition before ordinary
// initialization. Admission freezes this output; recovery never re-renders it.
func writeNativeSupervisorDefinition(args []string, stdout, stderr io.Writer) (bool, int) {
	if len(args) == 0 || args[0] != "--native-supervisor-definition" {
		return false, 0
	}
	if len(args) != 5 {
		_, _ = fmt.Fprintln(stderr, "native supervisor definition requires home, source hint, executable and log paths")
		return true, 2
	}
	for _, path := range args[1:] {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "\x00\r\n") {
			_, _ = fmt.Fprintln(stderr, "native supervisor definition requires canonical absolute paths")
			return true, 2
		}
	}
	if err := buildinfo.PinCompiledRuntimeHome(); err != nil {
		_, _ = fmt.Fprintln(stderr, "native supervisor definition requires compiled runtime-home authority")
		return true, 1
	}
	home, root, executable, logPath := args[1], args[2], args[3], args[4]
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "native supervisor executable authority unavailable")
		return true, 1
	}
	logs, err := runtimesupervisor.LogPath(home)
	if err != nil || filepath.Join(bin, "vrooli") != executable || logs != logPath {
		_, _ = fmt.Fprintln(stderr, "native supervisor definition differs from compiled path authority")
		return true, 1
	}
	definition, err := platform.RuntimeSupervisorDefinition(runtime.GOOS, platform.RuntimeSupervisorOptions{Home: home, SourceRoot: root, Executable: executable, LogPath: logPath})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "native supervisor definition is unsupported")
		return true, 1
	}
	artifact, err := platform.RenderDefinition(definition, runtime.GOOS)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "native supervisor definition could not be rendered")
		return true, 1
	}
	if err := json.NewEncoder(stdout).Encode(artifact); err != nil {
		return true, 1
	}
	return true, 0
}
