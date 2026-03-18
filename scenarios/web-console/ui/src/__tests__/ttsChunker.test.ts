import { describe, it, expect } from "vitest";
import { splitIntoParagraphs, isSpeakable, TTS_MAX_CHUNK_LENGTH } from "../lib/ttsChunker";

describe("splitIntoParagraphs", () => {
  it("returns short text as a single chunk", () => {
    expect(splitIntoParagraphs("Hello world")).toEqual(["Hello world"]);
  });

  it("splits on double newlines", () => {
    const text = "Paragraph one.\n\nParagraph two.\n\nParagraph three.";
    expect(splitIntoParagraphs(text)).toEqual([
      "Paragraph one.",
      "Paragraph two.",
      "Paragraph three.",
    ]);
  });

  it("splits long blocks on single newlines", () => {
    const line = "x".repeat(300);
    const block = `${line}\n${line}`;
    // block is 601 chars, > 500, so split on \n
    const result = splitIntoParagraphs(block);
    expect(result).toEqual([line, line]);
  });

  it("filters empty paragraphs", () => {
    expect(splitIntoParagraphs("a\n\n\n\nb")).toEqual(["a", "b"]);
  });

  // --- Regression tests for long-message TTS bug ---

  it("splits a single long line without newlines into chunks under the limit", () => {
    // This is the primary bug: a 6000-char string with no newlines
    // would previously be sent as-is, exceeding the 5000-char backend limit
    const longText = "word ".repeat(1200).trim(); // ~6000 chars
    const result = splitIntoParagraphs(longText);
    for (const chunk of result) {
      expect(chunk.length).toBeLessThanOrEqual(TTS_MAX_CHUNK_LENGTH);
    }
    expect(result.length).toBeGreaterThan(1);
    // Verify no content is lost (allow for whitespace differences)
    const rejoined = result.join(" ");
    expect(rejoined.replace(/\s+/g, " ")).toEqual(longText.replace(/\s+/g, " "));
  });

  it("splits a long paragraph with sentences into sentence-boundary chunks", () => {
    // Each sentence is ~68 chars, 100 sentences = ~6800 chars (over 4500 limit)
    const sentence = "This is a moderately long sentence that helps us test the chunker. ";
    const longText = sentence.repeat(100).trim();
    const result = splitIntoParagraphs(longText);
    for (const chunk of result) {
      expect(chunk.length).toBeLessThanOrEqual(TTS_MAX_CHUNK_LENGTH);
    }
    expect(result.length).toBeGreaterThan(1);
  });

  it("hard-splits text with no sentence boundaries and no spaces", () => {
    const longText = "x".repeat(10000);
    const result = splitIntoParagraphs(longText);
    for (const chunk of result) {
      expect(chunk.length).toBeLessThanOrEqual(TTS_MAX_CHUNK_LENGTH);
    }
    expect(result.length).toBeGreaterThan(1);
    expect(result.join("")).toEqual(longText);
  });

  it("handles mixed content: short paragraphs + one very long paragraph", () => {
    const shortP = "Short paragraph.";
    const longP = "Long sentence here. ".repeat(300).trim(); // ~6000 chars
    const text = `${shortP}\n\n${longP}\n\n${shortP}`;
    const result = splitIntoParagraphs(text);
    // First and last should be short
    expect(result[0]).toEqual(shortP);
    expect(result[result.length - 1]).toEqual(shortP);
    // All chunks within limit
    for (const chunk of result) {
      expect(chunk.length).toBeLessThanOrEqual(TTS_MAX_CHUNK_LENGTH);
    }
  });

  it("handles text exactly at the limit", () => {
    const text = "a".repeat(TTS_MAX_CHUNK_LENGTH);
    const result = splitIntoParagraphs(text);
    expect(result).toEqual([text]);
  });

  it("handles text one char over the limit", () => {
    const text = "a".repeat(TTS_MAX_CHUNK_LENGTH + 1);
    const result = splitIntoParagraphs(text);
    for (const chunk of result) {
      expect(chunk.length).toBeLessThanOrEqual(TTS_MAX_CHUNK_LENGTH);
    }
    expect(result.join("")).toEqual(text);
  });

  // --- Regression tests for non-speakable chunk filtering ---

  it("filters out horizontal rules (---)", () => {
    const text = "Paragraph one.\n\n---\n\nParagraph two.";
    expect(splitIntoParagraphs(text)).toEqual([
      "Paragraph one.",
      "Paragraph two.",
    ]);
  });

  it("filters out code fences", () => {
    const text = "Before code.\n\n```typescript\n\nAfter code.";
    expect(splitIntoParagraphs(text)).toEqual([
      "Before code.",
      "After code.",
    ]);
  });

  it("filters out lone list markers", () => {
    const text = "Intro.\n\n*\n\n- \n\nContent here.";
    expect(splitIntoParagraphs(text)).toEqual([
      "Intro.",
      "Content here.",
    ]);
  });

  it("keeps headings with text content", () => {
    const text = "# My Heading\n\nParagraph.";
    const result = splitIntoParagraphs(text);
    expect(result).toContain("# My Heading");
  });

  it("keeps list items with text content", () => {
    const text = "Intro.\n\n- Item one\n- Item two";
    const result = splitIntoParagraphs(text);
    expect(result.some((c) => c.includes("Item one"))).toBe(true);
  });

  it("returns original text when all chunks are non-speakable", () => {
    // Edge case: if everything is filtered, fall back to original
    expect(splitIntoParagraphs("---")).toEqual(["---"]);
  });
});

describe("isSpeakable", () => {
  it.each([
    ["Hello world", true],
    ["# Heading", true],
    ["- Item text", true],
    ["Some code: x = 1", true],
    ["---", false],
    ["***", false],
    ["___", false],
    ["```", false],
    ["```typescript", false],
    ["* ", false],
    ["-", false],
    ["+", false],
    ["> ", false],
    [".", false],
    ["...", false],  // no word characters → filtered (Kokoro can synthesize it, but it's not useful speech)
  ])("isSpeakable(%j) → %s", (input, expected) => {
    expect(isSpeakable(input)).toBe(expected);
  });
});
