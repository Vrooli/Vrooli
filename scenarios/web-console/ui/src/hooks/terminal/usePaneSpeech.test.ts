import { describe, expect, it } from "vitest";
import { ensureSpeechChunks } from "./usePaneSpeech";

describe("ensureSpeechChunks", () => {
  it("keeps short paragraphs intact", () => {
    expect(ensureSpeechChunks(["hello", "world"])).toEqual(["hello", "world"]);
  });

  it("splits Unicode-heavy paragraphs by UTF-8 bytes", () => {
    const chunks = ensureSpeechChunks(["界".repeat(2000)]);
    expect(chunks.length).toBeGreaterThan(1);
    expect(chunks.every((chunk) => new TextEncoder().encode(chunk).length <= 4500)).toBe(true);
  });
});
