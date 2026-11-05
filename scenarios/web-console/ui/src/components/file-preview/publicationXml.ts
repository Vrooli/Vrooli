// Namespace-prefix independent XML helpers; caller supplies bounded package parts.
export function stripPublicationDoctype(source: string): string {
  // EPUB 2 commonly declares an external XHTML DTD. Discard the declaration
  // without resolving it; internal subsets and non-HTML doctypes stay rejected.
  // Comments and CDATA are matched first so technical prose stays byte-for-byte
  // intact even when it explains DTDs or entity syntax.
  return source.replace(/<!--[\s\S]*?-->|<!\[CDATA\[[\s\S]*?\]\]>|<!DOCTYPE(?:"[^"]*"|'[^']*'|[^'">])*?>|<!ENTITY\b/gi, token => {
    if (token.startsWith("<!--") || token.startsWith("<![CDATA[")) return token;
    if (/^<!DOCTYPE\s+html(?:\s+(?:PUBLIC\s+(?:"[^"]*"|'[^']*')\s+(?:"[^"]*"|'[^']*')|SYSTEM\s+(?:"[^"]*"|'[^']*')))?\s*>$/i.test(token)) return "";
    throw Error("invalid");
  });
}
export function publicationXml(source: string): Document {
  let clean = stripPublicationDoctype(source);
  if (/<(?:[a-z]+:)?html[\s>]/i.test(clean)) {
    const decoder = document.createElement("textarea");
    clean = clean.replace(/<!--[\s\S]*?-->|<!\[CDATA\[[\s\S]*?\]\]>|&([a-z][a-z\d]+);/gi, (entity: string, name: string | undefined) => {
      if (!name) return entity;
      if (["amp", "lt", "gt", "quot", "apos"].includes(name)) return entity;
      decoder.innerHTML = entity;
      return decoder.value === entity ? entity : Array.from(decoder.value).map(char => `&#${String(char.codePointAt(0))};`).join("");
    });
  }
  const doc = new DOMParser().parseFromString(clean, "application/xml");
  if (doc.getElementsByTagName("parsererror").length || doc.doctype) throw Error("invalid");
  return doc;
}
export function xmlChildren(root: Document | Element, name: string): Element[] {
  return Array.from(root.getElementsByTagNameNS("*", name));
}
export function publicationPath(base: string, target: string): string | null {
  if (!target || /^[a-z][a-z\d+.-]*:|^\/|\\/i.test(target)) return null;
  try {
    const url = new URL(target, `https://publication.invalid/${base}`);
    const path = decodeURIComponent(url.pathname.slice(1));
    return url.origin === "https://publication.invalid" && !path.includes("\\") && !Array.from(path).some(char => char.charCodeAt(0) < 32) ? path : null;
  } catch { return null; }
}
export function publicationParagraphs(root: Document | Element): string {
  return xmlChildren(root, "p").map(p => xmlChildren(p, "t").map(t => t.textContent).join("")).filter(Boolean).join("\n");
}
