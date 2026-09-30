package main

import "testing"

// A governed install relays a command to another machine and then waits for
// that machine to confirm it. Both happen inside one HTTP response, so the
// whole budget has to fit under the server's write timeout — otherwise the
// response is cut off mid-flight and the operator waits for nothing. The
// relay's original 180s ceiling was already over budget on its own, which is
// exactly the kind of thing that only shows up on a slow install.
func TestCapabilityInstallFitsInsideTheServerWriteTimeout(t *testing.T) {
	if capabilityRelayWindow+capabilityConfirmationWindow > capabilityInstallBudget {
		t.Fatalf("relay (%s) + confirmation (%s) exceeds the install budget (%s)",
			capabilityRelayWindow, capabilityConfirmationWindow, capabilityInstallBudget)
	}
	if capabilityInstallBudget >= httpWriteTimeout {
		t.Fatalf("install budget (%s) must leave headroom under the write timeout (%s)",
			capabilityInstallBudget, httpWriteTimeout)
	}
}

// Confirmation has to span more than one heartbeat or it is a coin flip: a
// Bridge node re-probes its tools once per heartbeat, so a window shorter than
// two of them would report "unconfirmed" for installs that plainly worked.
func TestConfirmationWindowSpansSeveralHeartbeats(t *testing.T) {
	const bridgeHeartbeatInterval = 15 // seconds; internal/config/config.go in vrooli-bridge/agent
	if capabilityConfirmationWindow.Seconds() < 2*bridgeHeartbeatInterval {
		t.Fatalf("confirmation window %s is shorter than two %ds heartbeats", capabilityConfirmationWindow, bridgeHeartbeatInterval)
	}
	if capabilityConfirmationInterval >= capabilityConfirmationWindow {
		t.Fatalf("poll interval %s does not fit inside the window %s", capabilityConfirmationInterval, capabilityConfirmationWindow)
	}
}
