# Handoff — HTML import

Status: **UI complete; page wiring is present; string-catalog sync pending.**
`DesignStudioPage` already imports `HtmlImport`, keeps `htmlImportOpen`, and mounts the dialog.
No file outside the list below was hand-edited in this slice.

## Files

| File | Change |
|---|---|
| `ui/src/workbench/studio/htmlImport.ts` | New pure `DOMParser` → `TemplateNode` mapper, region index, caps, typed errors |
| `ui/src/workbench/studio/htmlImport.test.ts` | 36 parser/mapping/sanitisation/cap tests |
| `ui/src/workbench/studio/HtmlImport.tsx` | New accessible import dialog (file picker, drag/drop, region chooser, name, action) |
| `ui/src/workbench/studio/HtmlImport.a11y.test.tsx` | 4 dialog tests, including two axe passes |
| `ui/src/workbench/studio/html-import.css` | New dialog styles, all `--wb-*` tokens |
| `docs/studio/HANDOFF-html-import.md` | This note |

## Product rule

An imported HTML page is **reference material, not evidence**. The import creates a new design
draft in the studio's existing layer model. It does not add a component to the RCL catalog and it
does not claim coverage the composition does not have.

RCL share is computed from the mapped layer types (`studio/rcl.ts`). A mapped `button` is the
studio's `button` layer, so the renderer uses the RCL Button and that layer legitimately counts.
Headings, paragraphs and layout map to `text`/`stack`/`row` and stay custom. The dialog says:

> Imported as a new design draft. Source HTML is reference material; styles and remote assets are
> not imported. RCL share reflects the layers this import maps to.

Nothing is fetched: no remote images, scripts, stylesheets, fonts, media or iframes. The parsed
document is detached (`DOMParser`) and is never inserted into the live DOM.

## Mapping

| Source | Layer | Props |
|---|---|---|
| `h1`, `h2` | `text` | `variant: "display"` |
| `h3`–`h6` | `text` | `variant: "title"` |
| `p`, `span`, `label`, plain text | `text` | `variant: "body"` |
| `small` | `text` | `variant: "body"`, `tone: "muted"` |
| `a` | `text` | `variant: "body"`, `tone: "accent"` |
| `button` | `button` | `variant: "secondary"` |
| `section`, `article`, `header`, `footer`, `main`, `nav`, `div` | `stack` | `gap: 14` |
| `ul`, `ol` | `stack` | `gap: 14` |
| `li` with element children | `row` | `gap: 12` |
| `li` without element children | `text` | `variant: "body"` |
| `table`, `tbody` | `stack` | `gap: 14` |
| `tr` | `row` | `gap: 12` |
| `td`, `th` | `text` | `variant: "body"` |
| `hr` | `divider` | — |
| `br` | `spacer` | `h: 12` |
| unknown with element children | `stack` | `gap: 14` |
| unknown with only copy | `text` | `variant: "body"` |

- The chosen region (or the whole body for `null`) is wrapped in a `page` node with `pad: 40`.
- Node ids are pre-order layer indexes: `html-<n>-<type>`, plus `html-page` for the wrapper.
  They never contain source text, so renaming copy does not change ids.
- A region's layer count includes the region element itself. Region ids are `region-<source index>`
  and are stable for the same document.
- Labels come from the mapped subtree's text joined with spaces, capped at 80 characters, so a
  `<section><h2>Region</h2><p>Copy</p></section>` reads `Region Copy` rather than `RegionCopy`.

## Sanitisation rules

1. Refuse input over **2 MiB** (`reason: "size"`) or blank (`reason: "empty"`); a parser failure is
   `reason: "parse"`. All three throw `HtmlImportError`.
2. Remove `script`, `style`, `link`, `meta`, `noscript`, `template`, `iframe`, `object`, `embed`,
   `audio`, `canvas`. `form` is **unwrapped**: the form element and its attributes go, its children
   stay.
3. Remove `img`, `picture`, `svg` and `video` subtrees and report
   `N images were skipped; remote assets are not fetched.` (singular for one). Nested images are
   counted once because the outer subtree is removed first.
