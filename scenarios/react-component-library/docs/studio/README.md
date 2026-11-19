# RCL design studio

The design studio is the React Component Library's authoring surface: compose a page from
RCL primitives and custom layers, inspect it at desktop/tablet/phone, measure it, and place real
components into it.

## Start here

| Document | Purpose |
|---|---|
| [`PLAN.md`](./PLAN.md) | Outcome, chosen information architecture, object model, phases, progress log |
| [`DEBT.md`](./DEBT.md) | Debt inventory with file:line, owner and effort; "do not fix now" list |
| [`HANDOFF-api.md`](./HANDOFF-api.md) | Designs + assets + brand contract, migrations, conflict envelope |
| [`HANDOFF-studio.md`](./HANDOFF-studio.md) | Studio shell, panes, command stack, autosave |
| [`HANDOFF-context.md`](./HANDOFF-context.md) | In-context frames, typed slots, placement, instruments, a11y |
| [`HANDOFF-lint.md`](./HANDOFF-lint.md) | How the ESLint backlog was cleared and the patterns to keep |
| [`HANDOFF-ai.md`](./HANDOFF-ai.md) | AI patch protocol, gateway schema subset, validation, apply/undo |
| [`HANDOFF-region-capture.md`](./HANDOFF-region-capture.md) | Crop-from-image flow and the `captured` asset path |
| [`HANDOFF-templates.md`](./HANDOFF-templates.md) | Template shelf API, one-time migration, offline queue |
| [`HANDOFF-sync.md`](./HANDOFF-sync.md) | Cross-tab events and the reconciliation rules |
| [`HANDOFF-html-import.md`](./HANDOFF-html-import.md) | HTML → custom layers, sanitisation, regions, caps |
| [`HANDOFF-extract.md`](./HANDOFF-extract.md) | Repeated-subtree signatures and the extraction candidates |
| [`HANDOFF-options.md`](./HANDOFF-options.md) | Design-owned radius/display: column, validation, model |
| [`evidence/`](./evidence/) | Served screenshots at 1440 and 390 |

## The shape

```
Design ── Frame ── Layer (TemplateNode) ── Theme (TemplatePalette)
   │         │
   │         └── device: desktop | tablet | phone, spec: TemplateNode
   ├── scenario/page binding (optional)
   ├── options (radius, display), design-owned
   ├── assets (reference | captured), stored server-side
   └── proposals (AI patch, inert until applied)
```

- **Layers** are the existing `templates/spec.ts` tree; the studio and the in-context view render
  the same spec, so a design is always previewable.
- **Slots** are typed (`role`, `accepts`) and may carry authored fallback content, so a design looks
  finished before a live component is placed. `TemplateRenderer` fills exactly one slot.
- **Provenance** is two numbers: layer share and painted-area share (`studio/rcl.ts`,
  `studio/checks.ts`). Images are always custom.
- **Checks** never pass without a measurement; unmeasured instruments say so.
- **AI** proposes a bounded patch through `ai-gateway`; the studio validates it against the live
  document and applies it as one undoable command. Discarding costs nothing.
- **Templates** are authoring drafts stored at `/api/templates` (with a browser cache and an honest
  offline queue). Selecting one does not apply it; the Apply button does.
- **HTML import** parses a detached document, strips scripts/styles/attributes and maps the structure
  to custom layers, with a region picker. Styles and remote assets are not imported; the dialog says
  so. The resulting RCL share is real, not a claim.
- **Extraction candidates** rank repeated custom shapes in the frame by duplication cost. It is
  deterministic structure, not a guess, and RCL primitives/slots/pages are excluded.
- **Design chrome** (corner radius + display face) lives in `DesignDocument.options`, so two designs
  can share a palette and still look unlike each other.
- **Two tabs** on the same design stay honest: an idle tab reloads a remote save, a tab with local
  work gets the conflict banner, and a browser-only write reads as **Local draft**, never as saved.

## Running

```bash
cd scenarios/react-component-library
vrooli scenario start react-component-library      # API 17193, UI 23906
cd ui && pnpm build && vrooli scenario restart react-component-library   # after UI changes
cd ui && pnpm dev                                  # Vite with the scenario's API proxy
```

Gates: `cd ui && pnpm test && pnpm lint && pnpm exec tsc --noEmit` and
`cd api && go test ./...`. `pnpm build` runs the catalog check, token check and string codegen.
Current: **123 files / 750 tests**, lint 0, tsc clean, build green; Go retains only the three
pre-existing library/fixture failures.

AI (optional): `vrooli scenario start ai-gateway`; without it the panel says the gateway is
unreachable and the rest of the studio keeps working.

## Strings

`en.json` is generated from the strings the UI actually renders:

```bash
cd ui
pnpm strings:catalog        # rebuild en.json from t(key, { defaultValue }) usage
pnpm strings:gen            # regenerate the typed registry
```

Other locales are incremental and fall back to English; the parity test only requires their keys
to exist in `en.json`. Never hand-edit `strings.generated.ts`.

## Screens

- `/design` — the studio (unbound design or the last one opened).
- `/design/:scenario/:page` — opens or creates the design bound to that page.
- `/assets/:id?tab=preview` → **In context** — component placed into a real page slot.
- Studio panes: Layers / System / Assets · Inspect / Checks / AI · templates and page binding in the
  Layers footer. The `+` next to the design tabs imports an HTML page. The System pane owns design
  chrome and extraction candidates.
