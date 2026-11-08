package hostsession

import "github.com/vrooli/vrooli/internal/statecontract"

// PersistentState declares the fallback host-session token file, read when
// the host exposes no kernel boot identifier.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "host-session",
		Facts: map[string]any{"fallback_session_file": sessionFile},
	}
}
