import { DEFAULT_TEMPLATE } from "../templates/library";
import { node, type Template, type TemplateNode, type TemplateNodeType } from "../templates/spec";

/**
 * HTML import converts an inert HTML document into the studio's existing
 * `TemplateNode` tree. It is reference material: the source page is never
 * inserted into the live document, nothing is fetched, and the map only keeps
 * text, headings and layout. Styling, scripts, media and attributes are gone.
 *
 * The parser is pure and table-testable: `parseHtml` takes a string, and the
 * returned object owns the (detached) DOM plus deterministic node ids.
 */
export const HTML_LIMITS = {
  /** Input bytes accepted before the file is refused. */
  bytes: 2 * 1024 * 1024,
  /** Source elements visited before the scan stops. */
  elements: 500,
  /** Content layers emitted before the import stops (the page root is extra). */
  layers: 200,
  /** Maximum produced layer depth: page is 0, its first children are 1. */
  depth: 12,
} as const;

export type HtmlImportReason = "size" | "empty" | "parse";

/** A typed refusal so the dialog can explain *why* a file was not imported. */
export class HtmlImportError extends Error {
  readonly reason: HtmlImportReason;

  constructor(reason: HtmlImportReason, message: string) {
    super(message);
    this.name = "HtmlImportError";
    this.reason = reason;
  }
}

/**
 * One selectable region of the source document. Regions are listed in document
 * order, so `depth` doubles as the indentation for a flat tree/list control.
 */
export interface HtmlRegion {
  id: string;
  /** Lower-case source tag (`section`, `p`, `table`…). */
  tag: string;
  /** Collapsed text snippet, capped at 80 characters; may be empty. */
  label: string;
  /** Source nesting relative to the body: top-level children are 0. */
  depth: number;
  /** Produced layers in this element's subtree, including the element itself. */
  layers: number;
  /** Direct source element children, after sanitisation. */
  children: number;
}

export interface ParsedHtml {
  /** `<title>`, the file name, or "Imported HTML". */
  title: string;
  /** Content layers across the whole document (page root excluded). */
  layers: number;
  /** Importable regions, document order. Elements that produced nothing are omitted. */
  regions: HtmlRegion[];
  warnings: string[];
  /** Build a fresh spec for one region; `null` means the whole body. */
  regionSpec(id: string | null): TemplateNode;
}

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

/** Image-bearing elements whose content is dropped rather than fetched. */
const IMAGE_SELECTOR = "img, picture, svg, video";

const CONTAINER_TAGS = new Set(["section", "article", "header", "footer", "main", "nav", "div"]);

const LIST_TAGS = new Set(["ul", "ol"]);
const TABLE_GROUP_TAGS = new Set(["table", "tbody"]);

const HEADING_DISPLAY = new Set(["h1", "h2"]);
const HEADING_TITLE = new Set(["h3", "h4", "h5", "h6"]);

interface ScanState {
  elements: number;
  layers: number;
  truncatedElements: boolean;
  truncatedDepth: boolean;
  truncatedLayers: boolean;
}

function collapse(text: string): string {
  return text.replace(/\s+/g, " ").trim();
}

function snippet(text: string): string {
  const collapsed = collapse(text);
  return collapsed.length > 80 ? `${collapsed.slice(0, 79)}…` : collapsed;
}

function countLayers(spec: TemplateNode): number {
  let total = 1;
  for (const child of spec.children ?? []) total += countLayers(child);
  return total;
}

/** Region labels read block children with a space, not concatenated textContent. */
function textContentOf(spec: TemplateNode): string {
  const own = typeof spec.props.text === "string" ? spec.props.text : "";
  const children = (spec.children ?? []).map(textContentOf).filter((text) => text.length > 0);
  return [own, ...children].filter((text) => text.length > 0).join(" ");
}

