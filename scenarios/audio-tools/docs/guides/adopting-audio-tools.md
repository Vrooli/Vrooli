# Adopting Audio Tools

This guide explains how a new Vrooli scenario adopts `audio-tools` for voice input (STT), text-to-speech (TTS), and long-text summarization. It is the canonical adoption procedure as of 2026-05-17.

The contract is **API↔API only**:
- Your scenario declares `audio-tools` as a runtime dependency in `.vrooli/service.json`.
- Your scenario's API calls `audio-tools` through `packages/api-core/discovery` + generated Connect-RPC clients in `packages/proto/gen/go/audio-tools/v1/...`.
- Your scenario's UI consumes a **local copy** of the `audio-integration/` folder (copy-paste; no cross-scenario imports). The canonical source of that folder lives at `scenarios/audio-tools/ui/src/audio-integration/`.

There is no shared UI library, no `@audio-tools/embed` package, and no window globals.

## Prerequisites

- Your scenario is generated from the `react-vite` template and ships a Go API + React/Vite UI + Go CLI.
- `audio-tools` is reachable from your scenario host (default: the local Vrooli lifecycle resolves it via `vrooli scenario start audio-tools`).
- Your scenario already has the standard `proto + Connect-RPC` substrate wired (see `prompt-manager skill read api-steer interoperability-steer`).

## Step 1 — Declare the dependency

Edit `scenarios/<your-scenario>/.vrooli/service.json`. Under `dependencies.scenarios`, add:

```json
{
  "dependencies": {
    "scenarios": {
      "audio-tools": {
        "required": true,
        "startup_policy": "try_start",
        "description": "Audio capabilities (STT/TTS/summarization). All audio flows through audio-tools.",
        "degraded_behavior": "Required at startup — adoption scenario fails fast if audio-tools is not reachable on boot."
      }
    }
  }
}
```

Use `"required": false` if your scenario should degrade gracefully when audio-tools is down (voice UI disabled, banner shown). Use `"required": true` only if your scenario cannot function without audio-tools.

## Step 2 — Wire the API-side integration adapter

Mirror the shape `web-console` uses. Create `scenarios/<your-scenario>/api/integrations/audiotools/`:

| File | Responsibility |
|---|---|
| `discovery.go` | Wraps `api-core/discovery.ResolveScenarioURLDefault("audio-tools")`. Returns the current base URL. |
| `client.go` | Holds the four generated Connect clients (`STTServiceClient`, `STTAdminServiceClient`, `TTSServiceClient`, `SummarizeServiceClient`, `AudioProcessingServiceClient`). Built lazily on first call; re-resolved on connection-refused-class failures. Bounded retry policy (default 3 retries, 30s per-call timeout). |
| `contracts.go` | Envelope/status normalization, error translation. Centralised so the rest of your API doesn't reach for status strings. |

Reference: `scenarios/web-console/api/integrations/audiotools/` is the canonical pattern. Copy it and rename the package.

Then expose a discovery endpoint on your scenario's API that the UI can call at boot:

```go
// In handlers/discovery/audio_tools.go (or your scenario's equivalent)
// Returns: { available, baseUrl, wsBaseUrl, unavailableReason }
```

Reference: `scenarios/web-console/handlers/discovery/audio_tools.go`.

## Step 3 — Copy `audio-integration/` into your UI

```bash
cp -r scenarios/audio-tools/ui/src/audio-integration scenarios/<your-scenario>/ui/src/audio-integration
```

