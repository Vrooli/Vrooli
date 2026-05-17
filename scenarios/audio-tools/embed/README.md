# `@audio-tools/embed`

Adoptable React component surface for the `scenarios/audio-tools` scenario.

Consumers import from this package and never reach into audio-tools'
internal feature folders:

```tsx
import {
  VoiceInputButton,
  AudioPlayerBar,
  VoiceSettingsPanel,
  TtsSettingsPanel,
} from "@audio-tools/embed";
```

The components are intentionally web-console-agnostic — they accept
generic callback props (`onTranscript`, `commandHandler`,
`audioUrl | audioBytes`, …) and never reference consumer-specific
concepts (terminal panes, conversation cursors, session IDs).

## Status

Generally available. The component surface was extracted from
`scenarios/web-console/ui/src/{hooks,components}/{voice,tts,...}` on
2026-05-16; see `docs/internal/PROGRESS.md` for the lifecycle log.

## Wiring in a consumer

```tsx
// In web-console:
//   ui/src/domains/audio/index.ts
//
//   export {
//     VoiceInputButton,
//     AudioPlayerBar,
//     // ...
//   } from "@audio-tools/embed";
```

That single re-export boundary makes the cross-scenario swap a one-line
change for consumers; existing import sites under `ui/src/...` continue
importing from `domains/audio`.