function reserveLayer(state: ScanState, type: TemplateNodeType): string | null {
  if (state.layers >= HTML_LIMITS.layers) {
    state.truncatedLayers = true;
    return null;
  }
  state.layers += 1;
  return `html-${state.layers}-${type}`;
}

function cloneSpec(spec: TemplateNode): TemplateNode {
  return {
    ...spec,
    props: { ...spec.props },
    ...(spec.accepts ? { accepts: [...spec.accepts] } : {}),
    ...(spec.children ? { children: spec.children.map(cloneSpec) } : {}),
  };
}

function pageSpec(children: TemplateNode[]): TemplateNode {
  return node("html-page", "page", { pad: 40 }, children);
}

/**
 * The mapping is intentionally one-directional and content-independent: ids
 * follow the pre-order layer index, never the text. That keeps the ids stable
 * between the whole-document view and a region subtree.
 */
class HtmlMapper {
  private readonly state: ScanState = {
    elements: 0,
    layers: 0,
    truncatedElements: false,
    truncatedDepth: false,
    truncatedLayers: false,
  };

  private readonly regionEntries: HtmlRegion[] = [];
  private readonly nodesByRegion = new Map<string, TemplateNode>();
  private readonly skippedImages: number;
  private wholeChildren: TemplateNode[] = [];

  constructor(private readonly body: HTMLElement) {
    this.skippedImages = this.removeSkippedImages();
    this.stripAttributes();
  }

  map(): { layers: number; regions: HtmlRegion[]; warnings: string[] } {
    this.wholeChildren = this.mapChildren(this.body, 1, 0);
    const regions = this.regionEntries.filter((entry) => this.nodesByRegion.has(entry.id));
    return {
      layers: this.state.layers,
      regions,
      warnings: this.warnings(),
    };
  }

  bodyChildren(): TemplateNode[] {
    return this.wholeChildren;
  }

  nodesForRegion(id: string): TemplateNode | undefined {
    return this.nodesByRegion.get(id);
  }

  private removeSkippedImages(): number {
    let skipped = 0;
    for (const image of Array.from(this.body.querySelectorAll(IMAGE_SELECTOR))) {
      // A <picture> (or <svg>) ancestor may already have removed this node.
      if (!image.isConnected) continue;
      image.remove();
      skipped += 1;
    }
    return skipped;
  }

  private stripAttributes(): void {
    for (const element of [this.body, ...Array.from(this.body.querySelectorAll("*"))]) {
      for (const name of element.getAttributeNames()) element.removeAttribute(name);
    }
  }

  private warnings(): string[] {
    const warnings: string[] = [];
    if (this.skippedImages > 0) {
      const noun = this.skippedImages === 1 ? "image was" : "images were";
      warnings.push(`${this.skippedImages} ${noun} skipped; remote assets are not fetched.`);
    }
    if (this.state.truncatedElements) {
      warnings.push(
        `Only the first ${HTML_LIMITS.elements} source elements were scanned; later content is missing.`,
      );
    }
    if (this.state.truncatedDepth) {
      warnings.push(`Nested content was cut off at ${HTML_LIMITS.depth} levels.`);
    }
    if (this.state.truncatedLayers) {
      warnings.push(`Only the first ${HTML_LIMITS.layers} layers were imported.`);
    }
    return warnings;
  }

  private mapChildren(parent: Element, depth: number, sourceDepth: number): TemplateNode[] {
    if (depth > HTML_LIMITS.depth) {
      this.state.truncatedDepth = true;
      return [];
    }
    const children: TemplateNode[] = [];
    for (const child of Array.from(parent.childNodes)) {
      if (this.state.elements >= HTML_LIMITS.elements) {
        this.state.truncatedElements = true;
        break;
      }
      if (this.state.layers >= HTML_LIMITS.layers) {
        this.state.truncatedLayers = true;
        break;
      }
      if (child.nodeType === 1 && child instanceof Element) {
        const mapped = this.mapElement(child, depth, sourceDepth);
        if (mapped) children.push(mapped);
        continue;
      }
      if (child.nodeType !== 3) continue;
      const text = collapse(child.nodeValue ?? "");
      if (!text) continue;
      const id = reserveLayer(this.state, "text");
      if (!id) return children;
      children.push(node(id, "text", { text, variant: "body" }));
    }
    return children;
  }

