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
- The focused pane's visible cells are read into a grid and the **ambient**
  background is selected with a **perimeter-biased** model (`ambientBackground`),
  not a flat full-buffer histogram. App chrome sits visually adjacent to the
  terminal's perimeter, so the cells nearest that perimeter decide the color:
  1. The **bottom status row(s) are excluded** so a tmux/shell status-line band
     does not hijack the chrome color.
  2. **Perimeter pass** — corner and edge-band cells are weighted (corners count
     more than plain edges); if one color holds a strong weighted majority
     (`BG_DETECT_PERIMETER_DOMINANCE`, 0.6) it wins. A large **center-only**
     content block — e.g. a coding-agent user message with its own background —
     is *not* part of the perimeter, so scrolling past it no longer flips the
     chrome while the terminal edges stay at the base background.
  3. **Full-screen fallback** — otherwise, if a single color fills the whole
     usable grid at a high share (`BG_DETECT_FULLSCREEN_DOMINANCE`, 0.75), it
     wins, so a real full-screen TUI whose new background reaches the edges
     still retints within the debounce window.
  4. Otherwise the sample is ambiguous and detection reports `null`. Thresholds
     sit above 0.5 so an even split is a deterministic `null` rather than a
     scan-order coin-flip. No color blending or averaging is performed.
- Per-cell colors resolve by mode: RGB truecolor used directly, palette indices
  against the ANSI 256 palette, and DEFAULT-mode cells against the current
  default background — which also tracks **`OSC 11`** (set-default-background)
  changes, with `OSC 111` resetting it.
- **Fallback chain:** detected ambient background → the pane's configured theme
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
tokens, `--wc-chrome-color`, `--wc-chrome-fg`, an HTML adaptive flag for the iOS
bottom edge sliver, and the `<meta name="theme-color">` tag — and **never** through
React state. Consequently the heavy `TerminalPane` / xterm subtree never
re-renders on a color change; only a style recalc occurs. Detection is
additionally debounced (~200ms), samples a bounded region, runs **only for the
focused pane**, pauses while the document is hidden, and a change-guard
collapses identical-color ticks (the common case under a busy TUI).

## Platform notes & limitations

The shared viewport hook owns drawable-frame and keyboard measurements.
Web Console projects that frame into `--wc-app-height`; `html`, `body`,
`#root`, and the workspace must all fill it. A fixed body contributes no
height to an auto-sized document element, so expanding only body/content
leaves an inconsistent root canvas around both normal chrome and portaled
drawers. Root overflow locking and frame sizing belong together. Keep safe
clearance as internal toolbar/drawer padding; tint layers supply color only.

- **Android / PWA and iOS Safari 15-18.6:** the `<meta name="theme-color">` tag
  is updated live, so browser chrome follows the terminal background.
- **iOS Safari 26.x:** WebKit no longer honors `theme-color` for tab tinting.
  It samples a fixed/sticky element at each physical viewport edge
  (`LocalFrameView::fixedContainerEdges` in WebKit): it hit-tests the middle of
  the edge, ignoring `pointer-events`, walks up from the hit element to the
  nearest `position: fixed` or `sticky` box, and accepts that box as an edge bar
  only when it is at least 90% of the viewport wide and less than its full
  height. The bar's colour is the one opaque background on the path from the hit
  element up to the bar. A second colour on that path, or a `backdrop-filter`,
  disqualifies it, and so does any box 10px or thinner. When no bar qualifies,
  the installed app draws its blurred scroll-edge effect over roughly the top
  40pt, header included.
  - **Top:** `TopSafeArea` is the bar. It is `position: sticky` (which lays out
    as before in the non-scrolling column) and its first child is the opaque
    `StatusBarFill`, so WebKit extends the strip's colour under the status bar.
    The fixed full-screen `body` cannot serve, because it is viewport-sized. A
    top sliver did not serve either: at 4px it is below the 10px floor, and
    restoring one on 2026-11-26 left the blur in place on the phone.
  - **Bottom:** a 4px fixed sliver remains while adaptive chrome is active. It
    sits **behind** app chrome (`z-index: 0`) so it never overpaints a bar that
    already reaches the edge.
  - `body` stays backed by `--wc-chrome-color` for seams and older versions.
  Safari 26.0/26.1 have reported bugs where post-paint color changes may not
  live-update the browser bars until navigation/repaint; Safari 26.2 reportedly
  improves that. In every version, the user-side Safari setting **Allow Website
  Tinting** must be enabled.
- **iOS standalone PWA (Add to Home Screen):** the status bar is page-owned only
  when the app launches standalone, which requires the **web-app-capable** metas.
  `index.html` ships both `mobile-web-app-capable` and
  `apple-mobile-web-app-capable` = `yes`; **without them iOS ignores
  `apple-mobile-web-app-status-bar-style` entirely** and draws an opaque system
  status bar that cannot show the tinted body (this was the cause of the status
  bar "not updating" in an installed build). `apple-mobile-web-app-status-bar-style`
  only accepts `default` / `black` / `black-translucent`, is read at launch (some
  reports say at install, so re-add the icon if a relaunch shows no change), and
  cannot be recolored per value. The app ships **`default`** (since 2026-11-26).
  Under `black-translucent` the page runs under the status bar, and the
  installed iOS 26 app blurred the top ~40pt over it, tab bar included. Making
  `TopSafeArea` a qualifying sticky bar did not remove that on the operator's
  iPhone, which fits the report that the translucent style switches off
  WebKit's colour extension (nesquena/hermes-webui#8365, device-tested). Under
  `default` the web view starts below the status bar (`safe-area-inset-top` is
  0, so `StatusBarFill` collapses), and iOS 26 extends the sticky top bar's
  colour under the status bar. Before iOS 26, `default` draws a plain system
  status bar that does not follow the tint.
- **Normal Safari tab:** the status strip at the top is system-drawn and not
  page-controllable in the default bottom-address-bar layout; only the bottom
  toolbar and (in top-address-bar layout) the top bar tint. Install the app to
  get full status-bar theming.
- The desktop sidebar and Messages view use the derived token palette; per-group
  color dots, pane header accents, and semantic warning/error state colors stay
  stable so the list remains structurally distinct over the tint.
- **Perimeter-biased detection is intentional, not exact.** A background that
  reaches most of the terminal's edges/corners — a full-screen TUI, or a content
  block large enough to fill the perimeter — *will* retint the chrome, because
  from the app's perspective that color is genuinely ambient. Conversely, a
  background confined to the center (with base-colored edges) will not, even if
  it covers a majority of cells. This favors a stable chrome that follows the
  terminal environment over one that chases transient content cards.
