# Explainer starter kit

A small stylesheet and script that the Hub puts into every HTML explainer, so
a page that writes plain HTML already looks like the Hub on a phone, in light
and dark. The contract is in `docs/concepts/DECISION_PACKETS.md` ("HTML
explainers"). Authors read `skills/notification-hub/references/html-explainers.md`.

| File | Role |
|---|---|
| `kit.css` | Hub colors, type and spacing as `--hub-*` custom properties (light and dark), base styles, the building blocks `.hub-pair`, `.hub-compare`, `.hub-timeline`, `.hub-tradeoffs`, `.hub-callout`, `.hub-figure`, `.hub-chip`, and the decision blocks `.hub-proportion` (a whole split into shares, coloured by `--hub-series-1..6`), `.hub-groups` (items chosen by group), `.hub-command` (a command with where / what / undo), `.hub-recurrence` ("asked N times") and `table.hub-matrix` (options against criteria, cards on a phone). `operatordecision.Page` in `packages/api-core` writes every decision block. |
| `kit.js` | Behaviour for `[data-hub-compare]` (an accessible range input plus pointer drag), `[data-hub-groups]` (a group's switch fills the form's multi-select through `hub.prefill`) and `[data-hub-copy]` (copies the command, or selects it where the clipboard is blocked). Plain ES2019, no dependencies, safe to run twice. |
| `theme.ts` (one folder up) | Which schemes a page's own colours work in. A page that only works in one is shown in it and locked there (`data-hub-scheme`); the API's packet check and the page tools make the same call (`packages/api-core/pagescheme`, shared fixtures in `api/internal/hub/testdata/page-schemes`). |
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

The page's colours live on `html`, never on `body`. Text inherits from `html`,
so a page that sets `:root { color: ...; background: ... }` replaces the kit's
colours for all its text. A kit colour on `body` would beat that inheritance and
leave the kit's dark-mode text (near white) on the page's own white cards: the
unreadable page of November 2026. The API holds a byte copy of `kit.css`
(`api/internal/hub/explainer_kit.css`) for its readability check; copy the file
over it after every change (`TestExplainerKitMirrorsTheUIKit` fails otherwise).

## The gutter

The Hub's frame runs the page the full width of the phone, so `body` carries the
page's one gutter: `0.75rem 1rem 1rem`, and `1rem` is the same inset as the
decision text above it. The browser's default 8px margin is zeroed by the Hub in
its own layer (`hub-base`, before `hub-kit`), kit or not. Authors are told not to
add their own body padding (`references/html-explainers.md`, "Basics"); a rule
that changes the gutter changes it for every page, so check it at 390 px.

## Opt out

Put `<meta name="hub-kit" content="off">` in the page. The Hub then injects
neither `kit.css` nor `kit.js`. The CSP and the bridge stay. The page owns all
its styles, and `[data-hub-compare]` does nothing.

## Changing the kit

- Keep `kit.css` small: every explainer page carries it, and comments do not count
  toward its size (budget 20 KiB in `kit.test.ts`). Tokens come from `ui/src/design-tokens.css`; update the values
  here when those change.
- Keep every rule inside the layer. `kit.test.ts` fails otherwise.
- Authored HTML must stay free of form controls (`form`, `input`, `select`,
  `textarea`), because the packet check flags them (`explainer_form`). `kit.js`
  therefore creates the compare range input at runtime; plain `<button>`
  elements, as the group switches and Copy use, are fine.
- Whenever a rule sets a background, it sets the text colour in the same rule
  from a token. The packet check refuses an agent's page with text below 3:1
  contrast in either theme (`page_unreadable`).
- Check a change on a 390 px wide screen in both themes. The worked examples in
  `skills/notification-hub/examples/` use every block; open them with the kit
  injected.
