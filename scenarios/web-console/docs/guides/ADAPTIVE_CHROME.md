# Adaptive App Chrome

The web-console can tint its surrounding **chrome and content shell** — the iOS
status-bar / notch region, the top app bar, the bottom mobile toolbar, the tab
bar, the desktop sidebar, and token-driven content surfaces such as Messages —
to match the **background color actually displayed in the focused terminal
pane**. When a full-screen TUI (e.g. the Grok interactive CLI) repaints the
terminal interior, the shell follows, closing the visual seam between terminal
and app.

This is a pure-polish, client-only feature. There is no backend, proto, or CLI
surface.

## When it applies

- **Only in single-focus display modes** (`Tabs` and `Sidebar`), where exactly
  one pane is visible. In `Grid` mode multiple panes are shown and no single pane
  owns the chrome, so the default slate chrome is kept.
- Controlled by **Settings → Workspace → Adaptive chrome** (default **on**;
  persisted in the workspace store). Turn it off to keep the static slate chrome.

## How detection works

- Detection reads the **xterm buffer cell colors** (unlocked by
  `allowProposedApi: true`) — not canvas pixels. No renderer addon is added.
- The focused pane's visible rows are sampled into a histogram; the dominant
  background wins. The **bottom row is excluded** so a tmux/shell status-line
  band does not hijack the chrome color.
- Per-cell colors resolve by mode: RGB truecolor used directly, palette indices
  against the ANSI 256 palette, and DEFAULT-mode cells against the current
  default background — which also tracks **`OSC 11`** (set-default-background)
  changes, with `OSC 111` resetting it.
- **Fallback chain:** detected rendered background → the pane's configured theme
  background → the app default (slate). Detection failure never errors.

Implementation: `src/lib/terminalBackground.ts` (pure helpers),
`src/hooks/terminal/useTerminalBackgroundDetector.ts` (the detector),
`src/lib/chromePalette.ts` (contrast-safe derived palette), and
`src/lib/chromeTheme.ts` (the imperative applier).

## Palette derivation

Adaptive chrome starts from one seed color: the focused terminal's rendered
background. `deriveChromePalette(seedHex)` converts that seed through OKLCH and
produces the existing semantic `--wc-*` design tokens as RGB triples:

- surfaces: `--wc-surface-base`, `--wc-surface-raised`,
  `--wc-surface-input`, `--wc-surface-header`
- text: `--wc-text-primary`, `--wc-text-secondary`, `--wc-text-muted`,
  `--wc-text-faint`
- borders: `--wc-border-default`, `--wc-border-hover`
- accent: `--wc-accent`, `--wc-accent-fg`, `--wc-accent-border`,
  `--wc-accent-active`

Dark terminal seeds produce lighter text and subtly lighter child surfaces;
light seeds invert the polarity and produce dark text. Text tokens are iterated
until core text meets WCAG AA contrast on the surfaces where it is used
(`primary` / `secondary` at 4.5:1, `muted` / `faint` at 3:1). The brand accent
keeps a cyan hue; only its lightness/chroma and badge foreground are adjusted
for contrast.

This is intentionally token-driven. Components keep consuming semantic Tailwind
classes such as `bg-wc-surface-input`, `text-wc-text-muted`, and
`border-wc-default`; adaptive chrome changes the token values, not each
surface's class list. As a result, the mobile toolbar's input/buttons, tabs,
sidebar rows, Messages view, borders, muted text, and accent badges re-tint
coherently.

Transient overlays and state notices opt out with `.wc-stable-theme`, which
locally re-declares the default slate token values. Current stable opt-outs
include settings/appearance/AI modals, launch/confirm dialogs, context menus,
message search/file/jump/audio sheets, playback popovers, and semantic banners
for connection, recoverable sessions, audio enablement, voice rejection, and
summarization errors. Terminal interior rendering and per-pane/group identity
colors are not changed.

## Performance

The detected color changes frequently while a TUI repaints, so it is applied
**imperatively** — `chromeTheme` derives the palette and writes the `--wc-*`
tokens, `--wc-chrome-color`, `--wc-chrome-fg`, an HTML adaptive flag for iOS
edge slivers, and the `<meta name="theme-color">` tag — and **never** through
React state. Consequently the heavy `TerminalPane` / xterm subtree never
re-renders on a color change; only a style recalc occurs. Detection is
additionally debounced (~200ms), samples a bounded region, runs **only for the
focused pane**, pauses while the document is hidden, and a change-guard
collapses identical-color ticks (the common case under a busy TUI).

## Platform notes & limitations

- **Android / PWA and iOS Safari 15-18.6:** the `<meta name="theme-color">` tag
  is updated live, so browser chrome follows the terminal background.
- **iOS Safari 26.x:** WebKit no longer honors `theme-color` for tab tinting.
  It samples the page background or a fixed/sticky element near the physical
  viewport edge. The app therefore also keeps `body` backed by
  `--wc-chrome-color` and renders top/bottom 4px fixed edge slivers while
  adaptive chrome is active. This satisfies Safari 26's geometry-based sampler.
  Safari 26.0/26.1 have reported bugs where post-paint color changes may not
  live-update the browser bars until navigation/repaint; Safari 26.2 reportedly
  improves that. In every version, the user-side Safari setting **Allow Website
  Tinting** must be enabled.
- **iOS standalone PWA:** `apple-mobile-web-app-status-bar-style` only accepts
  `default` / `black` / `black-translucent` and is read at launch — it cannot be
  recolored per value. With `black-translucent` the status bar overlays the page
  and shows our tinted `body` background, **but its text/icons are forced
  light**. This works for the common dark-terminal case; a very *light* terminal
  background can make the iOS status-bar text hard to read. The contrast-aware
  `--wc-chrome-fg` keeps in-app chrome text legible, but it cannot override the
  OS-forced light status-bar glyphs on iOS.
- The desktop sidebar and Messages view use the derived token palette; per-group
  color dots, pane header accents, and semantic warning/error state colors stay
  stable so the list remains structurally distinct over the tint.
