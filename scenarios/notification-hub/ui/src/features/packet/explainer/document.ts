/**
 * The sealed document an HTML explainer is shown in
 * (docs/concepts/DECISION_PACKETS.md, "HTML explainers").
 *
 * The agent's page is untrusted. It is rendered through `srcdoc` into an
 * `<iframe sandbox="allow-scripts">` (opaque origin: no cookies, storage or
 * API of the Hub), and this builder adds what keeps it inert:
 *
 *  1. a Content-Security-Policy meta as the very first thing in `<head>`, so
 *     nothing the page says can run or load before it applies;
 *  2. a one-rule baseline (`html, body { margin: 0 }`) in its own layer, so a
 *     page that has no kit still starts flush against the Hub's frame, and any
 *     rule the page writes still wins;
 *  3. the starter kit stylesheet, which declares everything inside
 *     `@layer hub-kit` so anything the page writes itself wins, unless the page
 *     opts out with `<meta name="hub-kit" content="off">` (then neither the
 *     kit's CSS nor its script is injected);
 *  4. the bridge script and then the kit's behaviours.
 *
 * Attachment names used as `src`, `href` and `poster` become `data:` URLs
 * within size limits; beyond them an element keeps a placeholder that opens
 * the Hub's viewer when tapped.
 */
export const EXPLAINER_CSP =
  "default-src 'none'; img-src data: blob:; media-src data: blob:; font-src data:; style-src 'unsafe-inline'; script-src 'unsafe-inline'";

/**
 * The Hub's frame gives the page its gutter, so the browser's own 8px body
 * margin must not stack on it. A layer, so anything the page writes wins.
 */
export const EXPLAINER_BASE_CSS = "@layer hub-base { html, body { margin: 0; } }";

/** Largest attachment inlined as a data URL, and the most one page may inline. */
export const INLINE_FILE_LIMIT = 12 * 1024 * 1024;
export const INLINE_PAGE_LIMIT = 40 * 1024 * 1024;

/** A blank, text-free tile an oversized attachment shows until it is tapped. */
export const OPEN_PLACEHOLDER =
  "data:image/svg+xml;charset=utf-8," +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="320" height="200" viewBox="0 0 320 200"><rect width="320" height="200" fill="#94a3b8" fill-opacity=".25"/><path d="M130 70h60a8 8 0 0 1 8 8v44a8 8 0 0 1-8 8h-60a8 8 0 0 1-8-8V78a8 8 0 0 1 8-8zm22 18v24l22-12z" fill="#64748b"/></svg>',
  );

const REFERENCE_ATTRIBUTES = ["src", "href", "poster"] as const;

/** What the builder needs to know about one attachment of the ask. */
export interface ExplainerAttachment {
  name: string;
  folder: boolean;
  sizeBytes: number;
  mediaType: string;
  entries: readonly { path: string; sizeBytes: number; mediaType: string }[];
}

/** One attribute value in the page that names an attachment. */
export interface ReferenceTarget {
  /** The attribute value exactly as the page wrote it. */
  value: string;
  name: string;
  /** The file inside a folder attachment; "" for a file attachment. */
  path: string;
  sizeBytes: number;
  mediaType: string;
}

function parse(html: string): Document {
  return new DOMParser().parseFromString(html, "text/html");
}

function referencedElements(doc: Document): Element[] {
  return [...doc.querySelectorAll(REFERENCE_ATTRIBUTES.map((attribute) => `[${attribute}]`).join(","))];
}

