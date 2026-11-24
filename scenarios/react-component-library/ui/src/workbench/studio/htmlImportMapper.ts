import { node, type TemplateNode, type TemplateNodeType } from "../templates/spec";
import { HTML_LIMITS, type HtmlRegion } from "./htmlImportTypes";
import {
  compareCssSpecificity,
  ensureInheritedTextColor,
  hasVisualOrGeometryStyle,
  htmlStyle,
  type HtmlCssRule,
  type HtmlPseudoRule,
} from "./htmlImportStyles";
import { safeSvg, svgDataUrl, type SvgCssRule } from "./htmlImportSvg";

const IMAGE_SELECTOR = "img, picture, video";
const CONTAINER_TAGS = new Set(["section", "article", "header", "footer", "main", "nav", "div"]);
const BLOCK_CONTAINER_TAGS = new Set([
  "section",
  "article",
  "header",
  "footer",
  "main",
  "nav",
  "div",
  "ul",
  "ol",
  "li",
  "table",
  "tbody",
]);
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

function sourceHint(element: Element): string | undefined {
  const raw =
    element.getAttribute("aria-label") ||
    element.getAttribute("data-app") ||
    element.getAttribute("data-name") ||
    element.getAttribute("title") ||
    element.id;
  const clean = collapse(raw);
  return clean ? snippet(clean.replace(/[-_]+/g, " ")) : undefined;
}

function byteLength(source: string): number {
  if (typeof TextEncoder !== "undefined") return new TextEncoder().encode(source).byteLength;
  return new Blob([source]).size;
}

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

function countLayers(spec: TemplateNode): number {
  let total = 1;
  for (const child of spec.children ?? []) total += countLayers(child);
  return total;
}

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

export class HtmlMapper {
  private readonly state: ScanState = {
    elements: 0,
    layers: 0,
    truncatedElements: false,
    truncatedDepth: false,
    truncatedLayers: false,
  };

  private readonly regionEntries: HtmlRegion[] = [];
  private readonly nodesByRegion = new Map<string, TemplateNode>();
  private readonly contextByRegion = new Map<string, string[]>();
  private readonly skippedImages: number;
  private readonly sourceDocument: Document;
  private unsupportedSvgs = 0;
  private svgBytes = 0;
  private styleBytes = 0;
  private stylesTruncated = false;
  private cachedPageCss: string | undefined;
  private pageCssResolved = false;
  private wholeChildren: TemplateNode[] = [];

  constructor(
    private readonly body: HTMLElement,
    sourceDocument: Document,
    private readonly svgStylesheets: SvgCssRule[],
    private readonly htmlStylesheets: HtmlCssRule[],
    private readonly pseudoStylesheets: HtmlPseudoRule[],
    private readonly sourcePaths: WeakMap<Element, number[]>,
  ) {
    this.sourceDocument = sourceDocument;
    this.skippedImages = this.removeSkippedImages();
  }

  map(): { layers: number; regions: HtmlRegion[]; warnings: string[] } {
    this.wholeChildren = this.mapChildren(this.body, 1, 0);
    this.pageCss();
    // Read descriptive identifiers for the chooser during mapping, then strip
    // all source attributes before the detached document leaves this mapper.
    this.stripAttributes();
    const regions = this.regionEntries
      .filter((entry) => this.nodesByRegion.has(entry.id))
      // Promote explicitly named source regions (for example the eight
      // data-app articles in a screen study) above their many child layers.
      .sort((left, right) => Number(Boolean(right.sourceHint)) - Number(Boolean(left.sourceHint)));
    return {
      layers: this.state.layers,
      regions,
      warnings: this.warnings(),
    };
  }

  bodyChildren(): TemplateNode[] {
    return this.wholeChildren;
  }

  pageCss(): string | undefined {
    if (this.pageCssResolved) return this.cachedPageCss;
    this.pageCssResolved = true;
    const declarations = new Map<string, string>();
    const root = this.sourceDocument.documentElement;
    const rootStyle = this.styleForElement(root);
    if (rootStyle) {
      try {
        for (const [name, value] of Object.entries(
          JSON.parse(rootStyle) as Record<string, string>,
        )) {
          if (name.startsWith("--")) declarations.set(name, value);
        }
      } catch {
        /* Keep mapping other elements if stored styles are malformed. */
      }
    }
    const body = this.styleForElement(this.body);
    if (body) {
      try {
        for (const [name, value] of Object.entries(JSON.parse(body) as Record<string, string>)) {
          declarations.set(name, value);
        }
      } catch {
        /* The mapper only emits valid JSON; keep root tokens if parsing fails. */
      }
    }
    this.cachedPageCss = declarations.size
      ? JSON.stringify(Object.fromEntries(declarations))
      : undefined;
    return this.cachedPageCss;
  }

