import { describe, expect, it } from "vitest";

import { DEFAULT_TEMPLATE } from "../templates/library";
import { walk, type Template, type TemplateNode, type TemplateNodeType } from "../templates/spec";
import {
  HTML_LIMITS,
  HtmlImportError,
  htmlTemplate,
  parseHtml,
  type HtmlImportReason,
  type ParsedHtml,
} from "./htmlImport";

function nodesOfType(spec: TemplateNode, type: TemplateNodeType): TemplateNode[] {
  const found: TemplateNode[] = [];
  walk(spec, (current) => {
    if (current.type === type) found.push(current);
  });
  return found;
}

function firstOfType(spec: TemplateNode, type: TemplateNodeType): TemplateNode {
  const found = nodesOfType(spec, type)[0];
  if (!found) throw new Error(`Expected a ${type} layer`);
  return found;
}

function decodeSvgDataUrl(dataUrl: string): string {
  const binary = atob(dataUrl.split(",")[1] || "");
  const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}

function textOf(spec: TemplateNode): string | undefined {
  const { text } = spec.props;
  return typeof text === "string" ? text : undefined;
}

function texts(spec: TemplateNode): string[] {
  return nodesOfType(spec, "text")
    .map((layer) => textOf(layer))
    .filter((text): text is string => typeof text === "string");
}

function reasonOf(run: () => unknown): HtmlImportReason | undefined {
  try {
    run();
  } catch (error) {
    return error instanceof HtmlImportError ? error.reason : undefined;
  }
  return undefined;
}

function regionFor(parsed: ParsedHtml, tag: string): string {
  const region = parsed.regions.find((candidate) => candidate.tag === tag);
  if (!region) throw new Error(`Expected a ${tag} region`);
  return region.id;
}

function maxDepth(spec: TemplateNode): number {
  let deepest = 0;
  const visit = (current: TemplateNode, depth: number) => {
    deepest = Math.max(deepest, depth);
    for (const child of current.children ?? []) visit(child, depth + 1);
  };
  visit(spec, 0);
  return deepest;
}

