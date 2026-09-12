// Module vrooli-autoheal-langrecover is the recovery floor's signature and
// strategy layer. It is deliberately a standalone module with ZERO third-party
// dependencies (stdlib only).
//
// Why its own module: the autoheal loop (the watchdog that must keep working
// when the API cannot build) needs these detectors. If the loop reached them
// by importing the api module, the loop would inherit api-core's dependency
// graph -- and a shared-package change that breaks api-core would then break
// the very watchdog meant to repair it. That is exactly the 2026-09-01 outage,
// where a new api-core import broke 98 scenario modules and the healer could
// not build itself. Keeping this module stdlib-only makes the recovery floor
// unbreakable by dependency drift.
//
// Do not add a require directive to this file.
module vrooli-autoheal-langrecover

go 1.25.0
