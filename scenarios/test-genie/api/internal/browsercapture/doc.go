// Package browsercapture drives the UI smoke capture on the Browser Automation
// Studio (BAS) workflow engine.
//
// Smoke's contract is iframe-embedded and handshake-gated: a scenario UI is
// loaded inside a host iframe shell, the @vrooli/iframe-bridge child must signal
// ready (a hard-fail assertion), storage-shim state is read, the frame is
// screenshotted, and console/network observations are collected. That contract
// cannot run on BAS's CaptureService.Capture verb, which only performs a
// top-level navigate — the bridge child returns early and never installs its
// marker when window.parent === window (see iframeBridgeChild.ts). So smoke runs
// on the BAS workflow engine, which exposes navigate/evaluate/assert/screenshot
// primitives rich enough to reproduce the host-iframe embedding and the
// handshake poll.
//
// This package authors that workflow inline (a Go-built node graph), executes it
// through the shared BAS workflow client (internal/playbooks/execution.Client),
// and maps the resulting timeline into an engine-agnostic evidence.Evidence that
// the shared verdict authority (internal/evidence.Analyze) judges. The browser
// engine choice never leaks past this package.
package browsercapture