describe("parseHtml element mapping", () => {
  it("wraps the whole body in a page without adding imported-page padding", () => {
    const spec = parseHtml("<p>Hello</p>").regionSpec(null);
    expect(spec.type).toBe("page");
    expect(spec.props.pad).toBe(0);
    expect(spec.children).toHaveLength(1);
  });

  it("keeps source block flow when RCL layers have grid defaults", () => {
    const spec = parseHtml(
      '<main><div><p>Block</p></div><section style="display:flex"><p>Row</p></section></main>',
    ).regionSpec(null);
    const main = spec.children?.[0];
    const group = main?.children?.[0];
    const explicitFlex = main?.children?.[1];

    expect(JSON.parse(String(main?.props.css)).display).toBe("block");
    expect(JSON.parse(String(group?.props.css)).display).toBe("block");
    expect(JSON.parse(String(explicitFlex?.props.css)).display).toBe("flex");
  });

  it("maps h1/h2 to display text and h3-h6 to title text", () => {
    const spec = parseHtml(
      "<h1>One</h1><h2>Two</h2><h3>Three</h3><h4>Four</h4><h5>Five</h5><h6>Six</h6>",
    ).regionSpec(null);
    const layers = nodesOfType(spec, "text");
    expect(layers.map((layer) => textOf(layer))).toEqual([
      "One",
      "Two",
      "Three",
      "Four",
      "Five",
      "Six",
    ]);
    expect(layers.map((layer) => layer.props.variant)).toEqual([
      "display",
      "display",
      "title",
      "title",
      "title",
      "title",
    ]);
  });

  it("maps body copy tags and gives small text a muted tone", () => {
    const spec = parseHtml(
      "<p>Paragraph</p><span>Span</span><label>Label</label><small>Fine print</small>Plain",
    ).regionSpec(null);
    const layers = nodesOfType(spec, "text");
    expect(layers.map((layer) => textOf(layer))).toEqual([
      "Paragraph",
      "Span",
      "Label",
      "Fine print",
      "Plain",
    ]);
    expect(layers.map((layer) => layer.props.variant)).toEqual([
      "body",
      "inline",
      "body",
      "body",
      "body",
    ]);
    expect(layers[3]?.props.tone).toBe("muted");
    expect(layers[0]?.props.tone).toBeUndefined();
  });

  it("maps anchors to accent text and buttons to secondary controls", () => {
    const spec = parseHtml(
      '<a href="https://example.test">Read more</a><button type="submit">Send</button>',
    ).regionSpec(null);
    const link = nodesOfType(spec, "text")[0];
    expect(link?.props).toMatchObject({ text: "Read more", tone: "accent" });
    const button = firstOfType(spec, "button");
    expect(button.props).toMatchObject({ text: "Send", variant: "secondary" });
  });

  it("preserves inherited source text color through RCL text variants", () => {
    const spec = parseHtml(
      '<section style="color:#171c1d"><p>Source-colored copy</p>Plain copy</section>',
    ).regionSpec(null);
    const text = nodesOfType(spec, "text");

    expect(text.map((layer) => JSON.parse(String(layer.props.css)).color)).toEqual([
      "inherit",
      "inherit",
    ]);
  });

  it("maps text-labeled source buttons to RCL Button while preserving their child layout", () => {
    const parsed = parseHtml(
      '<div class="row"><button class="category"><span>Groceries</span><span>€96 of €140</span></button></div>',
      undefined,
      [
        ".row { display:flex; gap:12px; }\n" +
          ".category { display:flex; flex-direction:column; gap:4px; }",
      ],
    );
    const spec = parsed.regionSpec(null);
    const row = nodesOfType(spec, "row").find((layer) => layer.children?.length === 1);
    const category = row?.children?.[0];

    expect(row?.props.gap).toBe(0);
    expect(JSON.parse(String(row?.props.css)).gap).toBe("12px");
    expect(category?.type).toBe("stack");
    expect(JSON.parse(String(category?.props.css))["flex-direction"]).toBe("column");
    expect(JSON.parse(String(category?.props.css)).display).toBe("flex");
    expect(nodesOfType(category!, "text").map(textOf)).toEqual(["Groceries", "€96 of €140"]);
    expect(nodesOfType(spec, "button")).toHaveLength(0);
  });

  it("maps visible-label navigation buttons to RCL Button and leaves icon-only buttons custom", () => {
    const parsed = parseHtml(
      '<nav><button class="tab"><svg viewBox="0 0 10 10"><circle cx="5" cy="5" r="4"/></svg><span>Home</span></button><button aria-label="Notifications"><svg viewBox="0 0 10 10"><circle cx="5" cy="5" r="4"/></svg></button></nav>',
      undefined,
      [".tab { display:flex; flex-direction:column; gap:3px; height:52px; flex:1 1 0; }"],
    );
    const spec = parsed.regionSpec(null);
    const button = firstOfType(spec, "button");

    expect(button.children).toHaveLength(2);
    expect(texts(button)).toContain("Home");
    expect(JSON.parse(String(button.props.css))).toMatchObject({
      display: "flex",
      "flex-direction": "column",
      height: "52px",
    });
    expect(nodesOfType(spec, "stack").some((layer) => texts(layer).length === 0)).toBe(true);
  });

  it("preserves the pressed state of a short labeled toggle as an RCL Button", () => {
    const spec = parseHtml(
      '<button class="pill" aria-pressed="false"><span>Watered</span></button>',
      undefined,
      [".pill { display:inline-flex; height:46px; padding:0 26px; border-radius:24px; }"],
    ).regionSpec(null);
    const button = firstOfType(spec, "button");

    expect(button.props).toMatchObject({ text: "", pressed: false, variant: "secondary" });
    expect(texts(button)).toEqual(["Watered"]);
  });

  it.each(["section", "article", "header", "footer", "main", "nav", "div"])(
    "maps %s to a stack with gap 14",
    (tag) => {
      const spec = parseHtml(`<${tag}><p>Inside</p></${tag}>`).regionSpec(null);
      const stack = firstOfType(spec, "stack");
      expect(stack.props.gap).toBe(14);
      expect(texts(stack)).toEqual(["Inside"]);
    },
  );

  it("names containers after their source role and own heading", () => {
    const spec = parseHtml(
      "<section><h2>Why now</h2><p>Copy</p></section><div><p>Group</p></div>",
    ).regionSpec(null);
    const stacks: TemplateNode[] = [];
    walk(spec, (layer) => {
      if (layer.type === "stack") stacks.push(layer);
    });
    expect(stacks[0]?.props.label).toBe("Section · Why now");
    expect(stacks[1]?.props.label).toBe("Group");
  });

  it("maps lists, list items and tables", () => {
    const spec = parseHtml(
      "<ul><li>Plain item</li><li><p>Rich item</p></li></ul>" +
        "<ol><li>One</li></ol>" +
        "<table><tbody><tr><th>Head</th><td>Cell</td></tr></tbody></table>",
    ).regionSpec(null);
    const stacks = nodesOfType(spec, "stack");
    expect(stacks.length).toBeGreaterThanOrEqual(4);
    expect(stacks.every((stack) => stack.props.gap === 14)).toBe(true);

    const rows = nodesOfType(spec, "row");
    // Rich list item and the table row; the plain item stays copy.
    expect(rows).toHaveLength(2);
    expect(rows.every((row) => row.props.gap === 12)).toBe(true);
    expect(texts(spec)).toEqual(["Plain item", "Rich item", "One", "Head", "Cell"]);
  });

  it("maps hr to a divider and br to a 12px spacer", () => {
    const spec = parseHtml("<p>Before</p><hr><br><p>After</p>").regionSpec(null);
    expect(firstOfType(spec, "divider").props).toEqual({});
    expect(firstOfType(spec, "spacer").props).toEqual({ h: 12 });
  });

  it("maps unknown elements by content: text when copy-only, stack when structural", () => {
    const spec = parseHtml(
      "<widget>Bare copy</widget><gadget><p>Nested copy</p></gadget>",
    ).regionSpec(null);
    expect(texts(spec)).toEqual(["Bare copy", "Nested copy"]);
    const stack = firstOfType(spec, "stack");
    expect(stack.props.gap).toBe(14);
    expect(texts(stack)).toEqual(["Nested copy"]);
  });

  it("keeps form content while dropping the form wrapper", () => {
    const spec = parseHtml("<form><p>Name</p><button>Continue</button></form>").regionSpec(null);
    expect(spec.children?.map((child) => child.type)).toEqual(["text", "button"]);
  });
});