function resolveReference(value: string, attachments: readonly ExplainerAttachment[]): ReferenceTarget | null {
  const trimmed = value.trim().replace(/^\.\//, "");
  if (!trimmed) return null;
  const slash = trimmed.indexOf("/");
  const name = slash < 0 ? trimmed : trimmed.slice(0, slash);
  const attachment = attachments.find((item) => item.name === name);
  if (!attachment) return null;
  if (slash < 0) {
    return attachment.folder ? null : { value, name, path: "", sizeBytes: attachment.sizeBytes, mediaType: attachment.mediaType };
  }
  const path = trimmed.slice(slash + 1);
  const entry = attachment.folder ? attachment.entries.find((item) => item.path === path) : undefined;
  return entry ? { value, name, path, sizeBytes: entry.sizeBytes, mediaType: entry.mediaType } : null;
}

/** Every distinct attachment reference in the page, in document order. */
export function findReferences(html: string, attachments: readonly ExplainerAttachment[]): ReferenceTarget[] {
  const found = new Map<string, ReferenceTarget>();
  for (const element of referencedElements(parse(html))) {
    for (const attribute of REFERENCE_ATTRIBUTES) {
      const value = element.getAttribute(attribute);
      if (value === null || found.has(value)) continue;
      const target = resolveReference(value, attachments);
      if (target) found.set(value, target);
    }
  }
  return [...found.values()];
}

/**
 * Which references are inlined and which stay placeholders. Document order
 * decides: the first files that fit the page budget are inlined.
 */
export function planInlining(
  references: readonly ReferenceTarget[],
  limits: { file: number; page: number } = { file: INLINE_FILE_LIMIT, page: INLINE_PAGE_LIMIT },
): { inline: ReferenceTarget[]; deferred: ReferenceTarget[] } {
  const inline: ReferenceTarget[] = [];
  const deferred: ReferenceTarget[] = [];
  let used = 0;
  for (const reference of references) {
    if (reference.sizeBytes <= limits.file && used + reference.sizeBytes <= limits.page) {
      inline.push(reference);
      used += reference.sizeBytes;
    } else {
      deferred.push(reference);
    }
  }
  return { inline, deferred };
}

export interface ExplainerDocumentOptions {
  /** The kit stylesheet as published: every rule already sits in `@layer hub-kit`. */
  kitCss: string;
  kitJs: string;
  bridgeJs: string;
  theme: "light" | "dark";
  /** Attribute value -> `data:` URL, for the references that were inlined. */
  dataUrls: ReadonlyMap<string, string>;
  /** The references that stay placeholders and open the viewer instead. */
  deferred: readonly ReferenceTarget[];
}

// A script or style body must not be able to close its own element.
const inertScript = (source: string) => source.replace(/<\/(script)/gi, "<\\/$1");
const inertStyle = (source: string) => source.replace(/<\/(style)/gi, "<\\/$1");

function element(doc: Document, tag: string, attributes: Record<string, string>, text = ""): HTMLElement {
  const node = doc.createElement(tag);
  for (const [name, value] of Object.entries(attributes)) node.setAttribute(name, value);
  if (text) node.textContent = text;
  return node;
}

/** Build the complete `srcdoc` for an explainer page. */
export function buildExplainerDocument(html: string, options: ExplainerDocumentOptions): string {
  const doc = parse(html);
  const deferred = new Map(options.deferred.map((target) => [target.value, target]));

  // A page must not redirect its own frame or re-root its relative URLs.
  doc.querySelectorAll("base, meta[http-equiv]").forEach((node) => {
    if (node.tagName === "BASE" || node.getAttribute("http-equiv")?.toLowerCase() === "refresh") node.remove();
  });

  for (const node of referencedElements(doc)) {
    for (const attribute of REFERENCE_ATTRIBUTES) {
      const value = node.getAttribute(attribute);
      if (value === null) continue;
      const dataUrl = options.dataUrls.get(value);
      const placeholder = deferred.get(value);
      if (dataUrl) {
        node.setAttribute(attribute, dataUrl);
      } else if (placeholder) {
        node.setAttribute(attribute, attribute === "href" ? "#" : OPEN_PLACEHOLDER);
        node.setAttribute("data-hub-open", placeholder.path ? `${placeholder.name}/${placeholder.path}` : placeholder.name);
      }
    }
  }

  const kitOff = Boolean(doc.querySelector('meta[name="hub-kit" i][content="off" i]'));
  const injected: HTMLElement[] = [
    element(doc, "meta", { "http-equiv": "Content-Security-Policy", content: EXPLAINER_CSP }),
    element(doc, "style", { "data-hub": "base" }, EXPLAINER_BASE_CSS),
    ...(kitOff ? [] : [element(doc, "style", { "data-hub": "kit" }, inertStyle(options.kitCss))]),
    element(doc, "script", { "data-hub": "bridge" }, inertScript(options.bridgeJs)),
    ...(kitOff || !options.kitJs ? [] : [element(doc, "script", { "data-hub": "kit" }, inertScript(options.kitJs))]),
  ];
  for (const node of [...injected].reverse()) doc.head.insertBefore(node, doc.head.firstChild);
  doc.documentElement.setAttribute("data-theme", options.theme);

  return `<!doctype html>\n${doc.documentElement.outerHTML}`;
}
