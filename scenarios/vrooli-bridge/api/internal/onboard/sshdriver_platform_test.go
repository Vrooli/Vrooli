package onboard

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlatformProbeCommandSupportsWindowsOpenSSH(t *testing.T) {
	posix := platformProbeCommand()
	require.Contains(t, posix, "uname -s")
	require.NotContains(t, posix, "powershell.exe")

	windows := windowsPlatformProbeCommand()
	require.Contains(t, windows, "powershell.exe -NoProfile -NonInteractive")
	require.Contains(t, windows, "RuntimeInformation")
	require.Contains(t, windows, "VBPLATFORM=windows/")
	require.NotContains(t, strings.ToLower(windows), "processor_architecture")
}

func TestRemoteScriptPathUsesWindowsHomeRelativePowerShellPath(t *testing.T) {
	path, err := remoteScriptPath("windows")
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(path, ".ps1"))
	require.NotContains(t, path, "/")
	require.NotContains(t, path, "\\")
}

func TestWindowsShipCommandsExtractInPlace(t *testing.T) {
	extract := buildTreeExtractCommand(`C:\Users\node\Vrooli`, "windows")
	require.Contains(t, extract, "powershell.exe -NoProfile -NonInteractive")
	require.Contains(t, extract, "tar.exe -xUf - -C $dest")
	require.NotContains(t, extract, "chmod")
	// The checkout is never moved aside: node-owned ignored data lives in it.
	require.NotContains(t, extract, "Move-Item -Force -LiteralPath $dest")
	probe := buildTreeProbeCommand(`C:\Users\node\Vrooli`, "windows")
	require.Contains(t, probe, syncDestMarker)
	require.Contains(t, probe, syncDigestMarker)
	require.NotContains(t, buildTreeDeleteCommand("", "windows"), `"`+"'")
}

func TestWindowsArtifactPathsAndValidationUseAbsolutePaths(t *testing.T) {
	dir := `C:\Users\node\AppData\Local\Vrooli\Bridge\bootstrap\artifacts-test`
	path := remoteArtifactPath(dir, "vrooli.exe", "windows")
	require.Equal(t, dir+`\vrooli.exe`, path)
	require.True(t, validRemotePath(path, "windows"))
	command := windowsArtifactFinaliseCommand(RemoteArtifacts{
		Vrooli:    path,
		BridgeCLI: remoteArtifactPath(dir, "bridge.exe", "windows"),
		Agent:     remoteArtifactPath(dir, "agent.exe", "windows"),
	})
	require.Contains(t, command, "Test-Path")
}
