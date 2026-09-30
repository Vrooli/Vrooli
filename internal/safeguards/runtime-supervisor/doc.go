// Package runtimesupervisorsafeguard converges the runtime supervisor's native
// unit exactly as autoheal_watchdog converges the loop's: render the one
// ServiceDefinition, ask the native manager whether it would load it, install
// it as the invoking user, enable it, restart it when its content changed, and
// then prove it is active before reporting applied.
//
// Until 2026-09-02 the supervisor unit was the only long-lived unit with no
// safeguard: it was rendered once by `vrooli runtime supervisor install` and
// never looked at again, which is how a unit rendered on 2026-08-18 crash-looped
// 495 times after a boot with an argv the CLI no longer accepted. `vrooli
// runtime supervisor install --user` now calls Converge, the same code this
// handler runs from setup and re-inspects in the readiness phase.
package runtimesupervisorsafeguard