describe("parseHtml sanitisation", () => {
  it("removes scripts, styles, frames and embedded content", () => {
    const spec = parseHtml(
      "<script>alert('nope')</script>" +
        "<style>p { color: red }</style>" +
        '<iframe src="https://example.test">frame fallback</iframe>' +
        "<noscript>noscript copy</noscript>" +
        "<template><p>template copy</p></template>" +
        '<object data="movie">object copy</object>' +
        '<embed src="movie">' +
        "<audio>audio copy</audio>" +
        "<canvas>canvas copy</canvas>" +
        '<meta name="x" content="y">' +
        '<link rel="stylesheet" href="https://example.test/a.css">' +
        "<p>Kept</p>",
    ).regionSpec(null);
    expect(texts(spec)).toEqual(["Kept"]);
  });

  it("drops source attributes while retaining safe inline styling on editable layers", () => {
    const spec = parseHtml(
      '<p id="hero" class="lead" style="color:red" onclick="steal()" href="javascript:void(0)" data-x="1">Copy</p>',
    ).regionSpec(null);
    const allowed = new Set(["text", "variant", "tone", "gap", "pad", "h", "label", "css"]);
    const seen = new Set<string>();
    walk(spec, (layer) => {
      for (const key of Object.keys(layer.props)) seen.add(key);
    });
    expect([...seen].every((key) => allowed.has(key))).toBe(true);
    expect(firstOfType(spec, "text").props).toMatchObject({
      text: "Copy",
      variant: "body",
      css: '{"color":"red"}',
    });
  });

  it("drops images and media sources with a warning instead of fetching them", () => {
    const parsed = parseHtml(
      '<p>Before</p><img src="https://example.test/a.png" alt="A">' +
        '<picture><source srcset="b.png"><img src="b.png" alt="B"></picture>' +
        "<svg><text>Vector copy</text></svg>" +
        '<video src="c.mp4">video copy</video>' +
        "<p>After</p>",
    );
    const spec = parsed.regionSpec(null);
    expect(texts(spec)).toEqual(["Before", "After"]);
    expect(parsed.warnings.some((warning) => warning.includes("images were skipped"))).toBe(true);
    expect(
      parsed.warnings.some((warning) => warning.includes("remote assets are not fetched")),
    ).toBe(true);
  });

  it("imports safe inline SVG and expands local sprite symbols into bounded image layers", () => {
    const parsed = parseHtml(
      '<svg class="sprite"><symbol id="mark" viewBox="0 0 24 24"><path d="M2 2h20v20H2z" fill="currentColor"/></symbol></svg>' +
        '<section><svg aria-label="Mark"><use href="#mark"/></svg><svg viewBox="0 0 10 10"><circle cx="5" cy="5" r="4" fill="#123456"/></svg></section>',
    );
    const spec = parsed.regionSpec(null);
    const images = nodesOfType(spec, "image");
    expect(images).toHaveLength(2);
    const symbol = String(images[0]?.props.src);
    expect(symbol).toMatch(/^data:image\/svg\+xml;base64,/);
    const decoded = decodeSvgDataUrl(symbol);
    expect(decoded).toContain('viewBox="0 0 24 24"');
    expect(decoded).toContain("<path");
    expect(decoded).not.toContain("<use");
    expect(images[0]?.props).toMatchObject({ alt: "Mark", fit: "contain", h: 180 });
    expect(parsed.warnings).toEqual([]);
  });

  it("keeps safe artwork while rejecting unresolved, external, and recursive sprite references", () => {
    const parsed = parseHtml(
      '<svg class="sprite"><symbol id="loop" viewBox="0 0 12 12"><path d="M1 1h10v10H1z"/><use href="#loop"/></symbol></svg>' +
        '<svg xmlns:xlink="http://www.w3.org/1999/xlink"><path d="M0 0h2v2z"/><use href="#loop"/><use xlink:href="#loop"/><use href="#missing"/><use href="https://example.test/icon.svg#mark"/></svg>',
    );
    const images = nodesOfType(parsed.regionSpec(null), "image");
    expect(images).toHaveLength(1);

    const svg = decodeSvgDataUrl(String(images[0]?.props.src));
    expect(svg).toContain("<path");
    expect(svg).toContain('viewBox="0 0 12 12"');
    expect(svg).not.toContain("<use");
    expect(svg).not.toContain("example.test");
    expect(parsed.warnings.some((warning) => warning.includes("sanitized"))).toBe(true);
  });

  it("preserves canonical SVG tag and attribute casing for viewBoxes, gradients, and clips", () => {
    const parsed = parseHtml(
      '<svg viewBox="0 0 20 20"><defs><linearGradient id="paint" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#fff"/><stop offset="1" stop-color="#000"/></linearGradient><clipPath id="clip"><circle cx="10" cy="10" r="9"/></clipPath></defs><path d="M0 0h20v20H0z" fill="url(#paint)" clip-path="url(#clip)"/></svg>',
    );
    const data = String(firstOfType(parsed.regionSpec(null), "image").props.src);
    const svg = decodeSvgDataUrl(data);
    expect(svg).toContain('viewBox="0 0 20 20"');
    expect(svg).toContain("<linearGradient");
    expect(svg).toContain("gradientUnits=");
    expect(svg).toContain("<clipPath");
    expect(svg).toContain('clip-path="url(#clip)"');
    expect(svg).not.toContain("lineargradient");
    expect(svg).not.toContain("clippath");
    expect(parsed.warnings).toEqual([]);
  });

  it("carries safe local CSS presentation styles into editable SVG artwork", () => {
    const parsed = parseHtml(
      '<svg class="sb-signal" viewBox="0 0 18 12"><rect class="bar" width="3" height="4"/></svg>',
      "eight-screens.html",
      [".sb-signal { width: 17px; height: 11px; } .bar { fill: #123456; opacity: .7; }"],
    );
    const svg = decodeSvgDataUrl(String(firstOfType(parsed.regionSpec(null), "image").props.src));
    expect(svg).toContain('width="17px"');
    expect(svg).toContain('height="11px"');
    expect(svg).toContain('fill="#123456"');
    expect(svg).toContain('opacity=".7"');
    expect(svg).not.toContain("<style");
  });

  it("flattens inherited SVG colors and local CSS variables into self-contained artwork", () => {
    const parsed = parseHtml(
      '<article class="plant"><svg class="leaf" viewBox="0 0 10 10"><path fill="currentColor" d="M0 0h5v5z"/><path fill="var(--bg)" d="M5 5h5v5z"/></svg></article>',
      "plant.html",
      [".plant { --leaf: #14523f; --bg: #d7ef66; } .leaf { color: var(--leaf); }"],
    );
    const svg = decodeSvgDataUrl(String(firstOfType(parsed.regionSpec(null), "image").props.src));

    expect(svg).toContain('fill="#14523f"');
    expect(svg).toContain('fill="#d7ef66"');
    expect(svg).not.toContain("currentColor");
    expect(svg).not.toContain("var(--");
  });

  it("inherits the HTML ancestor color for currentColor SVG presentation", () => {
    const parsed = parseHtml(
      '<section class="status"><svg viewBox="0 0 10 10"><path fill="currentColor" d="M0 0h5v5z"/></svg></section>',
      "status.html",
      [".status { color: #eef0ec; }"],
    );
    const svg = decodeSvgDataUrl(String(firstOfType(parsed.regionSpec(null), "image").props.src));

    expect(svg).toContain('fill="#eef0ec"');
    expect(svg).not.toContain("currentColor");
  });

  it("flattens inherited root SVG paint for standalone image layers", () => {
    const parsed = parseHtml(
      '<section class="status"><svg viewBox="0 0 10 10"><path d="M0 0h5v5z"/></svg></section>',
      "status.html",
      [".status { color:#eef0ec; } .status svg { color:inherit; fill:inherit; stroke:inherit; }"],
    );
    const svg = decodeSvgDataUrl(String(firstOfType(parsed.regionSpec(null), "image").props.src));

    expect(svg).toContain('fill="#eef0ec"');
    expect(svg).not.toContain('fill="inherit"');
  });

  it("keeps safe source CSS on selectable HTML layers and the page root", () => {
    const parsed = parseHtml(
      '<main class="screen"><h1 class="title">Hello</h1><button class="action">Go</button></main>',
      "screen.html",
      [
        ":root { --ink: #eee; } body { background: #101010; } .screen { display: grid; gap: 16px; padding: 20px; background: var(--ink); background-image: url(https://evil.test/x); } .title { color: #abc; } h1 { color: #f00; } @media (max-width: 400px) { .title { color: #f00; } } .action { border-radius: 999px; }",
      ],
    );
    const spec = parsed.regionSpec(null);
    const cssOf = (node: TemplateNode) =>
      JSON.parse(String(node.props.css || "{}")) as Record<string, string>;
    const title = nodesOfType(spec, "text").find((layer) => textOf(layer) === "Hello");
    const button = firstOfType(spec, "button");
    const screen = nodesOfType(spec, "stack")[0];
    expect(cssOf(spec)).toMatchObject({ "--ink": "#eee", background: "#101010" });
    expect(cssOf(screen!)).toMatchObject({
      display: "grid",
      gap: "16px",
      padding: "20px",
      background: "var(--ink)",
    });
    expect(cssOf(title!)).toMatchObject({ color: "#abc" });
    expect(cssOf(button)).toMatchObject({ "border-radius": "999px" });
    expect(JSON.stringify(spec)).not.toContain("evil.test");
  });

  it("retains empty styled source elements as selectable artwork layers", () => {
    const parsed = parseHtml('<main><i class="ornament"></i></main>', undefined, [
      ".ornament { display: block; width: 24px; height: 24px; border-radius: 50%; background: #abc; }",
    ]);
    const artwork = firstOfType(parsed.regionSpec(null), "surface");
    expect(artwork.props.label).toBe("Artwork");
    expect(JSON.parse(String(artwork.props.css))).toMatchObject({
      width: "24px",
      height: "24px",
      background: "#abc",
    });
  });

  it("imports static CSS before/after pseudo-elements as editable child layers", () => {
    const parsed = parseHtml(
      '<div class="frame"><i class="peak" data-label="47.60"></i></div>',
      undefined,
      [
        '.frame { position: relative; } .frame::before { content: ""; position: absolute; left: 0; width: 3px; height: 20px; background: #abc; } .peak { display: block; position: relative; height: 80px; background: #456; } .peak::after { content: attr(data-label); position: absolute; right: -8px; top: -27px; padding: 3px 6px; background: #f3f4ec; color: #1a1e14; }',
      ],
    );
    const spec = parsed.regionSpec(null);
    let labelHost: TemplateNode | undefined;
    const visit = (layer: TemplateNode) => {
      if (layer.children?.some((child) => child.props.text === "47.60")) labelHost = layer;
      for (const child of layer.children ?? []) visit(child);
    };
    visit(spec);
    expect(nodesOfType(spec, "text").map(textOf)).toContain("47.60");
    expect(nodesOfType(spec, "surface").some((layer) => layer.props.label === "::before")).toBe(
      true,
    );
    expect(labelHost?.type).toBe("surface");
    expect(JSON.parse(String(labelHost?.props.css)).overflow).toBe("visible");
  });

  it("rejects external and executable CSS while preserving safe SVG declarations", () => {
    const parsed = parseHtml('<svg class="art"><path class="shape" d="M0 0"/></svg>', undefined, [
      ".shape { fill: #abc; stroke: url(https://evil.test/a); } .art { fill: url(javascript:alert(1)); }",
    ]);
    const svg = decodeSvgDataUrl(String(firstOfType(parsed.regionSpec(null), "image").props.src));
    expect(svg).toContain('fill="#abc"');
    expect(svg).not.toContain("evil.test");
    expect(svg).not.toContain("javascript:");
  });

  it("drops executable and external SVG content and reports unsupported artwork", () => {
    const parsed = parseHtml(
      '<svg viewBox="0 0 10 10"><script>alert(1)</script><image href="https://evil.test/a.png"/><path d="M0 0" onclick="steal()" fill="url(https://evil.test/x)"/></svg>',
    );
    const spec = parsed.regionSpec(null);
    const serialized = JSON.stringify(spec);
    expect(serialized).not.toContain("evil.test");
    expect(serialized).not.toContain("steal");
    expect(serialized).not.toContain("alert");
    // The unsupported <image> element is discarded while safe path geometry remains.
    const source = decodeSvgDataUrl(String(firstOfType(spec, "image").props.src));
    expect(source).toContain("<path");
    expect(source).not.toContain("image");
    expect(parsed.warnings.some((warning) => warning.includes("sanitized"))).toBe(true);
  });

  it("bounds aggregate SVG output when local artwork is repeated", () => {
    const paths = '<path d="M0 0h10v10z"/>'.repeat(1000);
    const repeated =
      `<svg class="sprite"><symbol id="heavy" viewBox="0 0 10 10">${paths}</symbol></svg>` +
      '<svg><use href="#heavy"/></svg>'.repeat(20);
    const parsed = parseHtml(repeated);
    const images = nodesOfType(parsed.regionSpec(null), "image");
    expect(images.length).toBeLessThan(20);
    expect(parsed.warnings.some((warning) => warning.includes("total SVG limit"))).toBe(true);
  });

  it("reports a single skipped image without pluralising wrongly", () => {
    const parsed = parseHtml('<p>Copy</p><img src="a.png">');
    expect(parsed.warnings.some((warning) => warning.includes("1 image was skipped"))).toBe(true);
  });

  it("collapses whitespace and skips empty text", () => {
    const parsed = parseHtml("<p>  Hello\n\t   world  </p>   \n  <p> </p>");
    const spec = parsed.regionSpec(null);
    expect(texts(spec)).toEqual(["Hello world"]);
    // The whitespace-only paragraph is skipped; only importable regions are listed.
    expect(parsed.regions.filter((region) => region.tag === "p")).toHaveLength(1);
  });
});

