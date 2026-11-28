import { describe, expect, it } from "vitest";

import { parseHtml } from "./htmlImport";
import { walk, type TemplateNode } from "../templates/spec";

function collect(spec: TemplateNode, type: string): TemplateNode[] {
  const found: TemplateNode[] = [];
  walk(spec, (layer) => {
    if (layer.type === type) found.push(layer);
  });
  return found;
}

describe("HTML import boundary behavior", () => {
  it("returns a safe full-page source preview when no selectable region matches", () => {
    const parsed = parseHtml(
      '<!doctype html><title>Whole page</title><main><p onclick="bad()">Readable</p></main>',
    );
    const preview = parsed.sourcePreview("missing-region");

    expect(preview).toContain("Readable");
    expect(preview).toContain("Imported source");
    expect(preview).not.toContain("onclick");
    expect(preview).not.toContain("bad()");
  });

  it("falls back to the selected tag and sanitizes supplemental preview CSS", () => {
    const parsed = parseHtml('<section><p class="preview">Selected copy</p></section>');
    const region = parsed.regions.find((candidate) => candidate.tag === "section");
    if (!region) throw new Error("Expected the unlabelled section to be discoverable.");

    const preview = parsed.sourcePreview(region.id, [
      ".preview { background: url(data:image/png;base64,AAAA); mask: url(https://remote.invalid/mask.svg); } </style><script>unsafe()</script>",
    ]);

    expect(preview).toContain("Selected copy");
    expect(preview).toContain("data:image/png;base64,AAAA");
    expect(preview).toContain("mask: none");
    expect(preview).toContain("\\3C /style");
    expect(preview).not.toContain("remote.invalid");
    expect(preview).not.toContain("</style><script>unsafe()");
  });

  it("previews the exact selected nested region when source tags repeat", () => {
    const parsed = parseHtml(
      '<section><p>Outer region</p><div><section><p>Nested target</p></section></div><section><p>Sibling target</p></section></section>',
    );
    const nested = parsed.regions.find(
      (candidate) => candidate.tag === "section" && candidate.label === "Nested target",
    );
    if (!nested) throw new Error("Expected the nested section to be discoverable.");

    const preview = parsed.sourcePreview(nested.id);

    expect(preview).toContain("Nested target");
    expect(preview).not.toContain("Outer region");
    expect(preview).not.toContain("Sibling target");
  });

  it("keeps safe embedded image data while neutralizing remote CSS and unsafe style attributes", () => {
    const parsed = parseHtml(
      '<main style="background-image:url(https://remote.invalid/a.png)"><p style="color:red; background:url(data:image/png;base64,AAAA)">Copy</p></main>',
    );
    const preview = parsed.sourcePreview(null);

    expect(preview).toContain("background:url(data:image/png;base64,AAAA)");
    expect(preview).toContain("background-image:none");
    expect(preview).not.toContain("remote.invalid");
  });

  it("supports attribute generated pseudo copy and ignores unsupported pseudo content", () => {
    const parsed = parseHtml('<p data-label="New" class="copy">Announcement</p>', undefined, [
      ".copy::before { content: attr(data-label); color: #123; } .copy::after { content: counter(step); }",
    ]);
    const spec = parsed.regionSpec(null);
    const text = collect(spec, "text");

    expect(text.map((layer) => layer.props.text)).toEqual(["Announcement", "New"]);
    expect(text[1]?.props.css).toContain('"color":"#123"');
  });

  it("uses visual-only pseudo layers for empty quoted content and skips none content", () => {
    const parsed = parseHtml("<div class='mark'></div>", undefined, [
      '.mark::before { content: ""; width: 8px; background: #abc; } .mark::after { content: none; }',
    ]);
    const spec = parsed.regionSpec(null);
    const surfaces = collect(spec, "surface");

    expect(surfaces).toHaveLength(1);
    expect(surfaces[0]?.props.label).toBe("::before");
    expect(surfaces[0]?.props.css).toContain('"background":"#abc"');
  });

  it("preserves explicit CSS cascade order and lets safe inline declarations win", () => {
    const parsed = parseHtml(
      '<p id="title" class="copy" style="color: #789; font-weight: 700">Copy</p>',
      undefined,
      ["p { color: red } .copy { color: green } #title { color: blue }"],
    );
    const text = collect(parsed.regionSpec(null), "text")[0];

    expect(JSON.parse(String(text?.props.css))).toMatchObject({
      color: "#789",
      "font-weight": "700",
    });
  });

  it("skips nested at-rules and rejects unsafe or malformed CSS while keeping valid rules", () => {
    const parsed = parseHtml('<p class="copy">Copy</p>', undefined, [
      "@media screen { .copy { color: red; } } .copy { color: blue; background: url(https://remote.invalid/a.png); nonsense: value; } p:unknown-pseudo { color: green; }",
    ]);
    const text = collect(parsed.regionSpec(null), "text")[0];
    const css = JSON.parse(String(text?.props.css)) as Record<string, string>;

    expect(css).toMatchObject({ color: "blue" });
    expect(css).not.toHaveProperty("background");
    expect(css).not.toHaveProperty("nonsense");
  });

  it("applies selector specificity before stylesheet source order", () => {
    const parsed = parseHtml('<p class="copy">Copy</p>', undefined, [
      ".copy { color: red; } p { color: blue; }",
    ]);
    const text = collect(parsed.regionSpec(null), "text")[0];

    expect(JSON.parse(String(text?.props.css))).toMatchObject({ color: "red" });
  });

  it("keeps local SVG paint references while excluding other URL-based CSS values", () => {
    const parsed = parseHtml(
      '<svg viewBox="0 0 10 10"><defs><linearGradient id="paint"><stop offset="0"/></linearGradient></defs><path class="shape" d="M0 0h5v5z"/></svg>',
      undefined,
      [
        '.shape { fill: var(--missing, #abc); stroke: url("#paint"); width: url(#paint); filter: url(https://remote.invalid/filter.svg#blur); }',
      ],
    );
    const image = collect(parsed.regionSpec(null), "image")[0];
    const previewSvg = atob(String(image?.props.src).split(",")[1] || "");

    expect(previewSvg).toContain('fill="#abc"');
    expect(previewSvg).toContain('stroke="url(&quot;#paint&quot;)"');
    expect(previewSvg).not.toContain('width="url(#paint)"');
    expect(previewSvg).not.toContain("remote.invalid");
  });

  it("keeps safe SVG artwork while stripping unsafe attributes and unsupported subtrees", () => {
    const parsed = parseHtml(
      '<svg viewBox="0 0 10 10"><path id="safe" d="M0 0h5v5z" onload="run()" fill="url(https://remote.invalid/paint.svg#x)"/><foreignObject><p>Embedded HTML</p></foreignObject><circle id="bad id" cx="5" cy="5" r="2"/></svg>',
    );
    const image = collect(parsed.regionSpec(null), "image")[0];
    const svg = atob(String(image?.props.src).split(",")[1] || "");

    expect(svg).toContain('id="safe"');
    expect(svg).toContain('<path id="safe" d="M0 0h5v5z"');
    expect(svg).not.toContain("onload");
    expect(svg).not.toContain("remote.invalid");
    expect(svg).not.toContain("foreignObject");
    expect(svg).not.toContain("Embedded HTML");
    expect(svg).not.toContain('id="bad id"');
    expect(parsed.warnings.some((warning) => warning.includes("sanitized"))).toBe(true);
  });

  it("expands local SVG symbols and rejects missing, non-symbol, and recursive references", () => {
    const parsed = parseHtml(
      '<svg class="sprite"><symbol id="safe" viewBox="0 0 8 8"><path d="M0 0h4v4z"/></symbol><g id="not-a-symbol"/><symbol id="cycle"><use href="#cycle"/></symbol></svg>' +
        '<svg><use href="#safe"/><use href="#missing"/><use href="#not-a-symbol"/></svg>' +
        '<svg><use href="#cycle"/></svg>',
    );
    const images = collect(parsed.regionSpec(null), "image");
    const expanded = atob(String(images[0]?.props.src).split(",")[1] || "");
    const recursive = atob(String(images[1]?.props.src).split(",")[1] || "");

    expect(images).toHaveLength(2);
    expect(expanded).toContain('viewBox="0 0 8 8"');
    expect(expanded).toContain("<path");
    expect(expanded).not.toContain("missing");
    expect(expanded).not.toContain("not-a-symbol");
    expect(recursive).toContain("<svg");
    expect(recursive).not.toContain("<use");
    expect(parsed.warnings.some((warning) => warning.includes("sanitized"))).toBe(true);
  });

  it("sanitizes preview source attributes and keeps only data images", () => {
    const parsed = parseHtml(
      '<main><a href="javascript:run()" onclick="steal()" srcdoc="<script>steal()</script>">Link</a><img src="https://remote.invalid/a.png" srcset="https://remote.invalid/b.png 2x"><img src="data:image/png;base64,AAAA"></main>',
    );
    const preview = parsed.sourcePreview(null);

    expect(preview).toContain('src="data:image/png;base64,AAAA"');
    expect(preview).not.toContain("javascript:");
    expect(preview).not.toContain("onclick");
    expect(preview).not.toContain("srcdoc");
    expect(preview).not.toContain("remote.invalid");
    expect(preview).not.toContain("steal()");
  });
});
