import { htmlStyle, staticCssRules, type HtmlCssRule } from "./htmlImportStyles";

/** SVG-only CSS rule retained while sanitizing imported vector artwork. */
export type SvgCssRule = { selector: string; declarations: Map<string, string> };

const SVG_ELEMENTS = new Set([
  "svg",
  "g",
  "path",
  "circle",
  "ellipse",
  "rect",
  "line",
  "polyline",
  "polygon",
  "defs",
  "lineargradient",
  "radialgradient",
  "stop",
  "clippath",
  "mask",
  "title",
  "desc",
]);
// SVG layers are self-contained image data. Source styles are resolved onto
// SVG elements as presentation attributes; CSS itself is never embedded.
const SVG_ATTRIBUTES = new Set([
  "id",
  "viewbox",
  "preserveaspectratio",
  "width",
  "height",
  "x",
  "y",
  "x1",
  "x2",
  "y1",
  "y2",
  "cx",
  "cy",
  "r",
  "rx",
  "ry",
  "d",
  "points",
  "fill",
  "fill-rule",
  "fill-opacity",
  "stroke",
  "stroke-width",
  "stroke-linecap",
  "stroke-linejoin",
  "stroke-dasharray",
  "stroke-dashoffset",
  "stroke-opacity",
  "opacity",
  "transform",
  "offset",
  "stop-color",
  "stop-opacity",
  "clip-path",
  "mask",
  "gradientunits",
  "gradienttransform",
  "spreadmethod",
  "clippathunits",
  "maskunits",
  "maskcontentunits",
  "color",
  "display",
  "visibility",
  "font-size",
  "font-family",
  "font-weight",
  "font-style",
  "text-anchor",
  "dominant-baseline",
  "vector-effect",
  "mix-blend-mode",
  "filter",
]);
const SVG_TAG_CASE: Record<string, string> = {
  lineargradient: "linearGradient",
  radialgradient: "radialGradient",
  clippath: "clipPath",
};
const SVG_ATTRIBUTE_CASE: Record<string, string> = {
  viewbox: "viewBox",
  preserveaspectratio: "preserveAspectRatio",
  gradientunits: "gradientUnits",
  gradienttransform: "gradientTransform",
  spreadmethod: "spreadMethod",
  clippathunits: "clipPathUnits",
  maskunits: "maskUnits",
  maskcontentunits: "maskContentUnits",
  fillrule: "fillRule",
  fillopacity: "fillOpacity",
  strokewidth: "strokeWidth",
  strokelinecap: "strokeLinecap",
  strokelinejoin: "strokeLinejoin",
  strokedasharray: "strokeDasharray",
  strokedashoffset: "strokeDashoffset",
  strokeopacity: "strokeOpacity",
  stopcolor: "stopColor",
  stopopacity: "stopOpacity",
};



