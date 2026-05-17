# audio-integration

Canonical copy-paste reference for scenarios that consume audio-tools (STT,
TTS, summarization) from a React UI.

This folder is the **source of truth**. It also exists verbatim under
`scenarios/web-console/ui/src/audio-integration/`. Other adopters copy it
byte-for-byte into `<their-scenario>/ui/src/audio-integration/`.

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
