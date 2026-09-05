# audio-integration

Swarm-manager's scenario-local audio boundary talks only to its own
same-origin `AudioAdminService` and `AudioRuntimeService`; the server owns the
inter-scenario hop to audio-tools.

This directory is scenario-local and must not be copied into another scenario.
New adopters must follow
[`scenarios/audio-tools/docs/guides/adopting-audio-tools.md`](../../../../audio-tools/docs/guides/adopting-audio-tools.md)
and use the shared `@vrooli/audio-capture-browser` package with a scenario
transport adapter.

The shipped model is deliberately split: `@vrooli/audio-capture-browser`
owns the protocol, journal, capture lifecycle, state machine, provider
contract, and shared UI-facing types. This directory contains only
swarm-manager's same-origin API/proto adapters, configuration context, and
presentation bindings. A local file must either re-export the package or
document a genuine host difference with `HOST DIFFERENCE`. The boundary test
rejects unmarked divergent files.
