import type { HtmlRegion } from "./htmlImportTypes";

type DocumentParser = (source: string) => Document;

function elementAtPath(root: Element, path: number[] | undefined): Element | null {
  if (!path) return null;
  let current: Element = root;
  for (const index of path) {
    const next = current.children.item(index);
    if (!next) return null;
    current = next;
  }
  return current;
}

const SOURCE_REMOVED_TAGS = ["script", "iframe", "object", "embed", "form", "base"] as const;

function safeCss(source: string): string {
  return source
    .replace(/@import\s+[^;]+;?/gi, "")
    .replace(/url\([^)]*\)/gi, (value) =>
      /^url\(\s*['"]?data:image\//i.test(value) ? value : "none",
    )
    .replace(/<\/style/gi, "\\3C /style");
}

export function buildSourcePreview(
  source: string,
  parseDocument: DocumentParser,
  regionId: string | null,
  regions: HtmlRegion[],
  stylesheets: string[],
): string {
  const document = parseDocument(source);
  const inlineStyles = Array.from(document.querySelectorAll("style"), (style) => style.textContent);
  for (const element of Array.from(document.querySelectorAll(SOURCE_REMOVED_TAGS.join(",")))) {
    element.remove();
  }
  for (const element of Array.from(document.querySelectorAll("style, link"))) element.remove();

  const region = regions.find((candidate) => candidate.id === regionId);
  let content: Element | null = null;
  if (region) {
    content = elementAtPath(document.body, region.sourcePath);
    // Keep older or externally constructed ParsedHtml values usable. Source
    // paths are the identity; tag matching is only a compatibility fallback.
    content ??= Array.from(document.body.querySelectorAll(region.tag))[0] ?? null;
  }

  for (const element of Array.from(document.querySelectorAll("*"))) {
    for (const attribute of Array.from(element.attributes)) {
      const name = attribute.name.toLowerCase();
      const value = attribute.value.trim();
      if (name.startsWith("on") || name === "srcdoc") {
        element.removeAttribute(attribute.name);
      } else if (name.endsWith("href") && !value.startsWith("#")) {
        element.removeAttribute(attribute.name);
      } else if ((name === "src" || name === "srcset") && !value.startsWith("data:image/")) {
        element.removeAttribute(attribute.name);
      } else if (name === "style") {
        element.setAttribute(attribute.name, safeCss(value));
      }
    }
  }

  const styles = [
    ...inlineStyles.map(safeCss).filter(Boolean),
    ...stylesheets.map(safeCss).filter(Boolean),
    ".rail-wrap{padding:0!important}.rail-wrap:after{display:none!important}.rail{--s:.88!important;gap:0!important;overflow:hidden!important;padding:0!important}",
  ].join("\n");
  const style = document.createElement("style");
  style.textContent = styles;
  document.head.replaceChildren(style);
  const meta = document.createElement("meta");
  meta.setAttribute("http-equiv", "Content-Security-Policy");
  meta.setAttribute(
    "content",
    "default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:; object-src 'none'; base-uri 'none'; form-action 'none'; connect-src 'none'; frame-src 'none'",
  );
  document.head.prepend(meta);
  const viewport = document.createElement("meta");
  viewport.setAttribute("name", "viewport");
  viewport.setAttribute("content", "width=device-width, initial-scale=1");
  document.head.append(viewport);
  document.title = region?.sourceHint || document.title || "Imported source";

  if (content) {
    const rail = document.createElement("div");
    rail.className = "rail-wrap";
    const inner = document.createElement("div");
    inner.className = "rail";
    const sprite = document.querySelector("svg.sprite");
    if (sprite) inner.append(sprite.cloneNode(true));
    inner.append(content.cloneNode(true));
    rail.append(inner);
    document.body.replaceChildren(rail);
  }
  return "<!doctype html>" + document.documentElement.outerHTML;
}
