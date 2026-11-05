import { describe, expect, it } from "vitest";
import { splitMarkdownChunks } from "./splitMarkdownChunks";

const para = (n: number) => `Paragraph ${String(n)} ${"word ".repeat(40)}`.trim();

describe("splitMarkdownChunks", () => {
  it("returns a single chunk for small content", () => {
    expect(splitMarkdownChunks("# Hi\n\nShort.", 8000)).toEqual(["# Hi\n\nShort."]);
  });

  it("splits large content at blank-line block boundaries", () => {
    const content = Array.from({ length: 40 }, (_, i) => para(i)).join("\n\n");
    const chunks = splitMarkdownChunks(content, 500);
    expect(chunks.length).toBeGreaterThan(1);
    // Reassembled chunks preserve every paragraph and never cut mid-line.
    expect(chunks.join("\n\n").replace(/\n+/g, "\n")).toContain("Paragraph 39");
    for (const chunk of chunks) expect(chunk.startsWith("Paragraph")).toBe(true);
  });

  it("never cuts inside a fenced code block", () => {
    const fence = "```js\n" + Array.from({ length: 30 }, (_, i) => `const x${String(i)} = ${String(i)};`).join("\n") + "\n```";
    const content = `${para(0)}\n\n${fence}\n\n${para(1)}`;
    const chunks = splitMarkdownChunks(content, 200);
    // The whole fence lands in exactly one chunk (opening and closing together).
    const withFence = chunks.filter((c) => c.includes("```js"));
    expect(withFence).toHaveLength(1);
    const fenced = withFence[0] ?? "";
    expect(fenced).toContain("```js");
    expect(fenced.match(/```/g)).toHaveLength(2);
  });

  it("keeps a document with footnotes whole", () => {
    const content = Array.from({ length: 40 }, (_, i) => para(i)).join("\n\n") + "\n\n[^a]: A footnote.";
    expect(splitMarkdownChunks(content, 300)).toHaveLength(1);
  });

  it("copies link reference definitions into every chunk", () => {
    const body = Array.from({ length: 40 }, (_, i) => `${para(i)} [see][ref]`).join("\n\n");
    const content = `${body}\n\n[ref]: https://example.com`;
    const chunks = splitMarkdownChunks(content, 400);
    expect(chunks.length).toBeGreaterThan(1);
    for (const chunk of chunks) expect(chunk).toContain("[ref]: https://example.com");
  });
});