  private mapElement(element: Element, depth: number, sourceDepth: number): TemplateNode | null {
    this.state.elements += 1;
    const index = this.state.elements;
    if (index > HTML_LIMITS.elements) {
      this.state.truncatedElements = true;
      return null;
    }
    if (depth > HTML_LIMITS.depth) {
      this.state.truncatedDepth = true;
      return null;
    }

    const tag = element.tagName.toLowerCase();
    const region: HtmlRegion = {
      id: `region-${index}`,
      tag,
      label: "",
      depth: sourceDepth,
      layers: 0,
      children: element.children.length,
    };
    this.regionEntries.push(region);

    const mapped = this.mapTag(element, tag, depth, sourceDepth);
    if (mapped) {
      region.label = snippet(textContentOf(mapped));
      region.layers = countLayers(mapped);
      this.nodesByRegion.set(region.id, mapped);
    }
    return mapped;
  }

  private mapTag(
    element: Element,
    tag: string,
    depth: number,
    sourceDepth: number,
  ): TemplateNode | null {
    if (HEADING_DISPLAY.has(tag)) return this.textLayer(element, "display");
    if (HEADING_TITLE.has(tag)) return this.textLayer(element, "title");
    if (tag === "p" || tag === "span" || tag === "label") return this.textLayer(element, "body");
    if (tag === "small") return this.textLayer(element, "body", "muted");
    if (tag === "a") return this.textLayer(element, "body", "accent");
    if (tag === "button") return this.buttonLayer(element);
    if (CONTAINER_TAGS.has(tag)) return this.container(element, "stack", depth, sourceDepth, 14);
    if (LIST_TAGS.has(tag)) return this.container(element, "stack", depth, sourceDepth, 14);
    if (TABLE_GROUP_TAGS.has(tag)) return this.container(element, "stack", depth, sourceDepth, 14);
    if (tag === "li") {
      return element.children.length > 0
        ? this.container(element, "row", depth, sourceDepth, 12)
        : this.textLayer(element, "body");
    }
    if (tag === "tr") return this.container(element, "row", depth, sourceDepth, 12);
    if (tag === "td" || tag === "th") return this.textLayer(element, "body");
    if (tag === "hr") return this.leaf("divider", {});
    if (tag === "br") return this.leaf("spacer", { h: 12 });
    // Unknown element: structural when it has element children, copy otherwise.
    return element.children.length > 0
      ? this.container(element, "stack", depth, sourceDepth, 14)
      : this.textLayer(element, "body");
  }

  private textLayer(
    element: Element,
    variant: "display" | "title" | "body",
    tone?: "muted" | "accent",
  ): TemplateNode | null {
    const text = collapse(element.textContent);
    if (!text) return null;
    const id = reserveLayer(this.state, "text");
    if (!id) return null;
    return node(id, "text", tone ? { text, variant, tone } : { text, variant });
  }

  private buttonLayer(element: Element): TemplateNode | null {
    const text = collapse(element.textContent);
    if (!text) return null;
    const id = reserveLayer(this.state, "button");
    if (!id) return null;
    return node(id, "button", { text, variant: "secondary" });
  }

  private leaf(
    type: TemplateNodeType,
    props: Record<string, string | number | boolean>,
  ): TemplateNode | null {
    const id = reserveLayer(this.state, type);
    if (!id) return null;
    return node(id, type, props);
  }

