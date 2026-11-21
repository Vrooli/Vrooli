import { describe, expect, it } from "vitest";

import { readFileSync } from "node:fs";
import { join } from "node:path";
import {
  EXPLAINER_BASE_CSS,
  EXPLAINER_CSP,
  INLINE_FILE_LIMIT,
  OPEN_PLACEHOLDER,
  buildExplainerDocument,
  findReferences,
  planInlining,
  type ExplainerAttachment,
  type ExplainerDocumentOptions,
} from "./document";

// Vitest does not hand CSS to `?raw` imports, so read the published file directly.
const kitCss = readFileSync(join(process.cwd(), "src/features/packet/explainer/kit/kit.css"), "utf8");

const attachments: ExplainerAttachment[] = [
  { name: "set-a.png", folder: false, sizeBytes: 1000, mediaType: "image/png", entries: [] },
  { name: "huge.mp4", folder: false, sizeBytes: INLINE_FILE_LIMIT + 1, mediaType: "video/mp4", entries: [] },
  { name: "renders", folder: true, sizeBytes: 0, mediaType: "", entries: [{ path: "walk/01.png", sizeBytes: 2000, mediaType: "image/png" }] },
];

const options = (overrides: Partial<ExplainerDocumentOptions> = {}): ExplainerDocumentOptions => ({
  kitCss: "@layer hub-kit{body{margin:0}}",
  kitJs: "window.hubKit={};",
  bridgeJs: "window.hub={};",
  theme: "dark",
  dataUrls: new Map(),
  deferred: [],
  ...overrides,
});

const parse = (html: string) => new DOMParser().parseFromString(html, "text/html");

