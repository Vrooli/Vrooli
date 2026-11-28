# @vrooli/file-viewer

The shared file viewer used by Web Console and Notification Hub: one shell, one
controller, and 20 preview kinds (Markdown, code, images, SVG, PDF, audio, video,
Office documents, spreadsheets, CSV, slides, e-books, 3D models, fonts, SQLite
databases, archives, diffs, folders). An improvement here lands in every app.

The viewer is **source-agnostic**. It never talks to a server, a session or a
filesystem; it talks to a [`PreviewSource`](#the-previewsource-interface) that the
host app provides. Web Console implements one over its session-bound Connect
service. Notification Hub implements one over the attachments of a decision,
which are copies it serves itself. If the files are known up front, you do not
need to write a source at all: [`createManifestPreviewSource`](#when-the-files-are-known-up-front-manifest-source)
builds one from a list of files and URLs.

## Contents

- [The PreviewSource interface](#the-previewsource-interface)
- [When the files are known up front: manifest source](#when-the-files-are-known-up-front-manifest-source)
- [A hand-written source](#a-hand-written-source)
- [Showing a file](#showing-a-file)
- [Capabilities](#capabilities)
- [Strings, locale and host settings](#strings-locale-and-host-settings)
- [Optional engines: spreadsheets and diagrams](#optional-engines-spreadsheets-and-diagrams)
- [Markdown outside the viewer](#markdown-outside-the-viewer)
- [Consumer setup](#consumer-setup)
- [Selectors](#selectors)
- [Development](#development)

## The PreviewSource interface

This is the whole contract (source of truth: `src/source/types.ts`).

```ts
type PreviewKind =
  | "markdown" | "code" | "text" | "svg" | "image" | "pdf" | "audio" | "video" | "archive"
  | "document" | "presentation" | "ebook" | "model3d" | "font" | "database"
  | "spreadsheet" | "csv" | "diff" | "directory" | "unsupported";

interface PreviewModel {
  previewId: string;            // the source's own handle; the model is passed back to every other call
  inputPath: string;            // the path as asked for
  resolvedPath: string;         // canonical path: shown, and used for breadcrumbs and navigation
  basename: string;
  line?: number;                // 1-based line to reveal in text kinds
  kind: PreviewKind;            // chosen by the source; the viewer never sniffs
  mimeType: string;
  sizeBytes: number;
  canPreview: boolean;          // false: no renderer can show it, the viewer offers download
  canDownload: boolean;
  textContentAvailable: boolean; // readText() serves this target
  listingAvailable: boolean;     // listDirectory() serves this target (directories)
  archiveMember?: boolean;
  blobHref: string;             // URL of the bytes; fetched with credentials: "same-origin"; used by <img>, <video>, workers
  runtimeHref?: string;         // root for an HTML page's own sub-resources; omit for an inert page
  expiresMs: number;            // epoch ms after which the URLs stop working; 0 = never
}

type ResolveOutcome =
  | { outcome: "resolved"; model: PreviewModel }
  | { outcome: "ambiguous"; inputPath: string; candidates: PreviewCandidate[] }   // relative path matched several places
  | { outcome: "unresolved"; inputPath: string; searchBases: PreviewCandidate[] }; // matched nowhere; folders to browse

interface PreviewCapabilities {
  thumbnails: boolean;    // thumbnailHref(): image thumbnails in folder rows
  archives: boolean;      // listArchive() + openArchiveEntry(): archives open like folders
  htmlRuntime: boolean;   // model.runtimeHref is honoured: HTML pages load their own sub-resources
  locate: boolean;        // ambiguous/unresolved outcomes get a candidate picker / browse-to-locate banner
  parentFolders: boolean; // files have a host parent folder: "open containing folder"
}

interface PreviewSource {
  /** Identity of what this source can open (session id, ask id, ...). A different scope resets the viewer. */
  readonly scope: string;
  readonly capabilities: PreviewCapabilities;

  resolve(path: string, options?: { context?: "message_link" | "inline_code" | "cli"; signal?: AbortSignal }): Promise<ResolveOutcome>;
  readText(model: PreviewModel, signal?: AbortSignal): Promise<PreviewTextContent>;
  listDirectory(model: PreviewModel, options: {
    sort: "dirs_first_name" | "name" | "size_desc" | "mtime_desc";
    showHidden: boolean;
    pageToken?: string;
    signal?: AbortSignal;
  }): Promise<PreviewListing>;

  // Only when capabilities.archives:
  listArchive?(model: PreviewModel, signal?: AbortSignal): Promise<PreviewArchive>;
  openArchiveEntry?(source: { previewId: string; entryId: string }, signal?: AbortSignal): Promise<{ status: string; target: PreviewModel | null }>;

  // Only when capabilities.thumbnails:
  thumbnailHref?(model: PreviewModel, options?: { entry?: string; size?: number }): string | undefined;
}
```

Rules a source follows:

- **`scope` is identity, not object identity.** Creating a new source object on every
  render is harmless as long as `scope` is equal; a changed `scope` aborts in-flight
  requests, clears history and closes the viewer.
- **Declare only what you can serve.** Everything a capability gates is hidden when
  it is `false`; a source never stubs an unsupported operation. Use `NO_CAPABILITIES`
  as the base.
- **Kinds come from the source.** Classify on the server (Web Console and the Hub
  both use `packages/api-core/filekind`) so every app picks the same renderer.
- **Fail with an `Error` that has a readable `message`.** The viewer shows it on the
  error card with Retry. `AbortError` is ignored.
- **`readText`** returns `truncated: true` when it cut the content at its inline limit.
- **`listDirectory`** is paged: return a `nextPageToken` and the viewer offers "Load
  more". Sort and hidden-file options restart the listing from page one.

## When the files are known up front: manifest source

Notification Hub attachments, release bundles and test fixtures are all lists of
files with URLs. `createManifestPreviewSource` turns such a list into a complete
source: a virtual read-only tree (folders are implied by the paths), text fetched
from the URLs on demand, sorting, hidden-file filtering, folder listings and
thumbnails.

```ts
import { createManifestPreviewSource } from "@vrooli/file-viewer";

const source = createManifestPreviewSource({
  scope: `ask:${ask.id}`,
  files: ask.attachments.flatMap((attachment) =>
    attachment.entries.map((entry) => ({
      path: `${attachment.name}/${entry.path}`,          // virtual tree path
      url: `/api/v1/asks/${ask.id}/attachments/${attachment.name}?path=${encodeURIComponent(entry.path)}`,
      kind: entry.previewKind,                           // from the server's classifier
      mimeType: entry.mediaType,
      sizeBytes: entry.size,
    })),
  ),
});
```

Resolved paths are canonical absolute paths in that tree (`/plan.md`,
`/screens/home.png`), so a relative image `![](diagram.png)` inside `/docs/a.md`
resolves to `/docs/diagram.png`. `resolve("/")` opens the root folder. A path that is
not in the manifest resolves to `unresolved` with no folders to browse, which the
viewer reports as unavailable. It declares only `thumbnails` (when any file has a
`thumbnailUrl`); there are no host parent folders and no archives to open. If you
need archive browsing, wrap the manifest source and add `listArchive` /
`openArchiveEntry` with `capabilities.archives: true`.

## A hand-written source

A source over a single file the app already has a URL for needs only three members
besides `scope`. Everything else is optional:

```ts
import { NO_CAPABILITIES, type PreviewSource } from "@vrooli/file-viewer";

function singleFileSource(id: string, file: { name: string; url: string; kind: PreviewKind; mimeType: string; size: number }): PreviewSource {
  const model = (inputPath: string) => ({
    previewId: id, inputPath, resolvedPath: `/${file.name}`, basename: file.name,
    kind: file.kind, mimeType: file.mimeType, sizeBytes: file.size,
    canPreview: true, canDownload: true,
    textContentAvailable: file.kind === "markdown" || file.kind === "code" || file.kind === "text",
    listingAvailable: false, blobHref: file.url, expiresMs: 0,
  });
  return {
    scope: id,
    capabilities: NO_CAPABILITIES,
    resolve: async (path) => ({ outcome: "resolved", model: model(path) }),
    readText: async (m) => {
      const response = await fetch(file.url, { credentials: "same-origin" });
      if (!response.ok) throw new Error(`Could not read ${m.basename}`);
      return { resolvedPath: m.resolvedPath, kind: m.kind, mimeType: m.mimeType, content: await response.text(), truncated: false };
    },
    listDirectory: async () => { throw new Error("This source has no folders"); },
  };
}
```

## Showing a file

`useFilePreviewController(source)` owns the viewer's state machine: resolving,
loading text or a listing, navigation history (32 frames), paging, Back, retry, the
ambiguity picker and browse-to-locate. `FileViewer` renders its state. Every async
landing is guarded by a request id, so rapid clicks can never land in the wrong
frame.

```tsx
import { FileViewer, useFilePreviewController } from "@vrooli/file-viewer";

function AttachmentViewer({ source, path, onClose }: { source: PreviewSource; path: string; onClose: () => void }) {
  const controller = useFilePreviewController(source);
  const { openPreview, close } = controller;
  useEffect(() => { void openPreview(path); return close; }, [openPreview, close, path]);
  return <FileViewer source={source} controller={controller} onClose={() => { close(); onClose(); }} />;
}
```

`FileViewer` props worth knowing:

| Prop | Meaning |
|---|---|
| `presentation` | `"drawer"` (full-page drawer, default) or `"dock"` (a side panel; add `focusMode`, `onFocus`, `onRestore`, `onResizeStart`) |
| `renderHeaderActions` | `(context) => ReactNode`: extra buttons beside the title (Web Console's Hand off and Run live here). Context carries the model, loaded text and path. |
| `renderFooter` | `(context) => ReactNode`: a host band under the preview (an attachment strip, a "Comment here" bar). Same context as `renderHeaderActions`, so it can pin notes to the open file. In the drawer it sits above the bottom safe-area inset; it does not scroll. Return `null` for none. |
| `showPath` | `false` hides the resolved path line under the title (default `true`) |
| `onClose` | Close handler for the drawer or dock |

The drawer presentation is the component library's `FullPageDrawer`, rendered in a portal at
`z-index: var(--layer-modal)`; a host sheet that must stack above it uses a higher layer
token. The viewer stops no pointer or touch events, and React events bubble through
the portal to the host's tree, so a wrapper `div` with swipe handlers sees them (the only
draggable part is the drawer's own grabber).

Open a document at a heading with `controller.openPreview(path, context?, { anchor: "plan-1" })`.
Markdown and Word readers scroll to and focus a heading by its GitHub-style slug.
They lowercase the heading, drop punctuation, replace spaces with `-` and number
repeated headings in document order (`plan`, `plan-1`, `plan-2`).
HTML previews apply the fragment to the sandboxed document's URL so browser
anchors and the document's own routing work. A message link such as
`report.html#project-O5` supplies the fragment directly. Encode a literal
filename hash as `%23`. Back and Retry retain the requested fragment.

Also exported for hosts that compose their own chrome (Web Console's attach-from-host
picker does): `PreviewBreadcrumbs`, `DirectoryPreview`, `DirectorySelection`,
`rendererForKind`, and the `Preview*` model types.

## Capabilities

| Capability | When `false` the viewer ... |
|---|---|
| `thumbnails` | shows file icons in folder rows instead of image thumbnails |
| `archives` | treats `archive` targets as download-only |
| `htmlRuntime` | renders HTML pages inert (no sub-resource loading) even if `runtimeHref` is set |
| `locate` | turns `ambiguous` and `unresolved` outcomes into an ordinary "unavailable" error (an ambiguous match opens its first candidate) |
| `parentFolders` | omits "open containing folder" |

## Strings, locale and host settings

The package owns its strings (English, Arabic, Japanese) as typed catalog modules:
a key that does not exist is a compile error, and a locale that misses a key
fails the build. Without a provider the viewer renders English.

```tsx
import { FileViewerProvider } from "@vrooli/file-viewer";

<FileViewerProvider
  locale={i18n.resolvedLanguage}                       // "en" | "ar" | "ja"; other tags fall back to English
  overrides={{ "viewer.download": "Save a copy" }}     // per-key overrides, typed against the catalog
  translate={(key, params) => undefined}               // optional adapter, consulted first; return undefined to fall through
  readerFontSize={{ value: size, onChange: setSize }}  // optional: persist the Markdown reader's text size
  rewriteHref={(href) => null}                         // optional: rewrite link targets (Web Console tunnels localhost)
  loaders={{ spreadsheet, diagram }}                   // optional: engines for workbooks and Mermaid (see below)
  fileInfo="host"                                      // optional: "bar" (default) or "host"; see below
>
  ...
</FileViewerProvider>
```

Resolution order for a key: `translate` adapter, then `overrides`, then the locale
catalog (exact tag, then language subtag), then English. Plural keys follow the
i18next convention (`key_one`, `key_other`, ... selected by `Intl.PluralRules` from
`params.count`). The pseudo-locale `"keys"` returns keys instead of copy, which is
what the package's own tests use. Text direction follows the locale (`ar` is
right-to-left); pass `dir` to override.

`fileInfo="host"` tells the viewer that the host shows the file's size, format and download
link itself (for instance in its own `renderFooter`): the preview draws no footer bar at all,
so its "Open in new tab" and "Copy source" actions are not offered either. The default,
`"bar"`, keeps the bar. Pair it with
`showPath={false}` when the host also names the file in its own header.

The reader's text size is internal state (default 16px, range 12 to 32) unless the
host supplies `readerFontSize`.

## Optional engines: spreadsheets and diagrams

Two features need a third-party library that not every host can, or wants to, install:
spreadsheet previews (SheetJS `xlsx`, published only as a tarball URL) and Mermaid
diagrams. They are **optional peer dependencies**, and the package never imports either
by bare specifier from its main entry. The host hands each over as a loader on
`FileViewerProvider`:

```tsx
import { FileViewerProvider, type FileViewerLoaders } from "@vrooli/file-viewer";

// Define once at module level (inline objects are fine too: the context is keyed by the functions).
const loaders: FileViewerLoaders = {
  // Parses workbooks in a worker. Importing this subpath is what pulls `xlsx` into your bundle.
  spreadsheet: () => import("@vrooli/file-viewer/spreadsheet").then((m) => m.spreadsheetEngine),
  // Mermaid's default export satisfies `DiagramEngine`.
  diagram: () => import("mermaid").then((m) => m.default),
};

<FileViewerProvider loaders={loaders}>...</FileViewerProvider>
```

Without a loader the content degrades cleanly instead of failing the build or the page:
a workbook shows "Spreadsheet preview isn't available here" with its download link, and a
Mermaid block shows its source with a note. The engines load on first use, not at start-up.
`FileViewerLoaders`, `SpreadsheetEngine` and `DiagramEngine` are exported if you want to
supply your own engine (a `SpreadsheetEngine` is `parse(href, signal) => Promise<WorkbookResult>`).

Why the spreadsheet loader returns an engine and not the `xlsx` module: parsing runs in a
Web Worker so a large workbook never freezes the page, and a worker is a separate bundle
that cannot receive a function from the page. The engine owns the worker; `xlsx` is only
ever imported inside it.

**Header row.** The first row of a sheet is treated as column headers when every cell is a
distinct, non-empty plain text value (no formulas, at most 80 characters) and the rows below
it hold numbers, booleans or dates. Headers label the grid columns and the row cards
("Length (s): 15"), and that row is not repeated as data. The options dialog's "Header
row" choice overrides the guess: automatic, first row is headers, or first row is data
(`workbookHeader` in `src/preview/workbookData.ts`).

## Markdown outside the viewer

The viewer's Markdown renderer is exported for chat bodies, approvals and any other
Markdown surface, so it renders the same everywhere:

```tsx
import { MarkdownRenderer } from "@vrooli/file-viewer";

<MarkdownRenderer
  content={text}
  imageSource={source}                 // optional PreviewSource: relative images resolve through it
  imageBasePath="/docs/"               // directory relative images sit next to
  onFileReferenceClick={(path) => ...} // inline-code and fenced-path chips
  onLinkClick={(href, event) => ...}
  onMermaidOpen={(code) => ...}
  progressive                          // chunked rendering for whole-file documents
/>
```

Syntax highlighting (Shiki) loads on first use. Mermaid diagrams need `loaders.diagram` (above); without it the block shows its source.

## Consumer setup

1. **Dependency.** Declare `"@vrooli/file-viewer": "file:../../../packages/file-viewer"` in the
   scenario UI's `package.json` (shared-package adoption is explicit in the consumer's
   manifest; see `docs/package-governance.md`). The package's own dependencies (three,
   PDF.js, SQLite wasm, Shiki, ...) come with it. These are **peer dependencies the
   app must already have**: `react`, `react-dom`, `@vrooli/react-component-library` (the
   viewer's drawer, buttons and path chip). **Optional peers**: `vite` (for the plugin
   below), and `xlsx` and `mermaid` (only if you pass the matching loader; see
   [Optional engines](#optional-engines-spreadsheets-and-diagrams)).
2. **Vite** (`vite.config.ts`):
   ```ts
   import { fileViewer } from "@vrooli/file-viewer/vite";
   export default defineConfig({
     plugins: [react(), fileViewer()],
     // Workers and wasm are located relative to the package files: keep it out of pre-bundling.
     optimizeDeps: { exclude: ["@vrooli/file-viewer"] },
     // Vitest: the package's dist uses Vite-only imports (?raw, ?url), so inline it, and the
     // component library with it (otherwise Node imports its usage-contract.json without an import attribute).
     test: { server: { deps: { inline: [/@vrooli\/(file-viewer|react-component-library)/] } } },
   });
   ```
   Nothing else is needed for Vite or Vitest config files. Every subpath export also carries
   a `default` condition, because Tailwind loads `tailwind.config.ts` through jiti, which
   resolves imports with the `require` condition.

   `fileViewer()` serves and emits the PDF.js assets (cmaps, fonts, wasm, ICC profiles)
   and builds the self-contained slide runtime (`virtual:presentation-runtime-url`).
3. **Tailwind** (`tailwind.config.ts`):
   ```ts
   import fileViewerPreset from "@vrooli/file-viewer/tailwind-preset";
   export default {
     presets: [fileViewerPreset],
     content: ["./src/**/*.{ts,tsx}", "./node_modules/@vrooli/file-viewer/dist/**/*.js"],
   };
   ```
4. **CSS.** Import `@vrooli/file-viewer/styles.css` once. It defines default values for
   every `--fv-*` variable (RGB channel triplets, like `--fv-accent: 135 183 215`, in
   `:where(:root)` so any declaration of yours wins) and the viewer's component styles.
   Map them to your own tokens to match your theme, where your tokens are declared:
   `:root { --fv-accent: var(--app-accent); }`. The variables are: `--fv-surface-base`,
   `-surface-raised`, `-surface-input`, `-text-primary`, `-text-secondary`, `-text-muted`,
   `-text-faint`, `-border-default` and `-border-hover` (these two carry their alpha:
   `255 255 255 / 0.1`), `-accent`, `-accent-fg`, `-accent-border`, `-error-text`,
   `-error-detail`, `-success`, and `--fv-z-chrome` (a layer number).
5. **Provider** (optional for English previews): `FileViewerProvider` configures locale, text overrides, link rewriting, the reader text size and engine loaders. Mount it once above everything that renders the viewer or its Markdown; it does not need to wrap the whole app, only those surfaces. Without a provider the viewer renders English with no spreadsheet or diagram engine.

**Testing a host that mounts the viewer.** The viewer's components inject the component
library's base stylesheet, which uses CSS cascade layers (`@layer`). jsdom 25 cannot parse
`@layer` at all and logs `Could not parse CSS stylesheet` on the first mount (a suite
that fails on unexpected console output then fails); jsdom 27 parses it, and is what this
package and Web Console test with. Use jsdom 27 or newer. Inline both
`@vrooli/file-viewer` and `@vrooli/react-component-library` in Vitest
(`server.deps.inline: [/@vrooli\/(file-viewer|react-component-library)/]`): inlining only the
viewer makes Node import the library's `usage-contract.json` without an import attribute. jsdom also lacks `matchMedia`,
which the component library's hooks call: install a stub that answers "does not match"
(see `src/test-setup.ts`). Neither is a viewer or library defect, and browsers need
neither.

Subpath exports: `@vrooli/file-viewer` (everything above), `/selectors`, `/paths`
(`basename`, `dirname`, `joinPath`, `pathCrumbs`), `/file-references` (path and link
detection in prose: `looksLikeFileReference`, `matchProseFilePaths`, ...), `/styles.css`,
`/spreadsheet` (the worker-backed `spreadsheetEngine`; needs `xlsx`), `/tailwind-preset`, `/vite`.

## Selectors

The viewer's `data-testid`s are stable test API. `fileViewerSelectors` (a literal selector
tree for `@vrooli/ui-selectors`) lists them; spread it into your registry so your manifest
carries them:

```ts
import { fileViewerSelectors } from "@vrooli/file-viewer/selectors";
const literalSelectors = { ...fileViewerSelectors, /* your own */ };
```

## Development

```bash
pnpm build        # tsc + asset copy -> dist/
pnpm test         # vitest (jsdom; the Vite plugin's tests run in node)
pnpm type-check
```

`@vrooli/react-component-library`, `xlsx` and `mermaid` are governed dev dependencies of this
package (and optional or required peers for consumers), so its build and tests resolve them from
its own `node_modules` with no path aliases. The `file:` copy of the component library is
refreshed with `vrooli package refresh`; a stale copy only matters if a test needs a newer
library release. `src/__tests__/optionalPeers.test.ts`
fails if anything outside `src/spreadsheet` imports `xlsx`, anything imports `mermaid`, or either
becomes a regular dependency.
The real-file tests read the demo files under `scenarios/web-console/fixtures/file-preview`
(`src/test-utils/fixtures.ts`).

**Builds overwrite `dist/` in place; do not delete it first.** Consumers install this
package with a pnpm `file:` dependency, and their `node_modules/@vrooli/file-viewer/dist/*`
files are hard links to these files. Removing `dist/` (`rmSync`, `rimraf`, `tsc --build
--clean`) and writing new files orphans those links, so a hand-run build leaves consumers on
the old build and the files it adds never reach them. `scripts/build.mjs` therefore compiles
over the existing files (same inodes) and afterwards deletes only outputs older than the
build, so removed sources still disappear. Lifecycle commands (`vrooli package refresh
file-viewer --no-restart`) detach the shipped files first, which gives each consumer a
consistent snapshot until its setup reinstalls; the in-place build is what keeps a manual
`node scripts/build.mjs` safe too.

Layout: `src/source` (the contract and the manifest source), `src/viewer` (controller
and shell), `src/preview` (renderers, per-kind data, hooks and workers), `src/markdown`,
`src/i18n` (catalogs and provider), `styles/` (CSS and Tailwind preset), `vite/` (the
Vite plugin).