Add the package dependencies (already in `audio-tools/ui`'s `package.json`):

- `@bufbuild/protobuf`
- `@connectrpc/connect`
- `@connectrpc/connect-web`
- `@vrooli/proto-types` (the workspace-local generated TS package)

What that folder contains:

- `client.tsx` — `AudioToolsProvider`, `useAudioToolsClient`, `createAudioToolsClient({ baseUrl })`. **Required** `baseUrl`; no fallback.
- `api/{voice,tts,protomap}.ts` — typed Connect-client wrappers for the audio-tools API.
- `hooks/voice/*` — STT providers (`VoiceStreamProvider`, `WhisperProvider`, `WebSpeechProvider`), VAD, wake-word, audio utilities.
- `hooks/tts/*` — TTS providers (`KokoroProvider`, `BrowserTTSProvider`), shared TTS types.
- `hooks/useVoiceCore.ts` — generic voice-input orchestrator (state machine, provider lifecycle, VAD-driven segmentation). Accepts opts; no store/capability/command-parser reads.
- `hooks/useTextToSpeechCore.ts` — generic TTS orchestrator (provider selection, autoplay queue, playback state). Accepts opts.
- `MicReadinessIndicator.tsx` — small generic UI component for mic permission state.

**Do not modify files in `audio-integration/` to suit your scenario.** Keep them byte-identical to the canonical source so updates can be re-copied without merge conflicts. Scenario-specific behavior belongs in adapters you write *outside* this folder.

## Step 4 — Bootstrap the provider

In your UI's `main.tsx`, fetch discovery, build the client, mount the provider:

```tsx
import { AudioToolsProvider, createAudioToolsClient } from "./audio-integration";
import { fetchAudioToolsDiscovery } from "./api/discovery"; // your scenario's discovery client

async function bootstrap(): Promise<{ baseUrl: string; unavailableReason: string }> {
  try {
    const ep = await fetchAudioToolsDiscovery();
    if (ep.available && ep.baseUrl) {
      return { baseUrl: ep.baseUrl, unavailableReason: "" };
    }
    return { baseUrl: "http://localhost:0", unavailableReason: ep.unavailableReason || "discovery_failed" };
  } catch {
    return { baseUrl: "http://localhost:0", unavailableReason: "discovery_failed" };
  }
}

void bootstrap().then(({ baseUrl, unavailableReason }) => {
  const client = createAudioToolsClient({ baseUrl });
  ReactDOM.createRoot(document.getElementById("root")!).render(
    <AudioToolsProvider client={client} unavailableReason={unavailableReason}>
      <App />
    </AudioToolsProvider>
  );
});
```

`unavailableReason` is read by `useAudioToolsUnavailableReason()` inside the provider context. Render your scenario's "audio unavailable" banner from that hook.

## Step 5 — Use the core hooks in your scenario

Write thin scenario adapters that pull your local state (store, settings, command vocabulary) and pass it to the generic core hooks:

```tsx
// scenarios/<your-scenario>/ui/src/hooks/useVoiceInput.ts
import { useVoiceCore } from "../audio-integration";
import { useMyStore } from "../stores/useMyStore";

export function useVoiceInput(onTranscript: (text: string) => void) {
  const voiceEnabled = useMyStore((s) => s.voiceEnabled);
  const voiceLanguage = useMyStore((s) => s.voiceLanguage);
  // …
  return useVoiceCore({
    voiceEnabled,
    voiceLanguage,
    vadSilenceTimeoutMs: 1500,
    persistentMode: false,
    wakeWordEnabled: false,
    segmentSilenceMs: 700,
    lowLatencyVoice: false,
    onTranscript,
    // Optional: pass your own capability probe and command parser.
    // capabilityCheck: async () => ({ whisperHealthy: true }),
    // parseCommand: (text) => myParser(text),
  });
}
```

Same pattern for TTS — write a `useTextToSpeech` adapter that owns your scenario's playback-event ingestion / hook-ack routing (if any) and delegates synthesis to `useTextToSpeechCore`.

If your scenario's needs are simpler than web-console's, you can call `useVoiceCore` / `useTextToSpeechCore` directly from a component without writing an adapter at all.

## Step 6 — Render UI

`audio-integration` ships only one UI component: `MicReadinessIndicator`. Everything else (mic button, audio player bar, settings panels, rejection banners, command suggestions) is your scenario's choice — the core hooks expose all the state you need.

For reference implementations of the consumer-side components, look at `scenarios/web-console/ui/src/components/` (`AudioPlayerBar.tsx`, `EnableAudioBanner.tsx`, `VoiceRejectionBanner.tsx`, `VoiceCommandSuggestion.tsx`, `AudioUnavailableBanner.tsx`) and `scenarios/web-console/ui/src/components/settings/{TtsSettingsSection,VoiceInputSection}.tsx`. They are scenario-specific (terminal panes, conversation cursors, i18n strings) but show how the core hooks map onto a real product surface.

When `react-component-library` stabilizes, generic versions of those components will move there. Until then, copy or write your own.

## Updating to a new audio-integration version

When audio-tools ships a new `audio-integration/`:

```bash
rsync -av --delete \
  scenarios/audio-tools/ui/src/audio-integration/ \
  scenarios/<your-scenario>/ui/src/audio-integration/
# rerun pnpm install if dependencies changed
pnpm -C scenarios/<your-scenario>/ui install
pnpm -C scenarios/<your-scenario>/ui type-check
pnpm -C scenarios/<your-scenario>/ui test
```

If you have scenario-specific divergence inside `audio-integration/`, you've already broken the copy-paste contract. Move that divergence into your adapters and restore byte-identity.

## What never to do

- Do not import from `scenarios/audio-tools/...` directly. Cross-scenario imports are banned.
- Do not add `@audio-tools/embed` or any shared package; that approach was retired 2026-05-17.
- Do not read `window.__AUDIO_TOOLS_URL__` or `window.__AUDIO_TOOLS_UNAVAILABLE_REASON__` — those globals do not exist. Use the provider context.
- Do not modify files inside your `audio-integration/` folder. Wrap, don't fork.

## Cross-references

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — audio-tools system map and proto surface.
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — how audio-tools integrates with BYOK and LPBS.
- [`../reference/api-endpoints.md`](../reference/api-endpoints.md) — Connect-RPC method index.
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — testability seams including the audio-integration boundary.
- `prompt-manager skill read api-steer interoperability-steer` — the project-wide rules this guide enforces.
