# audio-integration

Web-console's scenario-local audio boundary talks only to web-console's own
same-origin API. The API owns the inter-scenario hop to audio-tools, so the
browser never discovers or calls audio-tools directly.

This directory is not a copy-paste adoption surface. New adopters must follow
[`scenarios/audio-tools/docs/guides/adopting-audio-tools.md`](../../../../audio-tools/docs/guides/adopting-audio-tools.md)
and use the shared `@vrooli/audio-capture-browser` package with a scenario
transport adapter.

The remaining files here are web-console's themed API/proto adapters and
compatibility seams during the shared-package migration. Do not duplicate this
directory into another scenario.
