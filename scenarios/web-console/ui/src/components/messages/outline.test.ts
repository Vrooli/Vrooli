import { describe, expect, it } from "vitest";
import { messageOutline, messagePreview } from "./outline";

describe("messageOutline", () => {
  it("[REQ:P0-017d] counts words, headings, and fenced code blocks", () => {
    const text = [
      "# Plan",
      "Some intro words here.",
      "## Steps",
      "```ts",
      "# not a heading inside code",
      "const x = 1;",
      "```",
      "### Done",
      "~~~",
      "more code",
      "~~~",
    ].join("\n");
    expect(messageOutline(text)).toEqual({ words: 26, headings: 3, codeBlocks: 2 });
  });

  it("returns zeros for empty text", () => {
    expect(messageOutline("   ")).toEqual({ words: 0, headings: 0, codeBlocks: 0 });
  });

  it("does not treat a hashtag without a space as a heading", () => {
    expect(messageOutline("#tag and #another").headings).toBe(0);
  });
});


describe("messagePreview", () => {
  it("preserves short markdown and plaintext verbatim", () => {
    const text = "# Heading\n\n```ts\nconst x = 1;\n```";
    expect(messagePreview(text)).toEqual({ source: text, markdown: text, plainText: "", truncated: false });
  });

  it("bounds both character-heavy and line-heavy source without touching the original", () => {
    for (const text of ["long prose ".repeat(8000), "x\n".repeat(8000)]) {
      const preview = messagePreview(text);
      expect(preview.truncated).toBe(true);
      expect(preview.source.length).toBeLessThanOrEqual(6000);
      expect(preview.source.split("\n").length).toBeLessThanOrEqual(80);
      expect(text.startsWith(preview.source)).toBe(true);
    }
  });

  it("keeps complete code blocks and renders a cut fence as a noninteractive excerpt", () => {
    const complete = "# Intro\n\n```ts\nconst x = 1;\n```\n\n";
    const text = complete + "~~~~python\n" + "print('hello')\n".repeat(1000) + "~~~~";
    const preview = messagePreview(text);
    expect(preview.markdown).toBe(complete);
    expect(preview.plainText).toContain("print('hello')");
    expect(preview.plainText).not.toContain("~~~~");
    expect(preview.source).toContain("~~~~python");
  });

  it("does not close a fence on a shorter or different marker", () => {
    const text = "````ts\n```\n~~~\n" + "code\n".repeat(1000);
    const preview = messagePreview(text);
    expect(preview.markdown).toBe("");
    expect(preview.plainText).toMatch(/^```\n~~~\n/);
  });
});
