// Package smoke runs UI smoke tests for Vrooli scenarios on the Browser
// Automation Studio (BAS) workflow engine.
//
// A UI smoke test validates that a scenario's web UI loads correctly inside the
// host iframe shell, establishes the iframe-bridge handshake (a hard-fail
// gate), and produces no critical errors. Results include a screenshot, console
// output, and network failures.
//
// # Architecture
//
//   - smoke (this package): the Runner orchestrates the engine-agnostic
//     preflight, drives the BAS capture, judges evidence via the shared
//     internal/evidence analyzer, and persists artifacts.
//   - internal/browsercapture: authors the inline smoke workflow and maps the
//     BAS timeline into engine-agnostic evidence.
//   - preflight: validates preconditions (UI directory, bundle freshness, UI
//     port, iframe-bridge dependency).
//   - artifacts: persists the screenshot, console, network, and raw evidence.
//
// The browser engine choice never leaks past internal/browsercapture; the smoke
// package speaks only evidence and results.
package smoke