4. Strip **every attribute** from every remaining element in the body — `class`, `style`, `id`,
   `href`, `src`, `alt`, `aria-*` and inline handlers such as `onclick`. The mapper reads only tag,
   text and element children.
5. Collapse all whitespace (`\s+` → one space) and skip empty text nodes. Empty text leaves and
   empty containers produce no layer.

## Caps

| Cap | Value | Effect |
|---|---|---|
| Input | 2 MiB (`HTML_LIMITS.bytes`) | Typed refusal, file never read |
| Source elements scanned | 500 (`HTML_LIMITS.elements`) | Warning: only the first 500 elements were scanned |
| Content layers | 200 (`HTML_LIMITS.layers`) | Warning: only the first 200 layers were imported; the page root is extra |
| Produced layer depth | 12 (`HTML_LIMITS.depth`) | Page is depth 0; content at depth 13+ is dropped with a warning |

Truncation is always reported as a warning, never silently. The dialog shows warnings as a
labelled list (heading + text, not colour alone). `ParsedHtml.layers === 0` is the dialog's
"no importable content" state, so an image-only page cannot be imported as an empty design.

## API

```ts
export const HTML_LIMITS: { bytes: number; elements: number; layers: number; depth: number };

export class HtmlImportError extends Error {
  readonly reason: "size" | "empty" | "parse";
}

export interface HtmlRegion {
  id: string;       // region-<source index>
  tag: string;      // lower-case source tag
  label: string;    // collapsed, 80-char text snippet (may be empty)
  depth: number;    // source nesting; body children are 0
  layers: number;   // produced layers in the subtree, including the element
  children: number; // direct source element children after sanitisation
}

export interface ParsedHtml {
  title: string;              // <title>, then file name, then "Imported HTML"
  layers: number;             // content layers (page root excluded)
  regions: HtmlRegion[];      // importable regions, document order
  warnings: string[];
  regionSpec(id: string | null): TemplateNode; // null = whole body
}

export function parseHtml(source: string, fileName?: string): ParsedHtml;
export function htmlTemplate(parsed: ParsedHtml, regionId: string | null, name: string): Template;
```

`regionSpec` returns a **fresh** page-wrapped spec on each call, so callers cannot mutate the
cached parse. Unknown region ids throw `HtmlImportError("parse")`.

`htmlTemplate` uses `DEFAULT_TEMPLATE` for palette, display, radius, category and slot hint,
copies the palette, sets `local: true`, `tagline: "Imported HTML"`, and gives each import a unique
template id. A blank name falls back to `parsed.title`.

## Dialog

`HtmlImport` mounts whenever the page renders it; there is no `open` prop.

- File input accepts `.html`, `.htm`, `text/html`; drag/drop is an optional pointer path.
- Shows the file name and size. States: empty, parsing, parse error, oversized, no importable
  content, ready. Every busy state ends in a visible outcome; no spinner.
- A parsed document shows the title, `N layers · M regions`, warnings, and a **tree/list of native
  buttons**: whole document plus each region, indented by source depth, with tag, text snippet and
  layer count. `aria-pressed` marks the choice.
- "Import as a new design" calls `onImport(template, name)` and closes. The name field defaults to
  the document title and is editable.
- Accessibility: `role="dialog"`, `aria-modal`, an `<h2>` name, focus trap (`useFocusTrap`) with
  initial focus inside the dialog, Escape to close, backdrop click to close, visible focus rings,
  real `<label>` for the file input and name field, `role="alert"` for errors, and text warnings.
- All copy goes through `t("…", { defaultValue })`.

## Page integration (already present)

`DesignStudioPage.tsx` owns the wiring. The exact shape is:

```tsx
const [htmlImportOpen, setHtmlImportOpen] = useState(false);

const importHtmlDesign = (template: Template, name: string) => {
  setHtmlImportOpen(false);
  // An imported page is its own design. Leaving a page route first prevents
  // the binding effect from immediately selecting that page's design.
  if (scenario || page) void navigate(designPath());
  void studio
    .createFromTemplate(template, name, { scenario: null, page: null })
    .then(() =>
      flash(
        t("workbench.studio.htmlImported", {
          defaultValue: "Imported {{name}} as a new design.",
          name,
        }),
      ),
    )
    .catch(() =>
      flash(
        t("workbench.studio.htmlImportFailed", {
          defaultValue: "The HTML page could not be imported. Nothing was changed.",
        }),
      ),
    );
};

// StudioTopBar — the design menu button
{onImportHtml ? (
  <button
    type="button"
    className="studio-menu-row"
    onClick={() => {
      setMenuOpen(false);
      onImportHtml();
    }}
  >
    <FileUp aria-hidden size={13} />
    {t("workbench.studio.importHtmlDesign", {
      defaultValue: "Import an HTML page",
    })}
  </button>
) : null}

// DesignStudioPage — pass the opener and mount the dialog
<StudioTopBar
  // …
  onImportHtml={() => setHtmlImportOpen(true)}
/>

{htmlImportOpen ? (
  <HtmlImport onClose={() => setHtmlImportOpen(false)} onImport={importHtmlDesign} />
) : null}
```

`{ scenario: null, page: null }` matters: an imported reference page must not be bound to the
current scenario route. `createFromTemplate` seeds all three frames from the imported spec and
opens the new design.

## Deliberately not imported

- CSS, stylesheets, classes, inline styles, colours, spacing, fonts, responsive rules, layout from
  the source. The draft uses the studio's `DEFAULT_TEMPLATE` tokens.
- Scripts, event handlers, behaviours, forms as controls, links as links (`href` is stripped).
- Images, `picture`, SVG, video, audio, canvas, iframes and embedded objects. No remote request is
  made, even for a `src` that would otherwise load.
- Metadata, ids, ARIA, alt text, data attributes and the source DOM itself. The result is a layer
  tree, not a preserved HTML document, and it is not evidence of the source page's quality.
- Table structure beyond rows and text: columns, spans, captions and `thead`/`tfoot` semantics are
  flattened by the generic container rule.

## Validation

```bash
cd ui
pnpm exec vitest run src/workbench/studio/htmlImport.test.ts src/workbench/studio/HtmlImport.a11y.test.tsx
pnpm exec tsc --noEmit
pnpm exec eslint src/workbench/studio/htmlImport.ts src/workbench/studio/htmlImport.test.ts \
  src/workbench/studio/HtmlImport.tsx src/workbench/studio/HtmlImport.a11y.test.tsx
```

Results: **40/40 tests pass** (36 parser + 4 dialog, two axe passes), **`tsc` clean**,
**ESLint 0 errors** on the TypeScript files. `pnpm build`, the scenario restart and commits were
not run, per the task.

## Left for the coordinator

1. **Run the string-catalog sync** (`pnpm strings:catalog` then `pnpm strings:gen`). The dialog adds
   `workbench.studio.htmlImport*` keys and the page adds `htmlImported`/`htmlImportFailed`; until
   the sync lands, `strings:catalog:check` and `strings/codegen-fresh` can report drift.
2. The page wiring and top-bar trigger are already in place; verify the trigger opens the dialog at
   `/design` and that importing from `/design/:scenario/:page` lands an unbound design.
3. Optional visual pass at 1440 and 390: the region list is capped at 232 px and scrolls, and the
   footer stacks under 520 px.

## Coordinator integration (landed)

- The `+` next to the design tabs now has **Import an HTML page** (`StudioTopBar.onImportHtml`);
  the page mounts `HtmlImport` and calls
  `studio.createFromTemplate(template, name, { scenario: null, page: null })`. The explicit null
  binding is deliberate: an import must never overwrite the page the person is on. On a page route
  the page navigates to `/design` first so the binding effect cannot select the page's design.
- A successful import flashes "Imported <name> as a new design"; a failed create flashes that
  nothing changed.
- Post-review fixes: containers carry a source-role label with their own heading
  (`Section · Why now`, `Header`, `List`) instead of the synthetic id; the attribute test allows the
  new `label` prop. A `<button>` maps to the `button` layer and therefore counts toward RCL share;
  the dialog copy frames the source as reference material, not "0% RCL".
- Live: `import-sample.html` → 26 layers / 26 regions, one skipped image; the `MAIN` region imports
  only "Why now" onward; 390 reflows with no overflow. Evidence: `studio-html-import.png`,
  `studio-html-region.png`, `studio-imported.png`, `studio-390.png`.
