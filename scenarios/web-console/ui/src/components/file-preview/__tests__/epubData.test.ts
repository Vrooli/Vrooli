import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ebookLink, parseEbook, renderEbookChapter, type EbookData } from "../epubData";
import { publicationXml } from "../publicationXml";
import { MAX_PUBLICATION_XML } from "../publicationData";
const revoke = vi.fn();
const encode = (text: string) => new TextEncoder().encode(text);
export function ebookParts(): EbookData["parts"] {
  return {
    mimetype: encode("application/epub+zip"),
    "META-INF/container.xml": encode('<container><rootfiles><rootfile full-path="OPS/book.opf" media-type="application/oebps-package+xml"/></rootfiles></container>'),
    "OPS/book.opf": encode('<package xmlns:dc="urn:dc"><metadata><dc:title>Reading carefully</dc:title><dc:creator>Test Author</dc:creator></metadata><manifest><item id="second" href="second.xhtml" media-type="application/xhtml+xml"/><item id="first" href="first.xhtml" media-type="application/xhtml+xml"/><item id="remote-art" href="https://preview-invalid.test/art.png" media-type="image/png"/><item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/></manifest><spine><itemref idref="first"/><itemref idref="second" linear="no"/></spine></package>'),
    "OPS/nav.xhtml": encode('<html xmlns:epub="http://www.idpf.org/2007/ops"><body><nav epub:type="toc"><ol><li><a href="first.xhtml">Introduction</a><ol><li><a href="second.xhtml#detail">Details</a></li></ol></li></ol></nav></body></html>'),
    "OPS/first.xhtml": encode('<html><head><title>First</title></head><body><h1 id="intro">First chapter</h1><p>Read <strong>this phrase</strong> carefully.</p><script>window.evil=1</script><form><input/></form><p style="background:url(https://evil.test/track)"><a href="second.xhtml#detail">Next detail</a><a href="javascript:alert(1)">Unsafe</a><a href="https://example.com">External</a></p><img src="image.png" alt="A sample"/><img src="https://evil.test/track"/><iframe src="https://evil.test"/></body></html>'),
    "OPS/second.xhtml": encode('<html><body><h1 id="detail">Second chapter</h1><p>More detail.</p><a href="first.xhtml#intro">Return</a></body></html>'),
    "OPS/image.png": new Uint8Array([137, 80, 78, 71]),
  };
}
beforeEach(() => { vi.stubGlobal("URL", URL); URL.createObjectURL = vi.fn().mockReturnValue("blob:owned"); URL.revokeObjectURL = revoke; });
afterEach(() => { vi.unstubAllGlobals(); });
describe("EPUB package model and safe reading content", () => {
  it("honors spine order including nonlinear notes and nested EPUB 3 navigation", () => {
    const book = parseEbook(ebookParts()); expect(book.title).toBe("Reading carefully"); expect(book.creator).toBe("Test Author"); expect(book.chapters.map(c => c.path)).toEqual(["OPS/first.xhtml", "OPS/second.xhtml"]);
    expect(book.contents).toEqual([{ chapter: 0, fragment: "", title: "Introduction", depth: 0 }, { chapter: 1, fragment: "detail", title: "Details", depth: 1 }]);
    expect(ebookLink(book, "OPS/first.xhtml", "second.xhtml#detail")).toEqual({ chapter: 1, fragment: "detail" }); expect(ebookLink(book, "OPS/first.xhtml", "https://evil.test/second.xhtml")).toBeNull();
  });
  it("uses EPUB 2 NCX labels when EPUB 3 navigation is absent", () => {
    const parts = ebookParts(); parts["OPS/book.opf"] = encode('<package><metadata/><manifest><item id="first" href="first.xhtml" media-type="application/xhtml+xml"/><item id="toc" href="toc.ncx" media-type="application/x-dtbncx+xml"/></manifest><spine toc="toc"><itemref idref="first"/></spine></package>'); parts["OPS/toc.ncx"] = encode('<ncx><navMap><navPoint><navLabel><text>NCX chapter</text></navLabel><content src="first.xhtml#intro"/></navPoint></navMap></ncx>'); expect(parseEbook(parts).contents[0]).toMatchObject({ title: "NCX chapter", fragment: "intro" });
  });
  it("reads common EPUB 2 XHTML declarations and text entities without resolving a DTD", () => {
    const parts = ebookParts(); parts["OPS/first.xhtml"] = encode('<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd"><html><body><h1>A&nbsp;chapter</h1><p>&copy; &lt;safe&gt;</p></body></html>');
    const book = parseEbook(parts), chapter = renderEbookChapter(book, 0, "private-", "Table"); expect(chapter.html).toContain("© &lt;safe&gt;"); expect(book.chapters[0]?.title).toBe("A\u00a0chapter"); chapter.dispose();
    expect(publicationXml('<html><body><pre><![CDATA[<!DOCTYPE html> <!ENTITY example>]]></pre></body></html>').getElementsByTagName("pre")[0]?.textContent).toBe('<!DOCTYPE html> <!ENTITY example>');
  });
  it("removes active/remote content, scopes IDs, rewrites internal links, and releases owned images", () => {
    const book = parseEbook(ebookParts()), result = renderEbookChapter(book, 0, "private-", "Book table"), doc = new DOMParser().parseFromString(result.html, "text/html");
    expect(doc.querySelector("script,iframe,form,input,[style]")).toBeNull(); expect(doc.getElementById("private-intro")).not.toBeNull(); expect(doc.querySelectorAll("img")).toHaveLength(1); expect(doc.querySelector("img")?.getAttribute("src")).toBe("blob:owned"); expect(result.omitted).toBe(true); expect(doc.querySelector('a[href^="#wc-epub="]')).not.toBeNull(); expect(doc.querySelector('a[href="https://example.com"]')?.getAttribute("rel")).toContain("noopener"); expect(doc.querySelector('a[href^="javascript:"]')).toBeNull(); result.dispose(); expect(revoke).toHaveBeenCalledWith("blob:owned");
  });
  it("rejects missing spine members, malformed XML, encrypted content and oversized chapter counts", () => {
    const missing = ebookParts(); delete missing["OPS/second.xhtml"]; expect(() => parseEbook(missing)).toThrow("invalid"); const invalid = ebookParts(); invalid["OPS/book.opf"] = encode("<package>"); expect(() => parseEbook(invalid)).toThrow("invalid"); const encrypted = ebookParts(); encrypted["META-INF/encryption.xml"] = encode("<encryption/>"); expect(() => parseEbook(encrypted)).toThrow("encrypted"); const many = ebookParts(); many["OPS/book.opf"] = encode('<package><manifest><item id="c" href="first.xhtml" media-type="application/xhtml+xml"/></manifest><spine>' + '<itemref idref="c"/>'.repeat(301) + '</spine></package>'); expect(() => parseEbook(many)).toThrow(RangeError);
  });
  it("bounds the sanitized chapter and releases images when it cannot be retained", () => {
    const book = parseEbook(ebookParts()); book.parts["OPS/first.xhtml"] = encode('<html><body><img src="image.png"/><p>' + "x".repeat(MAX_PUBLICATION_XML) + '</p></body></html>');
    expect(() => renderEbookChapter(book, 0, "private-", "Table")).toThrow(RangeError); expect(revoke).toHaveBeenCalledWith("blob:owned");
  });
});