describe("parseHtml regions", () => {
  const page = "<main><section><h2>Region</h2><p>Copy</p></section><footer>End</footer></main>";

  it("lists importable regions in document order with depth and counts", () => {
    const parsed = parseHtml(page);
    expect(parsed.regions.map((region) => region.tag)).toEqual([
      "main",
      "section",
      "h2",
      "p",
      "footer",
    ]);
    expect(parsed.regions.map((region) => region.depth)).toEqual([0, 1, 2, 2, 1]);
    const section = parsed.regions.find((region) => region.tag === "section");
    expect(section).toMatchObject({ tag: "section", children: 2, depth: 1 });
    expect(section?.layers).toBe(3);
    expect(section?.label).toBe("Region Copy");
    expect(new Set(parsed.regions.map((region) => region.id)).size).toBe(parsed.regions.length);
  });

  it("imports only the chosen region subtree", () => {
    const parsed = parseHtml(page);
    const spec = parsed.regionSpec(regionFor(parsed, "section"));
    expect(spec.type).toBe("page");
    expect(spec.props.pad).toBe(0);
    expect(spec.children).toHaveLength(1);
    const section = spec.children?.[0];
    expect(section?.type).toBe("stack");
    expect(texts(spec)).toEqual(["Region", "Copy"]);
    expect(JSON.stringify(spec)).not.toContain("End");
  });

  it("carries inherited ancestor CSS context into an extracted region", () => {
    const parsed = parseHtml(
      '<main class="theme"><article data-app="profile"><h2>Profile</h2></article></main>',
      undefined,
      [".theme { --ink: #123456; color: #234567; font-family: Inter; background: #000; }"],
    );
    const region = parsed.regions.find((candidate) => candidate.sourceHint === "profile");
    expect(region).toBeDefined();
    const extracted = parsed.regionSpec(region!.id).children?.[0];
    const css = JSON.parse(String(extracted?.props.css ?? "{}")) as Record<string, string>;

    expect(css).toMatchObject({ "--ink": "#123456", color: "#234567", "font-family": "Inter" });
    expect(css).not.toHaveProperty("background");
  });

  it("keeps named screen regions discoverable while importing only the selected screen", () => {
    const precedingContent = Array.from(
      { length: 510 },
      (_, index) => `<p>Intro ${index}</p>`,
    ).join("");
    const screens = Array.from(
      { length: 8 },
      (_, index) =>
        `<article class="device" data-app="screen-${index + 1}"><h2>Screen ${index + 1}</h2><p>Copy ${index + 1}</p></article>`,
    ).join("");
    const parsed = parseHtml(`<main>${precedingContent}${screens}</main>`);
    const candidates = parsed.regions.filter((region) => region.tag === "article");

    expect(candidates).toHaveLength(8);
    expect(candidates.map((region) => region.sourceHint)).toEqual(
      Array.from({ length: 8 }, (_, index) => `screen ${index + 1}`),
    );

    const selected = parsed.regionSpec(candidates[4]!.id);
    expect(texts(selected)).toEqual(["Screen 5", "Copy 5"]);
    expect(JSON.stringify(selected)).not.toContain("Screen 4");
    expect(JSON.stringify(selected)).not.toContain("Screen 6");
    expect(JSON.stringify(selected)).not.toContain("screen-5");
  });

  it("builds a selected visual source view with local CSS and without executable or remote content", () => {
    const parsed = parseHtml(
      '<!doctype html><html><head><title>Study</title><style>.inline{color:blue}</style><link rel="stylesheet" href="https://evil.test/x.css"></head><body><svg class="sprite"><symbol id="i-home"><path d="M1 1"></path></symbol></svg><article class="device" data-app="run-red" onclick="steal()"><script>steal()</script><img src="https://evil.test/pixel"><svg><use href="#i-home"></use></svg><h1>Long run</h1></article><article data-app="other"><h1>Other screen</h1></article></body></html>',
    );
    const region = parsed.regions.find((candidate) => candidate.sourceHint === "run red");
    expect(region).toBeDefined();
    const source = parsed.sourcePreview(region?.id ?? null, [
      '@import url("https://evil.test/font.css"); .device { color: red; background: url(https://evil.test/image.png); }',
    ]);

    expect(source).toContain('class="device"');
    expect(source).toContain('id="i-home"');
    expect(source).toContain('href="#i-home"');
    expect(source).toContain("Long run");
    expect(source).toContain("color: red");
    expect(source).toContain("color:blue");
    expect(source).toContain("Content-Security-Policy");
    expect(source).not.toContain("Other screen");
    expect(source).not.toContain("steal()");
    expect(source).not.toContain("onclick");
    expect(source).not.toContain("evil.test");
    expect(source).not.toContain("<script");
  });

  it("reuses the same node ids whether a region or the whole document is imported", () => {
    const parsed = parseHtml(page);
    const whole = parsed.regionSpec(null);
    const section = parsed.regionSpec(regionFor(parsed, "section"));
    expect(section.children?.[0]?.id).toBe(whole.children?.[0]?.children?.[0]?.id);
  });

  it("throws a typed error for an unknown region id", () => {
    const parsed = parseHtml(page);
    expect(() => parsed.regionSpec("region-999")).toThrow(HtmlImportError);
  });

  it("builds deterministic ids that do not depend on content", () => {
    const first = parseHtml(page).regionSpec(null);
    const second = parseHtml(page).regionSpec(null);
    expect(JSON.stringify(first)).toBe(JSON.stringify(second));
    const ids: string[] = [];
    walk(first, (layer) => ids.push(layer.id));
    expect(ids.every((id) => /^html-\d+-[a-z]+$/.test(id) || id === "html-page")).toBe(true);
    const renamed = parseHtml(page.replace("Region", "Renamed")).regionSpec(null);
    const renamedIds: string[] = [];
    walk(renamed, (layer) => renamedIds.push(layer.id));
    expect(renamedIds).toEqual(ids);
  });

  it("returns fresh specs so a caller cannot mutate the cached parse", () => {
    const parsed = parseHtml(page);
    const first = parsed.regionSpec(null);
    first.children?.splice(0);
    expect(parsed.regionSpec(null).children).toHaveLength(1);
  });
});

