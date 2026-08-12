# audio-integration

Web-console's scenario-local audio boundary talks only to web-console's own
same-origin API. The API owns the inter-scenario hop to audio-tools, so the
browser never discovers or calls audio-tools directly.

This directory is not a copy-paste adoption surface. New adopters must follow
[`scenarios/audio-tools/docs/guides/adopting-audio-tools.md`](../../../../audio-tools/docs/guides/adopting-audio-tools.md)
and use the shared `@vrooli/audio-capture-browser` package with a scenario
transport adapter.

The shipped model is deliberately split: `@vrooli/audio-capture-browser`
owns the protocol, journal, capture lifecycle, state machine, provider
contract, and shared UI-facing types. This directory contains only
web-console's same-origin API/proto adapters, health probes, and presentation
bindings. A local file must either re-export the package or document a genuine
host difference with `HOST DIFFERENCE`; do not duplicate this directory into
another scenario.
