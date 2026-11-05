import { publicationPath, publicationXml, xmlChildren } from "./publicationXml";
import { sanitizeDocument } from "./sanitizeDocument";
import { MAX_PUBLICATION_XML } from "./publicationData";
export interface EbookChapter { path: string; title: string }
export interface EbookLink { chapter: number; fragment: string; title: string; depth: number }
export interface EbookData { title: string; creator: string; chapters: EbookChapter[]; contents: EbookLink[]; fixed: boolean; parts: Record<string, Uint8Array> }
export const epubText = (bytes?: Uint8Array) => bytes ? new TextDecoder("utf-8", { fatal: true }).decode(bytes) : "";
function requirePart(parts: EbookData["parts"], path: string) { if (!parts[path]) throw Error("invalid"); return publicationXml(epubText(parts[path])); }
export function ebookLink(book: Pick<EbookData, "chapters">, base: string, href: string): Pick<EbookLink, "chapter" | "fragment"> | null {
  const path = publicationPath(base, href), chapter = book.chapters.findIndex(item => item.path === path);
  if (chapter < 0) return null;
  let fragment = href.includes("#") ? href.slice(href.indexOf("#") + 1) : "";
  try { fragment = decodeURIComponent(fragment); } catch { return null; }
  return { chapter, fragment };
}
export function parseEbook(parts: EbookData["parts"]): EbookData {
  if (epubText(parts.mimetype).trim() !== "application/epub+zip") throw Error("invalid");
  if (parts["META-INF/encryption.xml"]) throw Error("encrypted");
  const container = requirePart(parts, "META-INF/container.xml"), packagePath = xmlChildren(container, "rootfile").find(node => node.getAttribute("media-type") === "application/oebps-package+xml")?.getAttribute("full-path");
  if (!packagePath || !publicationPath("container.xml", packagePath)) throw Error("invalid");
  const packageDoc = requirePart(parts, packagePath), manifest = new Map<string, { path: string; type: string; properties: string[] }>();
  for (const item of xmlChildren(packageDoc, "item")) {
    const id = item.getAttribute("id"), href = item.getAttribute("href"), path = href ? publicationPath(packagePath, href) : null;
    if (!id || manifest.has(id)) throw Error("invalid");
    // EPUB 3 may declare remote artwork or styles. Keep the local spine usable
    // while those resources stay unavailable; a remote spine item still fails.
    manifest.set(id, { path: path ?? "", type: item.getAttribute("media-type") ?? "", properties: (item.getAttribute("properties") ?? "").split(/\s+/) });
  }
  const spine = xmlChildren(packageDoc, "spine")[0], references = spine ? xmlChildren(spine, "itemref") : [];
  if (!references.length) throw Error("invalid"); if (references.length > 300) throw new RangeError();
  const chapters: EbookChapter[] = [], seen = new Set<string>();
  for (const reference of references) {
    const item = manifest.get(reference.getAttribute("idref") ?? "");
    if (!item || !["application/xhtml+xml", "text/html"].includes(item.type) || !parts[item.path] || seen.has(item.path)) throw Error("invalid");
    // Keep nonlinear spine entries addressable; they often contain footnotes.
    const doc = requirePart(parts, item.path), title = xmlChildren(doc, "h1")[0]?.textContent || xmlChildren(doc, "title")[0]?.textContent || String(chapters.length + 1);
    chapters.push({ path: item.path, title: title.trim().slice(0, 300) }); seen.add(item.path);
  }
  const book: EbookData = { title: xmlChildren(packageDoc, "title")[0]?.textContent.trim().slice(0, 500) ?? "", creator: xmlChildren(packageDoc, "creator").map(node => node.textContent.trim()).filter(Boolean).join(", ").slice(0, 1000), chapters, contents: [], fixed: xmlChildren(packageDoc, "meta").some(node => node.getAttribute("property") === "rendition:layout" && node.textContent.trim() === "pre-paginated"), parts };
  const nav = Array.from(manifest.values()).find(item => item.properties.includes("nav"));
  if (nav && parts[nav.path]) {
    const doc = requirePart(parts, nav.path), toc = xmlChildren(doc, "nav").find(node => (node.getAttributeNS("http://www.idpf.org/2007/ops", "type") ?? node.getAttribute("epub:type") ?? "").split(/\s+/).includes("toc"));
    if (toc) for (const anchor of xmlChildren(toc, "a")) {
      const target = ebookLink(book, nav.path, anchor.getAttribute("href") ?? "");
      if (!target) continue;
      let depth = 0, parent = anchor.parentElement; while (parent && parent !== toc) { if (parent.localName === "ol") depth++; parent = parent.parentElement; }
      book.contents.push({ ...target, title: anchor.textContent.trim().slice(0, 300) || chapters[target.chapter]?.title || "", depth: Math.min(3, Math.max(0, depth - 1)) });
    }
  }
  if (!book.contents.length && spine) {
    const ncx = manifest.get(spine.getAttribute("toc") ?? "");
    if (ncx && parts[ncx.path]) for (const point of xmlChildren(requirePart(parts, ncx.path), "navPoint")) {
      const target = ebookLink(book, ncx.path, xmlChildren(point, "content")[0]?.getAttribute("src") ?? ""); if (!target) continue;
      let depth = 0, parent = point.parentElement; while (parent) { if (parent.localName === "navPoint") depth++; parent = parent.parentElement; }
      book.contents.push({ ...target, title: xmlChildren(point, "navLabel")[0]?.textContent.trim().slice(0, 300) || chapters[target.chapter]?.title || "", depth: Math.min(depth, 3) });
    }
  }
  if (book.contents.length > 2000) throw new RangeError();
  if (!book.contents.length) book.contents = chapters.map((chapter, index) => ({ chapter: index, fragment: "", title: chapter.title, depth: 0 }));
  return book;
}
export function renderEbookChapter(book: EbookData, index: number, prefix: string, tableLabel: string) {
  const chapter = book.chapters[index]; if (!chapter) throw Error("invalid");
  const doc = publicationXml(epubText(book.parts[chapter.path])), body = xmlChildren(doc, "body")[0];
  if (!body) throw Error("invalid");
  for (const active of Array.from(body.querySelectorAll("script,style,link,base,iframe,object,embed,audio,video,source,track,form,input,button,textarea,select,svg,math"))) active.remove();
  const urls: string[] = []; let omitted = false, imageBytes = 0;
  for (const node of Array.from(body.querySelectorAll("[id]"))) node.id = `${prefix}${encodeURIComponent(node.id)}`;
  for (const image of xmlChildren(body, "img")) {
    image.removeAttribute("srcset");
    const path = publicationPath(chapter.path, image.getAttribute("src") ?? ""), data = path ? book.parts[path] : null, extension = path?.split(".").pop()?.toLowerCase(), type = extension === "jpg" ? "jpeg" : extension;
    if (!data || !type || !["png", "jpeg", "gif", "webp"].includes(type) || data.byteLength > 3 * 1024 * 1024 || (imageBytes += data.byteLength) > 12 * 1024 * 1024) { image.remove(); omitted = true; continue; }
    const url = URL.createObjectURL(new Blob([data as BlobPart], { type: `image/${type}` })); urls.push(url); image.setAttribute("src", url);
  }
  for (const link of xmlChildren(body, "a")) {
    const href = link.getAttribute("href") ?? "", target = ebookLink(book, chapter.path, href);
    if (target) link.setAttribute("href", `#wc-epub=${encodeURIComponent(JSON.stringify(target))}`);
    else if (!/^(https?:\/\/|mailto:)/i.test(href)) link.removeAttribute("href");
  }
  try {
    const html = sanitizeDocument(body.innerHTML, tableLabel, new Set(urls));
    if (new TextEncoder().encode(html).byteLength > MAX_PUBLICATION_XML) throw new RangeError("html");
    return { html, omitted, dispose: () => { urls.forEach(url => { URL.revokeObjectURL(url); }); } };
  }
  catch (error) { urls.forEach(url => { URL.revokeObjectURL(url); }); throw error; }
}