describe("parseHtml caps and warnings", () => {
  it("caps the import at the configured layer budget", () => {
    const items = Array.from(
      { length: HTML_LIMITS.layers + 10 },
      (_, index) => `<p>Item ${index}</p>`,
    ).join("");
    const parsed = parseHtml(items);
    expect(parsed.layers).toBe(HTML_LIMITS.layers);
    expect(nodesOfType(parsed.regionSpec(null), "text")).toHaveLength(HTML_LIMITS.layers);
    expect(
      parsed.warnings.some((warning) => warning.includes(`${HTML_LIMITS.layers} layers`)),
    ).toBe(true);
  });

  it("caps the scan at the configured element budget", () => {
    const empty = "<div></div>".repeat(HTML_LIMITS.elements + 20);
    const parsed = parseHtml(empty);
    expect(parsed.regions).toHaveLength(0);
    expect(
      parsed.warnings.some((warning) =>
        warning.includes(`${HTML_LIMITS.elements} source elements`),
      ),
    ).toBe(true);
  });

  it("caps depth at 12 produced layers and keeps shallower content", () => {
    const deep = `${"<div>".repeat(20)}<p>Deep copy</p>${"</div>".repeat(20)}`;
    const parsed = parseHtml(`<p>Top copy</p>${deep}`);
    const spec = parsed.regionSpec(null);
    expect(texts(spec)).toEqual(["Top copy"]);
    expect(maxDepth(spec)).toBeLessThanOrEqual(HTML_LIMITS.depth);
    expect(parsed.warnings.some((warning) => warning.includes("12 levels"))).toBe(true);
  });

  it("reports no importable content when only media was present", () => {
    const parsed = parseHtml('<img src="https://example.test/a.png">');
    expect(parsed.layers).toBe(0);
    expect(parsed.regions).toHaveLength(0);
    expect(parsed.regionSpec(null).children).toEqual([]);
  });
});

