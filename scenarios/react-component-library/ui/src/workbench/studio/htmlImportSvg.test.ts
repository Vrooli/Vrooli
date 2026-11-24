import { describe, expect, it, vi } from "vitest";

import { parseSvgCssRules, safeSvg, svgDataUrl } from "./htmlImportSvg";
import type { HtmlCssRule } from "./htmlImportStyles";

const svgNamespace = "http://www.w3.org/2000/svg";

function element(tag: string, attrs: Record<string, string> = {}): Element {
  const node = document.createElementNS(svgNamespace, tag);
  for (const [name, value] of Object.entries(attrs)) node.setAttribute(name, value);
  return node;
}

describe("SVG HTML import sanitizer", () => {
  it("keeps safe declarations and selectors while skipping at-rules and unsafe CSS", () => {
    const rules = parseSvgCssRules([
      "@media screen { path { fill: red; } } " +
        "path, .shape { fill: #abc; stroke: url('#paint'); width: url(#paint); " +
        "filter: url(https://remote.invalid/filter); fill: javascript:run(); } " +
        "empty { unknown: value; }",
    ]);

    expect(rules.map((rule) => rule.selector)).toEqual(["path", ".shape"]);
    expect([...rules[0]!.declarations]).toEqual([
      ["fill", "#abc"],
      ["stroke", "url('#paint')"],
    ]);
    expect(parseSvgCssRules(["path, { fill: red; } path { unknown: value; }"])).toEqual([
      { selector: "path", declarations: new Map([["fill", "red"]]) },
    ]);
  });

  it("returns null for non-SVG sources and serialized output over the byte limit", () => {
    const html = document.createElement("div");
    const svg = element("svg", { viewBox: "0 0 10 10" });

    expect(safeSvg(html, document, [], [], 10_000)).toBeNull();
    expect(safeSvg(svg, document, [], [], 1)).toBeNull();
  });

  it("resolves inherited HTML variables and colors, including recursive fallback", () => {
    const parent = document.createElement("div");
    parent.className = "painted";
    const svg = element("svg", { fill: "inherit" });
    const path = element("path", { d: "M0 0h1v1z", fill: "var(--loop)" });
    path.setAttribute("style", "stroke: currentColor; color: var(--missing, #123456)");
    svg.append(path);
    parent.append(svg);

    const htmlRules: HtmlCssRule[] = [
      {
        selector: ".painted",
        declarations: new Map([
          ["--loop", "var(--loop, #abcdef)"],
          ["color", "#fedcba"],
        ]),
        order: 0,
      },
    ];
    const result = safeSvg(svg, document, [], htmlRules, 10_000);

    expect(result?.svg).toContain('fill="#fedcba"');
    expect(result?.svg).toContain('stroke="#123456"');
    expect(result?.svg).toContain('fill="#abcdef"');
    expect(result?.rejected).toBe(false);
  });

  it("rejects unsafe presentation values and ignores invalid CSS selectors", () => {
    const svg = element("svg");
    const path = element("path", {
      id: "shape",
      mask: "url(https://remote.invalid/mask)",
      fill: "var(--missing)",
    });
    path.setAttribute("style", "stroke: #abc");
    svg.append(path);
    const result = safeSvg(
      svg,
      document,
      [
        { selector: "[", declarations: new Map([["fill", "red"]]) },
        { selector: "path", declarations: new Map([["stroke", "#abc"]]) },
      ],
      [],
      10_000,
    );

    expect(result?.rejected).toBe(true);
    expect(result?.svg).not.toContain("remote.invalid");
    expect(result?.svg).not.toContain("var(--missing)");
    expect(result?.svg).toContain('stroke="#abc"');
  });

  it("expands xlink symbol references and preserves the symbol viewBox", () => {
    const svg = element("svg");
    const symbol = element("symbol", { id: "mark", viewBox: "" });
    symbol.append(element("circle", { cx: "4", cy: "4", r: "2" }));
    const use = element("use");
    use.setAttributeNS("http://www.w3.org/1999/xlink", "xlink:href", "#mark");
    svg.append(symbol, element("use"), use);
    document.body.append(svg);

    const result = safeSvg(svg, document, [], [], 10_000);
    svg.remove();

    expect(result?.svg).toContain('viewBox=""');
    expect(result?.svg).toContain("<circle");
    expect(result?.rejected).toBe(true);
  });

  it("rejects cyclic CSS variables and invalid local mask references", () => {
    const svg = element("svg");
    const path = element("path", { mask: "url(#bad id)" });
    svg.append(path);
    const result = safeSvg(
      svg,
      document,
      parseSvgCssRules(["path { fill: var(--loop); }"]),
      [
        {
          selector: "svg",
          declarations: new Map([["--loop", "var(--loop)"]]),
          order: 0,
        },
      ],
      10_000,
    );

    expect(result?.rejected).toBe(true);
    expect(result?.svg).not.toContain("mask=");
    expect(result?.svg).not.toContain("fill=");
  });

  it("uses the Blob byte-size fallback when TextEncoder is unavailable", () => {
    const svg = element("svg");
    const originalTextEncoder = globalThis.TextEncoder;
    vi.stubGlobal("TextEncoder", undefined);

    try {
      expect(safeSvg(svg, document, [], [], 10_000)?.svg).toContain("<svg");
    } finally {
      vi.stubGlobal("TextEncoder", originalTextEncoder);
    }
  });

  it("encodes non-ASCII SVG bytes in its data URL", () => {
    const value = svgDataUrl("<svg><title>café</title></svg>");

    const binary = atob(value.slice(value.indexOf(",") + 1));
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
    expect(new TextDecoder().decode(bytes)).toContain("café");
  });
});
