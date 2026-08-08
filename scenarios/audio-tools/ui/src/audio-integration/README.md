# audio-integration

This directory is a scenario-local compatibility surface for audio-tools.
It is currently triplicated across audio-tools, web-console, and swarm-manager;
it is not a shared implementation. A comparison on 2026-08-02 found no files
with matching contents among the 34 files shared with web-console.

New adopters must follow
[`docs/guides/adopting-audio-tools.md`](../../../docs/guides/adopting-audio-tools.md)
and consume the shared `@vrooli/audio-capture-browser` package. The shared
package extraction was completed in Phase 4 of the audio-tools reliability
plan.

Scenario-specific transport and API adapters remain outside the shared browser
package. Do not copy this directory into another scenario or treat it as an
implementation to copy.
