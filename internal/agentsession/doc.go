// Package agentsession answers who owns a coding-agent session and whether
// that owner still exists, and stops a session only when its owner is
// confirmed gone.
//
// # The failure it prevents
//
// On 2026-10-07 `vrooli agent list` showed 89 vrooli-agent-* scopes holding
// 233 processes, 200 of them older than a day, with 5.7 GB resident and
// 17.9 GB in swap. Every row said "live". Forty-two were `opencode serve`
// helpers a bundled desktop app had started through the host agent path and
// that outlived the app by three weeks; others were Agent Manager runs and
// the operator's own web-console terminals. Nothing could tell a leak from a
// legitimate long-lived agent, because nothing recorded what a session
// belonged to.
//
// # The model
//
// Every session has an Owner of one kind: an Agent Manager run, a web-console
// session, a bundled desktop app instance, an operator terminal, or unknown.
// The owner is recorded at launch (the launcher's lease recorder calls
// OwnerAtLaunch) and backfilled for older sessions from the kernel and the
// session's own environment (Infer). It is never read from a scope name: the
// name is minted by the launcher and any process can claim any name.
//
// Liveness asks the owner's own authority, read-only: Agent Manager for a
// run's status, web-console for its session, the process table (pid plus
// start time, so a reused pid is not mistaken for the app) for a desktop app.
// A terminal or unknown owner is never orphaned automatically, and a probe
// that cannot answer leaves the owner unknown: an invented orphan is a reap
// target.
//
// # Platform shape
//
// Ownership, liveness and state derivation are platform-neutral. Process
// facts (parent, start time, controlling terminal) come from a build-tagged
// reader that is real on Linux and reports unsupported elsewhere, and stopping
// a session goes through platform-go's StopScope, which stops the systemd
// scope on Linux and returns ErrUnsupported on macOS and Windows until a
// process-group or Job Object adapter is host-verified.
package agentsession
