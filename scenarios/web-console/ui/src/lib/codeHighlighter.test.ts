import { beforeEach, describe, expect, it, vi } from "vitest";

const { createHighlighter } = vi.hoisted(() => ({ createHighlighter: vi.fn() }));
vi.mock("shiki", () => ({ createHighlighter }));

import { getCodeHighlighter, getLanguageFromPath, highlightCode } from "./codeHighlighter";

describe("code highlighter seams", () => {
  beforeEach(() => {
    createHighlighter.mockReset();
  });

  it("maps named files, extensions, and unknown paths", () => {
    expect(getLanguageFromPath("Dockerfile.prod")).toBe("dockerfile");
    expect(getLanguageFromPath("GNUmakefile")).toBe("makefile");
    expect(getLanguageFromPath(".env.local")).toBe("dotenv");
    expect(getLanguageFromPath("src/App.tsx")).toBe("tsx");
    expect(getLanguageFromPath("README")).toBeNull();
    expect(getLanguageFromPath("")).toBeNull();
    expect(getLanguageFromPath("thing.unknown")).toBeNull();
  });

  it("loads once, falls back to plaintext for unavailable languages, and recovers tokenization errors", async () => {
    const highlighter = {
      getLoadedLanguages: vi.fn().mockReturnValue(["typescript"]),
      loadLanguage: vi.fn(),
      codeToTokens: vi.fn().mockReturnValue({ tokens: [[{ content: "x", color: "#fff", fontStyle: 1 }, { content: "y", fontStyle: 2 }, { content: "z", fontStyle: 0 }]] }),
    };
    createHighlighter.mockResolvedValue(highlighter);
    await expect(getCodeHighlighter()).resolves.toBe(highlighter);
    await expect(getCodeHighlighter()).resolves.toBe(highlighter);
    await expect(highlightCode("xyz", "typescript")).resolves.toEqual([{ lineNumber: 1, tokens: [{ content: "x", color: "#fff", fontStyle: "italic" }, { content: "y", fontStyle: "bold" }, { content: "z", fontStyle: undefined }] }]);
    expect(createHighlighter).toHaveBeenCalledTimes(1);

    highlighter.getLoadedLanguages.mockReturnValue([]);
    highlighter.loadLanguage.mockRejectedValueOnce(new Error("not bundled"));
    highlighter.codeToTokens.mockReturnValueOnce({ tokens: [[{ content: "plain" }]] });
    await expect(highlightCode("plain", "ruby")).resolves.toEqual([{ lineNumber: 1, tokens: [{ content: "plain", color: undefined, fontStyle: undefined }] }]);
    highlighter.codeToTokens.mockImplementationOnce(() => { throw new Error("tokenizer failure"); });
    await expect(highlightCode("a\nb", null)).resolves.toEqual([{ lineNumber: 1, tokens: [{ content: "a" }] }, { lineNumber: 2, tokens: [{ content: "b" }] }]);
  });
});