describe("buildExplainerDocument", () => {
  it("puts the Content-Security-Policy first in the head, before anything the page wrote", () => {
    const doc = parse(buildExplainerDocument("<script>alert(1)</script><title>x</title><p>hi</p>", options()));
    const first = doc.head.firstElementChild;
    expect(first?.tagName).toBe("META");
    expect(first?.getAttribute("http-equiv")).toBe("Content-Security-Policy");
    expect(first?.getAttribute("content")).toBe(EXPLAINER_CSP);
    expect(doc.head.querySelector("script:not([data-hub])")).not.toBeNull();
    const order = [...doc.head.children].map((node) => node.getAttribute("data-hub") ?? node.tagName.toLowerCase());
    expect(order.slice(0, 5)).toEqual(["meta", "base", "kit", "bridge", "kit"]);
    expect(doc.head.children[1]?.tagName).toBe("STYLE");
    expect(doc.head.children[2]?.tagName).toBe("STYLE");
    expect(doc.head.children[4]?.tagName).toBe("SCRIPT");
  });

  // The Hub's frame supplies the gutter; the browser's 8px body margin must not stack on it,
  // and a page that opted out of the kit still starts flush. The layer lets the page's own rules win.
  it("zeroes the body margin in a layer, with or without the kit", () => {
    for (const html of ["<p>kit</p>", '<meta name="hub-kit" content="off"><p>no kit</p>']) {
      const base = parse(buildExplainerDocument(html, options())).querySelector('style[data-hub="base"]');
      expect(base?.textContent).toBe(EXPLAINER_BASE_CSS);
    }
    expect(EXPLAINER_BASE_CSS).toMatch(/^@layer hub-base \{ html, body \{ margin: 0; \} \}$/);
  });

  it("allows no network and keeps the sandbox-relevant directives", () => {
    expect(EXPLAINER_CSP).toContain("default-src 'none'");
    expect(EXPLAINER_CSP).not.toMatch(/https?:|connect-src|frame-src/);
  });

  it("starts a standards-mode document and tells the page its theme", () => {
    const html = buildExplainerDocument("<p>bare fragment</p>", options({ theme: "dark" }));
    expect(html.startsWith("<!doctype html>")).toBe(true);
    expect(parse(html).documentElement.getAttribute("data-theme")).toBe("dark");
  });

  it("embeds the host kind and recurrence as inert JSON before the bridge runs", () => {
    const html = buildExplainerDocument("<p>status</p>", options({ context: { kind: "status", recurrence: { count: 3, dates: ["2026-11-15"] } } }));
    const doc = parse(html);
    const context = doc.getElementById("hub-context");
    expect(context?.getAttribute("type")).toBe("application/json");
    expect(JSON.parse(context?.textContent ?? "{}")).toEqual({ kind: "status", recurrence: { count: 3, dates: ["2026-11-15"] } });
    expect(doc.head.querySelector("#hub-context")?.nextElementSibling?.getAttribute("data-hub")).toBe("bridge");
  });

  // A page whose literal colours only work in light is shown in light even when
  // the Hub is dark, and the lock tells the bridge to ignore theme changes.
  it("keeps a page written for one scheme in that scheme and locks it there", () => {
    const lightOnly = "<style>:root{color:#192723;background:#f5f7f4}.card{background:white}</style><div class=card>Option A</div>";
    const locked = parse(buildExplainerDocument(lightOnly, options({ kitCss, theme: "dark" }))).documentElement;
    expect(locked.getAttribute("data-theme")).toBe("light");
    expect(locked.getAttribute("data-hub-scheme")).toBe("light");

    const themed = parse(buildExplainerDocument('<div style="background:var(--hub-surface);color:var(--hub-fg)">x</div>', options({ kitCss, theme: "dark" }))).documentElement;
    expect(themed.getAttribute("data-theme")).toBe("dark");
    expect(themed.hasAttribute("data-hub-scheme")).toBe(false);
  });

  it("leaves the kit out, but not the policy or the bridge, when the page opts out", () => {
    const doc = parse(buildExplainerDocument('<meta name="hub-kit" content="off"><p>mine</p>', options()));
    expect(doc.querySelector('style[data-hub="kit"]')).toBeNull();
    expect(doc.querySelector('script[data-hub="kit"]')).toBeNull();
    expect(doc.querySelector('meta[http-equiv="Content-Security-Policy"]')).not.toBeNull();
    expect(doc.querySelector('script[data-hub="bridge"]')).not.toBeNull();
  });

  it("injects the published kit stylesheet, whose rules all sit in the hub-kit layer", () => {
    expect(kitCss).toContain("@layer hub-kit");
    const doc = parse(buildExplainerDocument("<p>x</p>", options({ kitCss })));
    expect(doc.querySelector('style[data-hub="kit"]')?.textContent).toBe(kitCss);
  });

  it("cannot be broken out of by a script or style body that mentions its closing tag", () => {
    const html = buildExplainerDocument("<p>x</p>", options({ bridgeJs: 'var s="</script><img src=x>";', kitCss: "/* </style><b> */" }));
    const doc = parse(html);
    expect(doc.querySelectorAll("img")).toHaveLength(0);
    expect(doc.querySelector('script[data-hub="bridge"]')?.textContent).toContain("<\\/script");
  });

  it("replaces attachment names with data URLs and leaves other values alone", () => {
    const html = '<img src="set-a.png"><a href="https://example.test/x">x</a><img src="renders/walk/01.png"><img src="unknown.png">';
    const dataUrls = new Map([["set-a.png", "data:image/png;base64,AAA"], ["renders/walk/01.png", "data:image/png;base64,BBB"]]);
    const doc = parse(buildExplainerDocument(html, options({ dataUrls })));
    expect(doc.querySelectorAll("img")[0]?.getAttribute("src")).toBe("data:image/png;base64,AAA");
    expect(doc.querySelector("a")?.getAttribute("href")).toBe("https://example.test/x");
    expect(doc.querySelectorAll("img")[1]?.getAttribute("src")).toBe("data:image/png;base64,BBB");
    expect(doc.querySelectorAll("img")[2]?.getAttribute("src")).toBe("unknown.png");
  });

  it("keeps a placeholder that opens the viewer for a file beyond the inline limits", () => {
    const refs = findReferences('<video src="huge.mp4" poster="set-a.png"></video><a href="huge.mp4">get</a>', attachments);
    const { inline, deferred } = planInlining(refs);
    expect(inline.map((ref) => ref.name)).toEqual(["set-a.png"]);
    const doc = parse(
      buildExplainerDocument('<video src="huge.mp4" poster="set-a.png"></video><a href="huge.mp4">get</a>', options({ dataUrls: new Map([["set-a.png", "data:image/png;base64,AAA"]]), deferred })),
    );
    const video = doc.querySelector("video");
    expect(video?.getAttribute("src")).toBe(OPEN_PLACEHOLDER);
    expect(video?.getAttribute("data-hub-open")).toBe("huge.mp4");
    expect(video?.getAttribute("poster")).toBe("data:image/png;base64,AAA");
    expect(doc.querySelector("a")?.getAttribute("href")).toBe("#");
    expect(doc.querySelector("a")?.getAttribute("data-hub-open")).toBe("huge.mp4");
  });

  it("names a file inside a folder as name/path when it opens the viewer", () => {
    const refs = findReferences('<img src="renders/walk/01.png">', attachments);
    const doc = parse(buildExplainerDocument('<img src="renders/walk/01.png">', options({ deferred: refs })));
    expect(doc.querySelector("img")?.getAttribute("data-hub-open")).toBe("renders/walk/01.png");
  });

  it("removes what would redirect or re-root the page", () => {
    const doc = parse(buildExplainerDocument('<base href="https://evil.test/"><meta http-equiv="refresh" content="0;url=https://evil.test"><p>x</p>', options()));
    expect(doc.querySelector("base")).toBeNull();
    expect(doc.querySelector('meta[http-equiv="refresh"]')).toBeNull();
    expect(doc.querySelector('meta[http-equiv="Content-Security-Policy"]')).not.toBeNull();
  });

  it("keeps a policy the page brings itself: policies can only add restrictions", () => {
    const doc = parse(buildExplainerDocument("<meta http-equiv=\"Content-Security-Policy\" content=\"img-src 'none'\"><p>x</p>", options()));
    expect(doc.querySelectorAll('meta[http-equiv="Content-Security-Policy"]')).toHaveLength(2);
  });
});

describe("findReferences and planInlining", () => {
  it("lists each distinct attribute value once, in document order", () => {
    const refs = findReferences('<img src="./set-a.png"><img src="set-a.png"><link href="renders"><img src="renders/walk/01.png"><img src="nope.png">', attachments);
    expect(refs.map((ref) => [ref.value, ref.name, ref.path])).toEqual([
      ["./set-a.png", "set-a.png", ""],
      ["set-a.png", "set-a.png", ""],
      ["renders/walk/01.png", "renders", "walk/01.png"],
    ]);
  });

  it("does not treat a folder name alone, or a path that is not an entry, as a file", () => {
    expect(findReferences('<a href="renders">x</a><a href="renders/missing.png">y</a>', attachments)).toEqual([]);
  });

  it("stops inlining when the page budget is spent", () => {
    const refs = ["a", "b", "c"].map((name) => ({ value: name, name, path: "", sizeBytes: 10, mediaType: "image/png" }));
    const { inline, deferred } = planInlining(refs, { file: 10, page: 25 });
    expect(inline.map((ref) => ref.name)).toEqual(["a", "b"]);
    expect(deferred.map((ref) => ref.name)).toEqual(["c"]);
  });
});
