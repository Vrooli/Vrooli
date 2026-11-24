import { describe, expect, it } from "vitest";

import { parseHtml } from "./htmlImport";
import { walk, type TemplateNode } from "../templates/spec";

describe("HTML import edge behavior", () => {
  it("keeps local visual references while removing remote source-preview fetches", () => {
    const parsed = parseHtml(
      '<article data-name="Featured_card">' +
        '<img src="data:image/png;base64,AAAA" srcset="data:image/png;base64,AAAA 1x" alt="Local art">' +
        '<img src="https://remote.invalid/pixel.png" srcset="https://remote.invalid/pixel.png 2x">' +
        '<a href="#details">Local anchor</a><a href="https://remote.invalid/page">Remote link</a>' +
        '<div srcdoc="<script>remoteCode()</script>">Local copy</div>' +
        "</article>",
    );
    const region = parsed.regions.find((candidate) => candidate.sourceHint === "Featured card");
    if (!region) throw new Error("Expected the named source region to be discovered.");

    const preview = parsed.sourcePreview(region.id);

    expect(preview).toContain('src="data:image/png;base64,AAAA"');
    expect(preview).toContain('srcset="data:image/png;base64,AAAA 1x"');
    expect(preview).toContain('href="#details"');
    expect(preview).toContain("Local copy");
    expect(preview).not.toContain("remote.invalid");
    expect(preview).not.toContain("srcdoc=");
    expect(preview).not.toContain("remoteCode");
  });

  it("ignores an unsupported stylesheet selector and still applies valid rules", () => {
    const parsed = parseHtml('<p class="copy">Selector recovery</p>', undefined, [
      ":not( { color: #f00; } .copy { color: #123456; }",
    ]);
    const text = parsed.regionSpec(null).children?.[0];

    expect(text?.type).toBe("text");
    expect(JSON.parse(String(text?.props.css))).toMatchObject({ color: "#123456" });
    expect(text?.props.text).toBe("Selector recovery");
  });

  it("omits later editable styles after the aggregate CSS budget is reached", () => {
    const value = "x".repeat(160);
    const style = `--accent:${value};--surface:${value};--outline:${value}`;
    const source = Array.from(
      { length: 640 },
      (_, index) => `<p style="${style}">Item ${index}</p>`,
    ).join("");
    const parsed = parseHtml(source);
    const imported: TemplateNode[] = [];
    walk(parsed.regionSpec(null), (layer) => {
      if (typeof layer.props.text === "string") imported.push(layer);
    });
    const first = imported.find((layer) => layer.props.text === "Item 0");
    const last = imported.find((layer) => layer.props.text === "Item 639");

    expect(parsed.warnings).toContain(
      "Some source styling was omitted after reaching the editable CSS size limit.",
    );
    expect(JSON.parse(String(first?.props.css))).toMatchObject({ "--accent": value });
    expect(JSON.parse(String(last?.props.css))).not.toHaveProperty("--accent");
  });
});
