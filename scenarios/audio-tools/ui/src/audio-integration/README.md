# audio-integration

This directory is a scenario-local compatibility surface for audio-tools.
Shared capture, PCM, voice-state, and playback behavior lives in
`@vrooli/audio-capture-browser`; this directory may retain only explicit
scenario adapters and documented host differences.

New adopters must follow
[`docs/guides/adopting-audio-tools.md`](../../../docs/guides/adopting-audio-tools.md)
and consume the shared `@vrooli/audio-capture-browser` package. The shared
package extraction is the governing boundary; this directory is not a
canonical implementation to copy.

Scenario-specific transport and API adapters remain outside the shared browser
package. The UI boundary test rejects unmarked divergent files and requires
each retained host difference to carry a `HOST DIFFERENCE:` marker. Do not
copy this directory into another scenario or treat it as an implementation to
copy.
