/**
 * @vrooliComponentSource react-component-library:VoiceInputButton
 * @vrooliComponentVersion 4.1.0
 * @vrooliComponentAdoption 1636c81c-83d8-4a90-854d-990050b400b0
 * @vrooliComponentAppliedAt 2026-08-18T01:12:49Z
 * @vrooliComponentSourceSha256 569a2880e958577c52b6555aeda44bfd6c67a7bd380137fcd5832e78338f055e
 * @vrooliComponentDriftHash 569a2880e958577c52b6555aeda44bfd6c67a7bd380137fcd5832e78338f055e
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
export const voiceInputButtonStyles = `
[data-rcl-voice-input] { --rcl-voice-accent: var(--color-foreground); --rcl-voice-surface: var(--color-surface); --rcl-voice-border: var(--color-border); background: var(--rcl-voice-surface); border-color: var(--rcl-voice-border); color: var(--rcl-voice-accent); overflow: hidden; }
[data-rcl-voice-input][data-state="preparing"] { --rcl-voice-accent: var(--color-warning); --rcl-voice-surface: color-mix(in srgb, var(--color-warning) 10%, var(--color-surface)); --rcl-voice-border: color-mix(in srgb, var(--color-warning) 50%, var(--color-border)); }
[data-rcl-voice-input][data-state="recording"], [data-rcl-voice-input][data-state="recovering"] { --rcl-voice-accent: var(--color-info); --rcl-voice-surface: color-mix(in srgb, var(--color-info) 16%, var(--color-surface)); --rcl-voice-border: var(--color-info); }
[data-rcl-voice-input][data-state="recording"][data-mode="timeout"] { --rcl-voice-accent: var(--color-danger); --rcl-voice-surface: color-mix(in srgb, var(--color-danger) 14%, var(--color-surface)); --rcl-voice-border: var(--color-danger); }
[data-rcl-voice-input][data-state="transcribing"] { --rcl-voice-accent: var(--color-primary); --rcl-voice-surface: color-mix(in srgb, var(--color-primary) 14%, var(--color-surface)); --rcl-voice-border: var(--color-primary); }
[data-rcl-voice-input][data-state="unavailable"], [data-rcl-voice-input][data-state="error"] { --rcl-voice-accent: var(--color-warning); --rcl-voice-surface: color-mix(in srgb, var(--color-warning) 10%, var(--color-surface)); --rcl-voice-border: var(--color-warning); }
[data-rcl-voice-glyph] { position: relative; z-index: 1; inline-size: var(--space-sm); block-size: var(--space-sm); flex: 0 0 auto; }
[data-rcl-voice-input][data-size="xs"] [data-rcl-voice-glyph] { inline-size: var(--space-2xs); block-size: var(--space-2xs); }
[data-rcl-voice-input][data-size="md"] [data-rcl-voice-glyph], [data-rcl-voice-input][data-size="lg"] [data-rcl-voice-glyph], [data-rcl-voice-input][data-size="xl"] [data-rcl-voice-glyph] { inline-size: var(--space-md); block-size: var(--space-md); }
[data-rcl-voice-input][data-state="preparing"] [data-rcl-voice-glyph], [data-rcl-voice-input][data-state="recovering"] [data-rcl-voice-glyph] { animation: rcl-voice-pulse var(--dur-slow) var(--ease-standard) infinite; }
[data-rcl-voice-input][data-state="transcribing"] [data-rcl-voice-glyph] { animation: rcl-voice-spin var(--dur-moderate) linear infinite; }
[data-rcl-voice-level] { position: absolute; inset-inline: 0; inset-block-end: 0; z-index: 0; border-radius: inherit; background: color-mix(in srgb, var(--rcl-voice-accent) 58%, transparent); pointer-events: none; transition: block-size var(--dur-quick) var(--ease-standard); }
[data-rcl-voice-timeout-ring] { position: absolute; inset: 0; z-index: 1; inline-size: calc(100% - (var(--space-2xs) * 2)); block-size: calc(100% - (var(--space-2xs) * 2)); margin: auto; overflow: visible; transform: rotate(-90deg); pointer-events: none; }
[data-rcl-voice-timeout-ring] circle { opacity: .82; }
[data-rcl-voice-visually-hidden] { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
@keyframes rcl-voice-pulse { 50% { opacity: .58; transform: scale(.92); } }
@keyframes rcl-voice-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { [data-rcl-voice-input] [data-rcl-voice-glyph], [data-rcl-voice-level] { animation: none; transition: none; } }
@media (forced-colors: active) { [data-rcl-voice-input] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-voice-level] { background: Highlight; } [data-rcl-voice-timeout-ring] { color: Highlight; } }
`;