describe("parseHtml refusals and title", () => {
  it("refuses blank input with an empty reason", () => {
    expect(reasonOf(() => parseHtml(""))).toBe("empty");
    expect(reasonOf(() => parseHtml("   \n\t "))).toBe("empty");
  });

  it("refuses input over 2 MiB with a size reason", () => {
    const oversized = `<p>${"a".repeat(HTML_LIMITS.bytes)}</p>`;
    expect(reasonOf(() => parseHtml(oversized))).toBe("size");
  });

  it("refuses aggregate HTML and local CSS over 2 MiB for direct callers", () => {
    const oversizedCss = `.art { fill: #123; }${" ".repeat(HTML_LIMITS.bytes)}`;
    expect(
      reasonOf(() =>
        parseHtml("<svg class='art'><path d='M0 0'/></svg>", undefined, [oversizedCss]),
      ),
    ).toBe("size");
    expect(
      reasonOf(() =>
        parseHtml(`<p>${"x".repeat(HTML_LIMITS.bytes - 5)}</p>`, undefined, [".art{fill:red}"]),
      ),
    ).toBe("size");
  });

  it("keeps the source title, then the file name, then a stable fallback", () => {
    expect(parseHtml("<title> Source title </title><p>x</p>", "fallback.html").title).toBe(
      "Source title",
    );
    expect(parseHtml("<p>x</p>", "fallback.html").title).toBe("fallback");
    expect(parseHtml("<p>x</p>").title).toBe("Imported HTML");
  });
});