function safeSvgDeclarations(source: string): Map<string, string> {
  const declarations = new Map<string, string>();
  for (const part of source.split(";")) {
    const colon = part.indexOf(":");
    if (colon < 1) continue;
    const name = part.slice(0, colon).trim().toLowerCase();
    const value = part.slice(colon + 1).trim();
    if (!SVG_ATTRIBUTES.has(name) || !value || /expression\s*\(|javascript:|@import/i.test(value))
      continue;
    const localReference = /^url\(\s*(['"]?)#[-\w:.]+\1\s*\)$/i.test(value);
    const referenceProperty = /^(clip-path|mask|filter|fill|stroke)$/.test(name);
    if (/url\s*\(/i.test(value) && !(localReference && referenceProperty)) continue;
    declarations.set(name, value);
  }
  return declarations;
}

export function parseSvgCssRules(stylesheets: string[]): SvgCssRule[] {
  const rules: SvgCssRule[] = [];
  for (const stylesheet of stylesheets) {
    // Parse once per document; at-rules and malformed declarations are ignored.
    for (const match of staticCssRules(stylesheet).matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
      const selectorList = (match[1] || "").trim();
      const declarations = safeSvgDeclarations(match[2] || "");
      if (!declarations.size) continue;
      for (const selector of selectorList.split(",")) {
        const cleanSelector = selector.trim();
        if (cleanSelector) rules.push({ selector: cleanSelector, declarations });
      }
    }
  }
  return rules;
}

function svgCssDeclarations(element: Element, rules: SvgCssRule[]): Map<string, string> {
  const declarations = new Map<string, string>();
  for (const rule of rules) {
    try {
      if (element.matches(rule.selector)) {
        for (const [name, value] of rule.declarations) declarations.set(name, value);
      }
    } catch {
      /* Invalid or unsupported selector is simply ignored. */
    }
  }
  for (const [name, value] of safeSvgDeclarations(element.getAttribute("style") || "")) {
    declarations.set(name, value);
  }
  return declarations;
}

export function safeSvg(
  source: Element,
  document: Document,
  stylesheets: SvgCssRule[],
  htmlStylesheets: HtmlCssRule[],
  maxBytes: number,
  resolving = new Set<string>(),
): { svg: string; rejected: boolean } | null {
  let rejected = false;
  const variables = new Map<string, string>();
  const ancestry: Element[] = [];
  for (let current: Element | null = source; current; current = current.parentElement) {
    ancestry.unshift(current);
  }
  for (const element of ancestry) {
    const style = htmlStyle(element, htmlStylesheets);
    if (!style) continue;
    try {
      for (const [name, value] of Object.entries(JSON.parse(style) as Record<string, string>)) {
        if (name.startsWith("--")) variables.set(name, value);
      }
    } catch {
      /* Keep mapping the vector if ancestor presentation is malformed. */
    }
  }
  const resolveVariables = (value: string, seen = new Set<string>()): string =>
    value.replace(
      /var\((--[\w-]+)(?:\s*,\s*([^()]+))?\)/g,
      (match, name: string, fallback?: string) => {
        if (seen.has(name)) return fallback?.trim() || match;
        const resolved = variables.get(name);
        if (resolved === undefined) return fallback?.trim() || match;
        const next = new Set(seen);
        next.add(name);
        return resolveVariables(resolved, next);
      },
    );
  let inheritedHtmlColor = "#000000";
  for (const element of ancestry) {
    const style = htmlStyle(element, htmlStylesheets);
    if (!style) continue;
    try {
      const color = (JSON.parse(style) as Record<string, string>).color;
      if (color) {
        const resolvedColor = resolveVariables(color);
        if (!/^inherit$|^currentcolor$/i.test(resolvedColor)) inheritedHtmlColor = resolvedColor;
      }
    } catch {
      /* Keep the last safe inherited color if presentation is malformed. */
    }
  }
  const declarationsFor = (element: Element): Map<string, string> => {
    const declarations = svgCssDeclarations(element, stylesheets);
    for (const [name, value] of safeSvgDeclarations(element.getAttribute("style") || "")) {
      declarations.set(name, value);
    }
    return declarations;
  };
  const clone = (element: Element, inheritedColor = "#000000"): Element | null => {
    const tag = element.localName.toLowerCase();
    if (!SVG_ELEMENTS.has(tag)) {
      rejected = true;
      return null;
    }
    const out = document.createElementNS("http://www.w3.org/2000/svg", SVG_TAG_CASE[tag] || tag);
    const declarations = declarationsFor(element);
    const sourceColor =
      element.getAttribute("color") || declarations.get("color") || inheritedColor;
    const sourceColorValue = resolveVariables(sourceColor);
    const resolvedColor = /^inherit$|^currentcolor$/i.test(sourceColorValue)
      ? inheritedColor
      : sourceColorValue;
    for (const attribute of Array.from(element.attributes)) {
      const name = attribute.name.toLowerCase();
      let value = attribute.value.trim();
      if (
        /^on/i.test(name) ||
        ["href", "xlink:href", "src"].includes(name) ||
        /javascript:|data:|https?:|@import/i.test(value)
      ) {
        rejected = true;
        continue;
      }
      if (!SVG_ATTRIBUTES.has(name)) continue;
      value = resolveVariables(value).replace(/currentcolor/gi, resolvedColor);
      if (tag === "svg" && /^(fill|stroke)$/.test(name) && /^inherit$/i.test(value)) {
        value = resolvedColor;
      }
      if (/var\(--[\w-]+\)/.test(value)) {
        rejected = true;
        continue;
      }
      if (name === "id" && !/^[\w:.-]{1,128}$/.test(value)) {
        rejected = true;
        continue;
      }
      if (name === "clip-path" || name === "mask") {
        if (!/^url\(#[\w:.-]+\)$/.test(value)) {
          rejected = true;
          continue;
        }
      }
      out.setAttribute(SVG_ATTRIBUTE_CASE[name] || name, value);
    }
    // CSS declarations outrank SVG presentation attributes. The declaration
    // collector preserves stylesheet order and applies inline style last.
    for (const [name, rawValue] of declarations) {
      let value = resolveVariables(rawValue).replace(/currentcolor/gi, resolvedColor);
      if (tag === "svg" && /^(fill|stroke)$/.test(name) && /^inherit$/i.test(value)) {
        value = resolvedColor;
      }
      if (/var\(--[\w-]+\)/.test(value)) {
        rejected = true;
        continue;
      }
      out.setAttribute(SVG_ATTRIBUTE_CASE[name] || name, value);
    }
    for (const child of Array.from(element.children)) {
      if (child.localName.toLowerCase() === "use") {
        const href =
          child.getAttribute("href") ||
          child.getAttributeNS("http://www.w3.org/1999/xlink", "href") ||
          "";
        const id = href.startsWith("#") ? href.slice(1) : "";
        if (!id || resolving.has(id)) {
          rejected = true;
          continue;
        }
        const referenced = document.getElementById(id);
        if (!referenced || referenced.localName.toLowerCase() !== "symbol") {
          rejected = true;
          continue;
        }
        if (!out.hasAttribute("viewBox") && referenced.hasAttribute("viewBox")) {
          out.setAttribute("viewBox", referenced.getAttribute("viewBox") || "");
        }
        resolving.add(id);
        const group = document.createElementNS("http://www.w3.org/2000/svg", "g");
        for (const nested of Array.from(referenced.children)) {
          const safeChild = clone(nested, resolvedColor);
          if (safeChild) group.append(safeChild);
        }
        resolving.delete(id);
        out.append(group);
      } else {
        const safeChild = clone(child, resolvedColor);
        if (safeChild) out.append(safeChild);
      }
    }
    return out;
  };
  if (source.localName.toLowerCase() !== "svg") return null;
  const result = clone(source, inheritedHtmlColor);
  if (!result) return null;
  const serialized = new XMLSerializer().serializeToString(result);
  if (byteLength(serialized) > maxBytes) return null;
  return { svg: serialized, rejected };
}

export function svgDataUrl(svg: string): string {
  const bytes = new TextEncoder().encode(svg);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return `data:image/svg+xml;base64,${btoa(binary)}`;
}


function byteLength(source: string): number {
  if (typeof TextEncoder !== "undefined") return new TextEncoder().encode(source).byteLength;
  return new Blob([source]).size;
}
