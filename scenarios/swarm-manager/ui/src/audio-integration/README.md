# audio-integration

swarm-manager's audio surface — voice (STT), TTS, wake-word, and speaker
verification against swarm-manager's own `AudioAdminService` /
`AudioRuntimeService` (same-origin Connect transport; the server owns the
inter-scenario hop to audio-tools).

> **Not a verbatim copy.** The "canonical copy-paste" claim is stale (as of
> 2026-05-29). The three audio-integration copies have diverged and are no
> longer byte-for-byte compatible:
> - **swarm-manager (this folder)** tracks **web-console's** voice logic
>   (wake-word MFCC re-derivation + live-threshold SSOT, `resetServerVadState`
>   auto-stop fix, segment space-joining, passive-arm guard). Scenario-specific
>   bits: proto namespace (`swarm-manager/v1`), inline transport, and the
>   `client.tsx` context bootstrap.
> - **web-console** is the reference for voice quality but uses MediaRecorder
>   streaming and dropped `client.tsx` for its own proto-backed surface.
> - **audio-tools** (the old "canonical") diverged to a raw-PCM streaming
>   pipeline (`pcm.ts`/`pcmCapture.ts`, the audio-format substrate) and still
>   lacks the web-console voice-quality fixes.
>
> Before copying any file across scenarios, diff first. A reconciliation of
> the three copies is tracked as a swarm-manager capture (2026-05-29).

## Adoption

1. Declare a dependency on the `audio-tools` scenario in your
   `.vrooli/service.json`.
2. Add a discovery endpoint to your API that returns audio-tools' base URL
   (use `api-core/discovery.ResolveScenarioURLDefault`).
3. Copy this folder verbatim into your UI: `<scenario>/ui/src/audio-integration/`.
4. At UI bootstrap, fetch the discovery endpoint, build a client, and mount
   the provider:

   ```tsx
   const client = createAudioToolsClient({ baseUrl });
   <AudioToolsProvider client={client} unavailableReason={reasonOrUndefined}>
     <App />
   </AudioToolsProvider>
   ```

5. Inside React components, use `useAudioToolsClient()` and the exported
   hooks/providers (e.g. `VoiceStreamProvider`, `KokoroProvider`).

## Rules

- **No cross-scenario imports.** This folder is duplicated, not shared.
- **No window globals.** All wiring flows through React context.
- **No no-arg `createAudioToolsClient()`.** The `baseUrl` is required.