  regionWithContext(id: string): TemplateNode | undefined {
    const source = this.nodesByRegion.get(id);
    if (!source) return undefined;
    const inherited = new Map<string, string>();
    const inheritedProperties = new Set([
      "color",
      "font",
      "font-family",
      "font-size",
      "font-style",
      "font-variant-numeric",
      "font-weight",
      "letter-spacing",
      "line-height",
      "text-align",
      "text-decoration",
      "text-rendering",
      "text-transform",
      "white-space",
    ]);
    for (const serialized of this.contextByRegion.get(id) ?? []) {
      try {
        for (const [name, value] of Object.entries(
          JSON.parse(serialized) as Record<string, string>,
        )) {
          if (name.startsWith("--") || inheritedProperties.has(name)) inherited.set(name, value);
        }
      } catch {
        // Invalid source declarations are discarded by htmlStyle; keep other context.
      }
    }
    try {
      for (const [name, value] of Object.entries(
        JSON.parse(String(source.props.css ?? "{}")) as Record<string, string>,
      )) {
        inherited.set(name, value);
      }
    } catch {
      // Preserve the mapped node if its style payload is malformed.
    }
    return {
      ...source,
      props: {
        ...source.props,
        ...(inherited.size ? { css: JSON.stringify(Object.fromEntries(inherited)) } : {}),
      },
    };
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

  private styleForElement(element: Element, account = true): string | undefined {
    const style = htmlStyle(element, this.htmlStylesheets);
    return account ? this.accountStyle(style) : style;
  }

  private accountStyle(style: string | undefined): string | undefined {
    if (!style) return undefined;
    const bytes = byteLength(style);
    if (this.styleBytes + bytes > HTML_LIMITS.styleBytes) {
      this.stylesTruncated = true;
      return undefined;
    }
    this.styleBytes += bytes;
    return style;
  }

  private pseudoLayers(element: Element, depth: number): TemplateNode[] {
    const layers: TemplateNode[] = [];
    for (const pseudo of ["before", "after"] as const) {
      const matched: HtmlPseudoRule[] = [];
      for (const rule of this.pseudoStylesheets) {
        if (rule.pseudo !== pseudo) continue;
        try {
          if (element.matches(rule.selector)) matched.push(rule);
        } catch {
          /* Ignore selectors unsupported by the detached DOM. */
        }
      }
      matched.sort(compareCssSpecificity);
      const declarations = new Map<string, string>();
      let content: string | undefined;
      for (const rule of matched) {
        for (const [name, value] of rule.declarations) declarations.set(name, value);
        if (rule.content !== undefined) content = rule.content;
      }
      if (content === undefined || content === "none" || content === "normal") continue;
      const attribute = content.match(/^attr\(([\w:-]+)\)$/i);
      if (attribute) content = element.getAttribute(attribute[1] || "") || "";
      else if (
        (content.startsWith('"') && content.endsWith('"')) ||
        (content.startsWith("'") && content.endsWith("'"))
      ) {
        content = content.slice(1, -1);
      } else {
        continue;
      }
      content = collapse(content);
      const style = this.accountStyle(
        ensureInheritedTextColor(
          declarations.size ? JSON.stringify(Object.fromEntries(declarations)) : undefined,
        ),
      );
      if (!content && !hasVisualOrGeometryStyle(style)) continue;
      if (depth + 1 > HTML_LIMITS.depth) {
        this.state.truncatedDepth = true;
        continue;
      }
      const id = reserveLayer(this.state, content ? "text" : "surface");
      if (!id) continue;
      layers.push(
        content
          ? node(id, "text", { text: content, variant: "body", ...(style ? { css: style } : {}) })
          : node(id, "surface", { css: style || "{}", label: `::${pseudo}` }),
      );
    }
    return layers;
  }

  private preservePseudoOverflow(mapped: TemplateNode): TemplateNode {
    if (mapped.type !== "surface") return mapped;
    let declarations: Record<string, string> = {};
    try {
      declarations = JSON.parse(String(mapped.props.css || "{}")) as Record<string, string>;
    } catch {
      return mapped;
    }
    if (
      "overflow" in declarations ||
      "overflow-x" in declarations ||
      "overflow-y" in declarations
    ) {
      return mapped;
    }
    declarations.overflow = "visible";
    const css = JSON.stringify(declarations);
    const accountedCss = this.accountStyle(css);
    return {
      ...mapped,
      props: { ...mapped.props, css: accountedCss || css },
    };
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
    if (this.unsupportedSvgs > 0)
      warnings.push(
        `${this.unsupportedSvgs} SVG artwork item${this.unsupportedSvgs === 1 ? " was" : "s were"} skipped or sanitized because of unsafe content or the ${HTML_LIMITS.svgBytes / 1024} KiB per-item / ${HTML_LIMITS.svgTotalBytes / 1024} KiB total SVG limit.`,
      );
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
    if (this.stylesTruncated) {
      warnings.push("Some source styling was omitted after reaching the editable CSS size limit.");
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
      children.push(node(id, "text", { text, variant: "body", css: '{"color":"inherit"}' }));
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
    const hint = sourceHint(element);
    const region: HtmlRegion = {
      id: `region-${index}`,
      tag,
      label: "",
      ...(hint ? { sourceHint: hint } : {}),
      ...(this.sourcePaths.get(element)
        ? { sourcePath: [...(this.sourcePaths.get(element) as number[])] }
        : {}),
      depth: sourceDepth,
      layers: 0,
      children: element.children.length,
    };
    this.regionEntries.push(region);
    const context: string[] = [];
    let ancestor = element.parentElement;
    while (ancestor && ancestor !== this.body.parentElement) {
      const style = this.styleForElement(ancestor, false);
      if (style) context.unshift(style);
      if (ancestor === this.body) break;
      ancestor = ancestor.parentElement;
    }
    this.contextByRegion.set(region.id, context);

    let mapped = this.mapTag(element, tag, depth, sourceDepth);
    const pseudoLayers = this.pseudoLayers(element, depth);
    if (pseudoLayers.length > 0) {
      if (!mapped) {
        const id = reserveLayer(this.state, "stack");
        if (id)
          mapped = node(id, "stack", { gap: 0, label: sourceLabel(element) || tag }, pseudoLayers);
      } else {
        mapped = this.preservePseudoOverflow(mapped);
        mapped = { ...mapped, children: [...(mapped.children ?? []), ...pseudoLayers] };
      }
    }
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
    if (tag === "svg") {
      if (element.classList.contains("sprite") || element.querySelector("symbol")) return null;
      const result = safeSvg(
        element,
        this.sourceDocument,
        this.svgStylesheets,
        this.htmlStylesheets,
        HTML_LIMITS.svgBytes,
      );
      if (!result) {
        this.unsupportedSvgs += 1;
        return null;
      }
      const size = byteLength(result.svg);
      if (this.svgBytes + size > HTML_LIMITS.svgTotalBytes) {
        this.unsupportedSvgs += 1;
        return null;
      }
      this.svgBytes += size;
      if (result.rejected) this.unsupportedSvgs += 1;
      return this.leaf(
        "image",
        {
          src: svgDataUrl(result.svg),
          alt: collapse(element.getAttribute("aria-label") || element.getAttribute("role") || ""),
          fit: "contain",
          h: 180,
        },
        element,
      );
    }
    if (element.children.length === 0 && collapse(element.textContent || "") === "") {
      const previewStyle = htmlStyle(element, this.htmlStylesheets);
      if (hasVisualOrGeometryStyle(previewStyle)) {
        const id = reserveLayer(this.state, "surface");
        if (!id) return null;
        const style = this.styleForElement(element);
        return node(id, "surface", {
          css: style || "{}",
          label: sourceLabel(element) || "Artwork",
        });
      }
    }
    if (HEADING_DISPLAY.has(tag)) return this.textLayer(element, "display");
    if (HEADING_TITLE.has(tag)) return this.textLayer(element, "title");
    if (tag === "span" && element.children.length > 0) {
      return this.container(element, "stack", depth, sourceDepth, 0);
    }
    if (tag === "span") return this.textLayer(element, "inline");
    if (tag === "p" || tag === "label") return this.textLayer(element, "body");
    if (tag === "small") return this.textLayer(element, "body", "muted");
    if (tag === "a") return this.textLayer(element, "body", "accent");
    if (tag === "button") {
      // Text-labeled navigation actions and short pressed-state controls have
      // an RCL Button mapping. Preserve compound content as custom layers until
      // a matching component can own its layout and behavior.
      const label = collapse(element.textContent);
      const labelWords = label ? label.split(/\s+/).length : 0;
      const navigationalAction = Boolean(element.closest("nav") && label);
      const shortPressedAction = element.hasAttribute("aria-pressed") && labelWords <= 3;
      if (element.children.length > 0 && !navigationalAction && !shortPressedAction) {
        return this.container(element, "stack", depth, sourceDepth, 0);
      }
      if (!label && element.children.length > 0) {
        return this.container(element, "stack", depth, sourceDepth, 0);
      }
      return this.buttonLayer(element, depth, sourceDepth);
    }
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
    if (tag === "hr") return this.leaf("divider", {}, element);
    if (tag === "br") return this.leaf("spacer", { h: 12 }, element);
    // Unknown element: structural when it has element children, copy otherwise.
    return element.children.length > 0
      ? this.container(element, "stack", depth, sourceDepth, 14)
      : this.textLayer(element, "body");
  }

  private textLayer(
    element: Element,
    variant: "display" | "title" | "body" | "inline",
    tone?: "muted" | "accent",
  ): TemplateNode | null {
    const text = collapse(element.textContent);
    if (!text) return null;
    const id = reserveLayer(this.state, "text");
    if (!id) return null;
    const style = ensureInheritedTextColor(this.styleForElement(element));
    return node(id, "text", {
      ...(tone ? { text, variant, tone } : { text, variant }),
      ...(style ? { css: style } : {}),
    });
  }

  private buttonLayer(element: Element, depth: number, sourceDepth: number): TemplateNode | null {
    const children =
      element.children.length > 0 ? this.mapChildren(element, depth + 1, sourceDepth + 1) : [];
    const text = children.length > 0 ? "" : collapse(element.textContent);
    if (!text && children.length === 0) return null;
    const id = reserveLayer(this.state, "button");
    if (!id) return null;
    const style = ensureInheritedTextColor(this.styleForElement(element));
    const pressed = element.getAttribute("aria-pressed");
    return node(
      id,
      "button",
      {
        text,
        variant: "secondary",
        ...(pressed === "true" ? { pressed: true } : pressed === "false" ? { pressed: false } : {}),
        ...(style ? { css: style } : {}),
      },
      children.length > 0 ? children : undefined,
    );
  }

  private leaf(
    type: TemplateNodeType,
    props: Record<string, string | number | boolean>,
    element?: Element,
  ): TemplateNode | null {
    const id = reserveLayer(this.state, type);
    if (!id) return null;
    const style = element ? this.styleForElement(element) : undefined;
    return node(id, type, { ...props, ...(style ? { css: style } : {}) });
  }

  private container(
    element: Element,
    type: "stack" | "row",
    depth: number,
    sourceDepth: number,
    gap: number,
  ): TemplateNode | null {
    const sourceStyle = this.styleForElement(element);
    let style = sourceStyle;
    if (sourceStyle) {
      try {
        const declarations = JSON.parse(sourceStyle) as Record<string, unknown>;
        if (
          declarations.display === undefined &&
          BLOCK_CONTAINER_TAGS.has(element.tagName.toLowerCase())
        ) {
          declarations.display = "block";
          style = JSON.stringify(declarations);
        }
      } catch {
        /* Preserve a malformed source style as-is; validation handles it. */
      }
    } else if (BLOCK_CONTAINER_TAGS.has(element.tagName.toLowerCase())) {
      style = JSON.stringify({ display: "block" });
    }
    let layout: Record<string, unknown> = {};
    try {
      layout = style ? (JSON.parse(style) as Record<string, unknown>) : {};
    } catch {
      /* Keep the semantic fallback when an authored style cannot be parsed. */
    }
    const display = layout.display;
    const direction = layout["flex-direction"];
    const layoutType: "stack" | "row" =
      display === "flex" && direction !== "column" && direction !== "column-reverse" ? "row" : type;
    const id = reserveLayer(this.state, layoutType);
    if (!id) return null;
    const children = this.mapChildren(element, depth + 1, sourceDepth + 1);
    if (children.length === 0) {
      // The reserved slot produced nothing; release it so empty wrappers do
      // not consume the layer budget.
      this.state.layers -= 1;
      return null;
    }
    const label = sourceLabel(element);
    return node(
      id,
      layoutType,
      { gap: sourceStyle ? 0 : gap, ...(label ? { label } : {}), ...(style ? { css: style } : {}) },
      children,
    );
  }
}
