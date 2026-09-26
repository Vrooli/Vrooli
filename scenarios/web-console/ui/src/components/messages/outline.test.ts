import { describe, expect, it } from "vitest";
import { messageOutline } from "./outline";

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