  private container(
    element: Element,
    type: "stack" | "row",
    depth: number,
    sourceDepth: number,
    gap: number,
  ): TemplateNode | null {
    const id = reserveLayer(this.state, type);
    if (!id) return null;
    const children = this.mapChildren(element, depth + 1, sourceDepth + 1);
    if (children.length === 0) {
      // The reserved slot produced nothing; release it so empty wrappers do
      // not consume the layer budget.
      this.state.layers -= 1;
      return null;
    }
    const label = sourceLabel(element);
    return node(id, type, label ? { gap, label } : { gap }, children);
  }
}

function byteLength(source: string): number {
  if (typeof TextEncoder !== "undefined") return new TextEncoder().encode(source).byteLength;
  return new Blob([source]).size;
}

function titleFromFileName(fileName: string | undefined): string {
  if (!fileName) return "";
  const base = fileName.replace(/^.*[\\/]/, "").replace(/\.[^./\\]*$/, "");
  return base.trim();
}

/**
 * A layer tree of `Stack · html 9 stack` is noise. Name a container after the
 * source role, plus its own heading when it has one, so the imported tree
 * reads like the page: "Section · Why now", "Header", "List".
 */
const SOURCE_LABELS: Record<string, string> = {
  header: "Header",
  footer: "Footer",
  main: "Main",
  nav: "Nav",
  section: "Section",
  article: "Article",
  aside: "Aside",
  div: "Group",
  ul: "List",
  ol: "List",
  li: "Item",
  table: "Table",
  thead: "Table head",
  tbody: "Table body",
  tfoot: "Table foot",
  tr: "Row",
};

function sourceLabel(element: Element): string | undefined {
  const base = SOURCE_LABELS[element.tagName.toLowerCase()];
  if (!base) return undefined;
  const heading = element.querySelector(
    ":scope > h1, :scope > h2, :scope > h3, :scope > h4, :scope > h5, :scope > h6",
  );
  const text = heading ? collapse(heading.textContent) : "";
  return text ? `${base} · ${text.slice(0, 24)}` : base;
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
export function parseHtml(source: string, fileName?: string): ParsedHtml {
  if (source.length > HTML_LIMITS.bytes || byteLength(source) > HTML_LIMITS.bytes) {
    throw new HtmlImportError("size", "The HTML document is larger than 2 MiB.");
  }
  if (source.trim() === "") {
    throw new HtmlImportError("empty", "The HTML document is empty.");
  }

  const document = parseDocument(source);

  // Remove executable/embedded wrappers first, then keep form content while
  // dropping the form element itself.
  const removals = document.body.querySelectorAll(REMOVED_TAGS.join(","));
  for (const element of Array.from(removals)) element.remove();
  for (const tag of UNWRAPPED_TAGS) {
    for (const element of Array.from(document.body.querySelectorAll(tag))) {
      element.replaceWith(...Array.from(element.childNodes));
    }
  }

  const mapper = new HtmlMapper(document.body);
  const { layers, regions, warnings } = mapper.map();
  const title = collapse(document.title) || titleFromFileName(fileName) || "Imported HTML";

  return {
    title,
    layers,
    regions,
    warnings,
    regionSpec(regionId: string | null): TemplateNode {
      if (regionId === null) {
        return pageSpec(mapper.bodyChildren().map(cloneSpec));
      }
      const found = mapper.nodesForRegion(regionId);
      if (!found) {
        throw new HtmlImportError("parse", `Unknown HTML region ${regionId}.`);
      }
      return pageSpec([cloneSpec(found)]);
    },
  };
}

let templateSequence = 0;

/**
 * Build a Template from one parsed document. The palette, display, radius,
 * category and slot hint come from the shared default so the studio renders the
 * import like any other draft; the tagline marks the source.
 */
export function htmlTemplate(parsed: ParsedHtml, regionId: string | null, name: string): Template {
  const cleanName = name.trim() || parsed.title || "Imported HTML";
  templateSequence += 1;
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
    spec: parsed.regionSpec(regionId),
    local: true,
  };
}
