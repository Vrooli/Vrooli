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

P0 skeleton during Phase F. Full implementations port from
`scenarios/web-console/ui/src/{hooks,components}/{voice,tts,...}` and
generalize each component. See
[`docs/internal/EXTRACTION-SOURCES.md`](../../../docs/internal/EXTRACTION-SOURCES.md)
for the classification of every source file (reusable vs consumer-glue).

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
