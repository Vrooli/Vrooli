# audio-integration

Swarm-manager's scenario-local audio boundary talks only to its own
same-origin `AudioAdminService` and `AudioRuntimeService`; the server owns the
inter-scenario hop to audio-tools.

This directory is scenario-local and must not be copied into another scenario.
New adopters must follow
[`scenarios/audio-tools/docs/guides/adopting-audio-tools.md`](../../../../audio-tools/docs/guides/adopting-audio-tools.md)
and use the shared `@vrooli/audio-capture-browser` package with a scenario
transport adapter.

The files that remain here are swarm-manager's themed adapters during the
shared-package migration.
