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
		"Run `secrets-manager keyring inspect` to see which entries gnome-keyring rejected in " + target + ".",
		"Run `secrets-manager keyring repair` to rewrite the malformed Vrooli-owned entries. It backs the file up first, declines entries other applications own, and needs no elevated privileges.",
		"Log out and back in, or reboot, so gnome-keyring-daemon reloads the repaired file.",
		"Confirm the RDP password survived with `grdctl status`; if it reads (empty), set it again with `grdctl rdp set-credentials <username> <password>`.",
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
		"After logging back in, check `grdctl status`; if the RDP password reads (empty), set it again with `grdctl rdp set-credentials <username> <password>`.",
	}
}
