# Proposal: embeddable Go API contract for desktop shells

Status: draft for template-manager owner review; E4 evidence is incomplete.

## Proposed contract

A Go API intended for an in-process desktop shell should expose an importable package at `api/desktop` with a constructor returning an `http.Handler`, an explicit lifecycle owner, and an error:

```go
 type Lifecycle interface {
     Start(context.Context) error
     Stop(context.Context) error
 }
 func New() (http.Handler, Lifecycle, error)
```

`New` should construct handlers without opening a listener or starting background work. `Start` owns background resources and is safe to call once; `Stop` closes them and is safe during shutdown. A thin standalone `main` may bind the same handler to loopback for existing sidecar use. Desktop shells mount the handler on their asset server and call the lifecycle methods from shell startup/shutdown hooks.

## Current evidence

The `hello-desktop` reference fixture now exports this contract at `scenarios/hello-desktop/api/desktop/embed.go`. Its standalone `api/main.go` uses the same handler with a loopback listener and graceful shutdown. Focused `go test ./...` in `scenarios/hello-desktop/api` passes, including route behavior and lifecycle start/stop.

## Evidence still needed before template-wide adoption

- Prove the same contract against other selected reference APIs, including their background resources.
- Complete Wails in-process generation, build, and packaged launch through scenario-to-desktop.
- Retain Linux WebKitWebDriver journey receipts and screenshots after operator host setup.
- Keep Electron sidecar behavior as the default and prove it still uses the standalone listener.

This proposal does not request template-manager changes in E4. The template-manager owner should decide whether this API shape fits the broader scenario templates after the evidence above is complete.
