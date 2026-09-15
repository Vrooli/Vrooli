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

func TestWindowsSyncCommandUsesNativeExtractionAndAtomicSwap(t *testing.T) {
	command := buildSyncRemoteCommandForPlatform(`C:\Users\node\Vrooli`, "windows")
	require.Contains(t, command, "powershell.exe -NoProfile -NonInteractive")
	require.Contains(t, command, "tar.exe -xf -")
	require.Contains(t, command, syncDestMarker)
	require.Contains(t, command, "Move-Item")
	require.NotContains(t, command, "chmod")
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
