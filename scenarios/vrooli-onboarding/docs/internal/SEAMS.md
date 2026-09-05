# Desktop wrapper seams

The generated Electron wrapper is a presentation host for the bundled
onboarding runtime. These module anchors describe its explicit process-boundary
seams; they do not own onboarding business decisions.

## main-process

Electron process composition and lifecycle wiring.

## splash-window-module

Startup splash window presentation while the bundled runtime becomes ready.

## window-state-module

Persisted Electron window geometry and visibility state.

## telemetry-module

Best-effort desktop telemetry delivery with no credential values.

## runtime-module

Loopback-only runtime control client and health coordination.

## storage-module

Desktop-local non-secret application metadata. Credentials are delegated to
the native credential authority and never stored here.

## ipc-module

Typed Electron main/renderer IPC boundary.

## auth-module

Bundle-local authentication token handling for the loopback runtime channel.

## bundle-module

Verified bundle path and manifest access.

## runtime-control-client

Renderer-independent client for runtime state and lifecycle operations.
