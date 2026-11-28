import { DEFAULT_TEMPLATE } from "../templates/library";
import { node, type Template, type TemplateNode } from "../templates/spec";
import { parseHtmlCssRules, parseHtmlPseudoRules } from "./htmlImportStyles";
import { parseSvgCssRules } from "./htmlImportSvg";
import { buildSourcePreview } from "./htmlImportPreview";
import { HtmlMapper } from "./htmlImportMapper";

/**
 * HTML import converts an inert HTML document into the studio's existing
 * `TemplateNode` tree. It is reference material: the source page is never
 * inserted into the live document, nothing is fetched, and the map only keeps
 * text, headings and layout. Styling, scripts, media and attributes are gone.
 *
 * The parser is pure and table-testable: `parseHtml` takes a string, and the
 * returned object owns the (detached) DOM plus deterministic node ids. A
 * bounded set of static presentation declarations is retained on each layer;
 * scripts, event handlers, external URLs and stateful CSS are not imported.
 */
export { HTML_LIMITS, HtmlImportError } from "./htmlImportTypes";
export type { HtmlImportReason, HtmlRegion, ParsedHtml } from "./htmlImportTypes";
import { HTML_LIMITS, HtmlImportError } from "./htmlImportTypes";
import type { ParsedHtml } from "./htmlImportTypes";

const REMOVED_TAGS = [
  "script",
  "style",
  "link",
  "meta",
  "noscript",
  "template",
  "iframe",
  "object",
  "embed",
  "audio",
  "canvas",
] as const;

const UNWRAPPED_TAGS = ["form"] as const;

function collapse(text: string): string {
  return text.replace(/\s+/g, " ").trim();
}

function elementPath(element: Element, root: Element): number[] | undefined {
  const path: number[] = [];
  let current: Element = element;
  while (current !== root) {
    const parent = current.parentElement;
    if (!parent) return undefined;
    const index = Array.prototype.indexOf.call(parent.children, current);
    if (index < 0) return undefined;
    path.unshift(index);
    current = parent;
  }
  return path;
}

function cloneSpec(spec: TemplateNode): TemplateNode {
  return {
    ...spec,
    props: { ...spec.props },
    ...(spec.accepts ? { accepts: [...spec.accepts] } : {}),
    ...(spec.children ? { children: spec.children.map(cloneSpec) } : {}),
  };
}

function pageSpec(children: TemplateNode[], css?: string): TemplateNode {
  return node("html-page", "page", { pad: 0, ...(css ? { css } : {}) }, children);
}

/**
 * The mapping is intentionally one-directional and content-independent: ids
 * follow the pre-order layer index, never the text. That keeps the ids stable
 * between the whole-document view and a region subtree.
 */

function byteLength(source: string): number {
  if (typeof TextEncoder !== "undefined") return new TextEncoder().encode(source).byteLength;
  return new Blob([source]).size;
}

function titleFromFileName(fileName: string | undefined): string {
  if (!fileName) return "";
  const base = fileName.replace(/^.*[\\/]/, "").replace(/\.[^./\\]*$/, "");
  return base.trim();
}

function parseDocument(source: string): Document {
  try {
    const document = new DOMParser().parseFromString(source, "text/html");
    if (document.documentElement.nodeName.toLowerCase() === "parsererror") {
      throw new HtmlImportError("parse", "The HTML could not be parsed.");
    }
    return document;
  } catch (cause) {
    if (cause instanceof HtmlImportError) throw cause;
    throw new HtmlImportError("parse", "The HTML could not be parsed.");
  }
}

/**
 * Parse an HTML document into importable regions. The document is detached and
 * sanitised before any mapping; nothing here touches the live DOM.
 */
export function parseHtml(
  source: string,
  fileName?: string,
  stylesheets: string[] = [],
): ParsedHtml {
  const totalBytes =
    byteLength(source) +
    stylesheets.reduce((total, stylesheet) => total + byteLength(stylesheet), 0);
  if (source.length > HTML_LIMITS.bytes || totalBytes > HTML_LIMITS.bytes) {
    throw new HtmlImportError("size", "The HTML document is larger than 2 MiB.");
  }
  if (source.trim() === "") {
    throw new HtmlImportError("empty", "The HTML document is empty.");
  }

  const document = parseDocument(source);
  const sourcePaths = new WeakMap<Element, number[]>();
  for (const element of Array.from(document.body.querySelectorAll("*"))) {
    const path = elementPath(element, document.body);
    if (path) sourcePaths.set(element, path);
  }

  const inlineStyles = Array.from(
    document.querySelectorAll("style"),
    (style) => style.textContent || "",
  );
  const removals = document.body.querySelectorAll(REMOVED_TAGS.join(","));
  for (const element of Array.from(removals)) element.remove();
  for (const tag of UNWRAPPED_TAGS) {
    for (const element of Array.from(document.body.querySelectorAll(tag))) {
      element.replaceWith(...Array.from(element.childNodes));
    }
  }

  const allStylesheets = [...inlineStyles, ...stylesheets];
  const mapper = new HtmlMapper(
    document.body,
    document,
    parseSvgCssRules(allStylesheets),
    parseHtmlCssRules(allStylesheets),
    parseHtmlPseudoRules(allStylesheets),
    sourcePaths,
  );
  const { layers, regions, warnings } = mapper.map();
  const title = collapse(document.title) || titleFromFileName(fileName) || "Imported HTML";

  return {
    title,
    layers,
    regions,
    warnings,
    sourcePreview(regionId, previewStylesheets = []) {
      return buildSourcePreview(source, parseDocument, regionId, regions, previewStylesheets);
    },
    regionSpec(regionId: string | null): TemplateNode {
      if (regionId === null) {
        return pageSpec(mapper.bodyChildren().map(cloneSpec), mapper.pageCss());
      }
      const found = mapper.regionWithContext(regionId);
      if (!found) {
        throw new HtmlImportError("parse", `Unknown HTML region ${regionId}.`);
      }
      return pageSpec([cloneSpec(found)], mapper.pageCss());
    },
  };
}

let templateSequence = 0;

/**
 * Build a Template from one parsed document. The palette, display, radius,
 * category and slot hint come from the shared default so the studio renders the
 * import like any other draft; the tagline marks the source.
 */
export function htmlTemplate(
  parsed: ParsedHtml,
  regionId: string | null,
  name: string,
  sourcePreview = "",
): Template {
  const cleanName = name.trim() || parsed.title || "Imported HTML";
  templateSequence += 1;
  const spec = parsed.regionSpec(regionId);
  return {
    id: `html-import-${Date.now().toString(36)}-${templateSequence.toString(36)}`,
    name: cleanName,
    tagline: "Imported HTML",
    category: DEFAULT_TEMPLATE.category,
    slotHint: DEFAULT_TEMPLATE.slotHint,
    idea: "Imported reference material mapped into the studio layer model.",
    palette: { ...DEFAULT_TEMPLATE.palette },
    display: DEFAULT_TEMPLATE.display,
    radius: DEFAULT_TEMPLATE.radius,
    spec: {
      ...spec,
      props: {
        ...spec.props,
        ...(sourcePreview ? { sourceHtml: sourcePreview } : {}),
      },
    },
    local: true,
  };
}
