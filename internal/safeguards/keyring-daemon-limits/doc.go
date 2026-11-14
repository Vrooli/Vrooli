// Package keyringdaemonlimits is the host safeguard that stops the operator's
// gnome-keyring-daemon from wedging on file-descriptor exhaustion.
//
// # The failure it prevents
//
// On 2026-09-01 the daemon's secrets child held 1,024 of 1,024 descriptors,
// 1,015 of them leaked eventfds. From that moment every accept() on its control
// socket failed. Three things follow from one wedged daemon:
//
//   - every host probe and secret lookup that touches it hangs or fails, which
//     is the "keyring wedges tax everything" incident recorded earlier;
//   - console sign-in stalls, because pam_gnome_keyring's auto_start hands the
//     login password to the daemon over that same control socket;
//   - rsyslog receives the failure line tens of thousands of times a second.
//     The two flat logs reached 320 GB in 70 hours and the disk hit 100%.
//
// # What it changes
//
// A systemd user drop-in at
// ~/.config/systemd/user/gnome-keyring-daemon.service.d/99-vrooli-limits.conf
// raises LimitNOFILE (default 65536). The leak still exists upstream, but at
// roughly a thousand descriptors a week it now takes more than a year to
// matter. Inspect also reports how many descriptors the running daemon holds
// against its effective limit; when that passes the saturation threshold
// (default 50%), Apply restarts the unit before it wedges. Setup is the one
// moment an operator is present, so that restart happens there rather than
// from a scenario.
//
// Everything is written and run as the invoking user: the unit belongs to the
// operator's session manager, not to root. macOS and Windows keep credentials
// in stores that do not have this failure mode.
//
// # What it does not do
//
// It does not fix the client that leaks. Finding that client is separate work;
// the descriptor count this safeguard reports is the measurement to use.
package keyringdaemonlimits
