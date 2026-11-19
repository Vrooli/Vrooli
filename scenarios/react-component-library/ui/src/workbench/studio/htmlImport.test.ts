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
  it("wraps the whole body in a page with pad 40", () => {
    const spec = parseHtml("<p>Hello</p>").regionSpec(null);
    expect(spec.type).toBe("page");
    expect(spec.props.pad).toBe(40);
    expect(spec.children).toHaveLength(1);
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
    expect(layers.every((layer) => layer.props.variant === "body")).toBe(true);
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

  it("strips every attribute, including handlers, styles and links", () => {
    const spec = parseHtml(
      '<p id="hero" class="lead" style="color:red" onclick="steal()" href="javascript:void(0)" data-x="1">Copy</p>',
    ).regionSpec(null);
    const allowed = new Set(["text", "variant", "tone", "gap", "pad", "h", "label"]);
    const seen = new Set<string>();
    walk(spec, (layer) => {
      for (const key of Object.keys(layer.props)) seen.add(key);
    });
    expect([...seen].every((key) => allowed.has(key))).toBe(true);
    expect(firstOfType(spec, "text").props).toEqual({ text: "Copy", variant: "body" });
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
    expect(spec.props.pad).toBe(40);
    expect(spec.children).toHaveLength(1);
    const section = spec.children?.[0];
    expect(section?.type).toBe("stack");
    expect(texts(spec)).toEqual(["Region", "Copy"]);
    expect(JSON.stringify(spec)).not.toContain("End");
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
  it("caps the import at 200 layers", () => {
    const items = Array.from({ length: 260 }, (_, index) => `<p>Item ${index}</p>`).join("");
    const parsed = parseHtml(items);
    expect(parsed.layers).toBe(HTML_LIMITS.layers);
    expect(nodesOfType(parsed.regionSpec(null), "text")).toHaveLength(HTML_LIMITS.layers);
    expect(parsed.warnings.some((warning) => warning.includes("200 layers"))).toBe(true);
  });

  it("caps the scan at 500 source elements", () => {
    const empty = "<div></div>".repeat(520);
    const parsed = parseHtml(empty);
    expect(parsed.regions).toHaveLength(0);
    expect(parsed.warnings.some((warning) => warning.includes("500 source elements"))).toBe(true);
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
