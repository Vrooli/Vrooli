package infra

import "strings"

func keyringPathFromMessage(message string) string {
	_, after, found := strings.Cut(message, keyringFormatMarker)
	if !found {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(after), ":"))
}

// corruptKeyringRemedies are user-scoped and never mint credentials or require
// root; this preserves the incident-forensics safety boundary.
func corruptKeyringRemedies(path string) []string {
	target := "the keyring file"
	if path != "" {
		target = path
	}
	return []string{
		"Run `vrooli credentials keyring inspect --format json` to see which entries gnome-keyring rejected in " + target + ".",
		"Run `vrooli credentials keyring repair --format json` to rewrite the malformed Vrooli-owned entries. It backs the file up first, declines entries other applications own, and needs no elevated privileges.",
		"Log out and back in, or reboot, so gnome-keyring-daemon reloads the repaired file.",
		"Confirm the remote-desktop credential metadata with `vrooli credentials status --identity vrooli/remote-desktop --field password`; provision it through the Vrooli credential authority if it is not configured.",
	}
}

func repairPendingRemedies(path string) []string {
	target := "the keyring file"
	if path != "" {
		target = path
	}
	return []string{
		"No further repair is needed: " + target + " parses correctly again.",
		"Log out and back in, or reboot, so gnome-keyring-daemon reloads it. Until then the running daemon still holds the file in its rejected state, and credential writes will hang on an unlock prompt nobody can answer.",
		"After logging back in, run `vrooli credentials doctor` and confirm the remote-desktop credential remains configured in the encrypted authority.",
	}
}
