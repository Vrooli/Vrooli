# Adopting Audio Tools

This is the one supported adoption guide for audio-tools. Consumers use the
API-to-API Connect contract and the shared `@vrooli/audio-capture-browser`
package; they do not copy another scenario's source tree or revive the retired
embed package.

## Prerequisites

- The scenario is generated from the `react-vite` template.
- `audio-tools` is declared in `.vrooli/service.json` with a meaningful
  `description` and `degraded_behavior`.
- The scenario API has a discovery endpoint for the audio-tools URL and uses
  generated Connect clients for the server-to-server calls.

## 1. Declare the dependency

Under `dependencies.scenarios` in `scenarios/<name>/.vrooli/service.json`:

```json
{
  "audio-tools": {
    "required": false,
    "startup_policy": "try_start",
    "description": "Speech input and audio capabilities for the scenario.",
    "degraded_behavior": "Voice controls are disabled with an actionable audio unavailable state when audio-tools is unhealthy."
  }
}
```

Use `required: true` only when the scenario cannot operate without the
dependency. The dependency analyzer's integration-conformance check validates
that the declaration, capability registry, and degraded behavior agree.

## 2. Wire the API adapter

Create a scenario-owned adapter under
`scenarios/<name>/api/integrations/audiotools/`:

- `discovery.go` resolves the current audio-tools URL through
  `api-core/discovery.ResolveScenarioURLDefault("audio-tools")`.
- `client.go` owns the generated Connect clients and bounded retry/re-discovery
  behavior.
- `contracts.go` translates transport errors and status metadata into the
  scenario's typed API surface.

Expose a discovery endpoint that returns availability, the base URL, and an
actionable unavailable reason. The browser must call the consumer scenario's
same-origin API; it must not compose or expose a foreign scenario URL.

## 3. Install and use the shared browser package

Install through the Scenario Dependency Analyzer:

```bash
scenario-dependency-analyzer deps install npm/@vrooli/audio-capture-browser \
  --scenario <name> --surface ui --apply
```

The package owns the implementation that must remain consistent for every
adopter:

- `useVoiceCore` and `PcmVoiceStreamProvider` for live capture;
- the turn journal, frame protocol, session identity, and replay dispatch;
- PCM conversion and bounded stream diagnostics.

The adopter supplies only transport functions, API/proto mapping, and themed
presentation. The microphone surface comes from the released
`react-component-library` `VoiceInputButton` component.

Example adapter:

```tsx
import { useVoiceCore } from "@vrooli/audio-capture-browser";
import { useMyStore } from "../stores/useMyStore";
import { createAudioTransport } from "../audio/transport";

export function useVoiceInput(onTranscript: (text: string) => void) {
  const settings = useMyStore((state) => state.voice);
  return useVoiceCore({
    ...settings,
    transport: createAudioTransport(),
    onTranscript,
  });
}
```

Do not implement a private PCM provider, reconnect loop, turn journal, or
microphone button in the adopting scenario.

## 4. Handle degraded behavior

Render the typed unavailable reason from the consumer API. The UI must keep the
turn journal recoverable when the transport or microphone source fails, and it
must surface whether recovery is reconnecting, re-acquiring the microphone, or
falling back to retained-audio HTTP transcription.

## 5. Validate the adoption

Run the scenario's UI/API suites and the dependency phase:

```bash
vrooli scenario test <name>
test-genie execute <name> --phases dependencies
```

The dependency phase must reach `integration_conformance` L3: the declared
dependency resolves, the shared capability registry describes it, the registry
has a health checker and operator action, and degraded behavior is declared.

## Updating

Upgrade the shared package through the governed dependency workflow, then run
the adopter's type-check, UI tests, and scenario suite. If a new adopter needs
behavior not represented by the package seam, extend the shared package and its
tests; do not fork the integration directory.

## Cross-references

- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
- [`../../../../packages/audio-capture-browser/`](../../../../packages/audio-capture-browser/)
