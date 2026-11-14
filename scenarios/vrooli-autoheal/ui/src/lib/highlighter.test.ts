import { describe, expect, it, vi } from "vitest";

const { createHighlighter, highlighter } = vi.hoisted(() => {
  const instance = {
    getLoadedLanguages: vi.fn(() => ["json"]),
    loadLanguage: vi.fn().mockRejectedValue(new Error("unsupported language")),
    codeToHtml: vi.fn((code: string, options: { lang: string }) => `${options.lang}:${code}`),
  };
  return { createHighlighter: vi.fn(async () => instance), highlighter: instance };
});

vi.mock("shiki", () => ({ createHighlighter }));

import { getHighlighter, highlightCode } from "./highlighter";

describe("code highlighting", () => {
  it("shares an in-flight highlighter creation", async () => {
    const [first, second] = await Promise.all([getHighlighter(), getHighlighter()]);
    expect(first).toBe(second);
    expect(createHighlighter).toHaveBeenCalledTimes(1);
  });

  it("uses loaded languages and falls back when loading fails", async () => {
    await expect(highlightCode("{}", "json")).resolves.toBe("json:{}");
    await expect(highlightCode("plain", "unknown")).resolves.toBe("text:plain");
    expect(highlighter.loadLanguage).toHaveBeenCalledWith("unknown");
  });
});
