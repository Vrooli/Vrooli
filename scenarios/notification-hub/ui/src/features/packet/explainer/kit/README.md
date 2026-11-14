# Explainer starter kit

A small stylesheet and script that the Hub puts into every HTML explainer, so
a page that writes plain HTML already looks like the Hub on a phone, in light
and dark. The contract is in `docs/concepts/DECISION_PACKETS.md` ("HTML
explainers"). Authors read `skills/notification-hub/references/html-explainers.md`.

| File | Role |
|---|---|
| `kit.css` | Hub colors, type and spacing as `--hub-*` custom properties (light and dark), base styles, and the building blocks `.hub-pair`, `.hub-compare`, `.hub-timeline`, `.hub-tradeoffs`, `.hub-callout`, `.hub-figure`, `.hub-chip`. |
| `kit.js` | Behaviour for `[data-hub-compare]`: an accessible range input plus pointer drag. Plain ES2019, no dependencies, safe to run twice. |
| `kit.test.ts` | Vitest and jsdom tests for `kit.js`, and checks on the `kit.css` contract. |

## What the Hub injects

Before it renders an explainer, the sealing step puts these at the start of
`<head>`, in this order:

1. The Content Security Policy `<meta>`.
2. `kit.css` in a `<style>` element.
3. The bridge script (`hub.open`, `hub.prefill`, `hub.theme`, height reporting).
4. `kit.js`.

Import the files as text (`import css from "./kit.css?raw"`). Do not parse or
rewrite them. `kit.js` does not implement the bridge, and does not need it.

## How the layer works

The whole stylesheet sits in `@layer hub-kit { ... }`. A style that is not in a
layer always beats a style in a layer, so any rule the page writes wins without
`!important` or a longer selector. A page can override a token
(`:root { --hub-primary: #7c3aed; }`) or a block (`.hub-callout { ... }`) and
leave the rest alone.

Colors are plain hex values: light first, then a dark set. The Hub sets
`data-theme="light"` or `"dark"` on the page root, and that attribute decides.
Without it the page follows the device (`prefers-color-scheme`). A device in
dark mode never overrides `data-theme="light"`. The kit uses no color function
that an older iPhone browser lacks for text or backgrounds; `color-mix()` only
tints a few accents, and each use has a plain value before it.

## Opt out

Put `<meta name="hub-kit" content="off">` in the page. The Hub then injects
neither `kit.css` nor `kit.js`. The CSP and the bridge stay. The page owns all
its styles, and `[data-hub-compare]` does nothing.

## Changing the kit

- Keep `kit.css` small: every explainer page carries it, and comments do not count
  toward its size. Tokens come from `ui/src/design-tokens.css`; update the values
  here when those change.
- Keep every rule inside the layer. `kit.test.ts` fails otherwise.
- Authored HTML must stay free of form controls, because the packet check flags
  them (`explainer_form`). `kit.js` therefore creates the compare range input at
  runtime.
- Check a change on a 390 px wide screen in both themes. The worked examples in
  `skills/notification-hub/examples/` use every block; open them with the kit
  injected.