describe("htmlTemplate", () => {
  const parsed = parseHtml("<main><h1>Page</h1><p>Copy</p></main>", "page.html");

  it("uses the default template tokens while staying local and honest", () => {
    const template: Template = htmlTemplate(parsed, null, "Imported page");
    expect(template.name).toBe("Imported page");
    expect(template.tagline).toBe("Imported HTML");
    expect(template.local).toBe(true);
    expect(template.category).toBe(DEFAULT_TEMPLATE.category);
    expect(template.display).toBe(DEFAULT_TEMPLATE.display);
    expect(template.radius).toBe(DEFAULT_TEMPLATE.radius);
    expect(template.slotHint).toBe(DEFAULT_TEMPLATE.slotHint);
    expect(template.palette).toEqual(DEFAULT_TEMPLATE.palette);
    expect(template.palette).not.toBe(DEFAULT_TEMPLATE.palette);
    expect(template.spec.type).toBe("page");
  });

  it("imports the chosen region and falls back to the parsed title for its name", () => {
    const sectionless = parseHtml("<title>Hello</title><main><h1>Hello</h1></main>", "page.html");
    const template = htmlTemplate(sectionless, regionFor(sectionless, "h1"), "   ");
    expect(template.name).toBe("Hello");
    expect(template.spec.children?.[0]?.props.text).toBe("Hello");
  });

  it("gives every import its own template id", () => {
    expect(htmlTemplate(parsed, null, "One").id).not.toBe(htmlTemplate(parsed, null, "Two").id);
  });
});
