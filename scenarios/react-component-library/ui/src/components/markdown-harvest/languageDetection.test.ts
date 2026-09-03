import { describe, expect, it } from "vitest";

import { languageLabel, normalizeCodeLanguage, remarkProsePaths } from "@vrooli/react-component-library/markdown-renderer/0.4.2";

describe("markdown language detection", () => {
  it("normalizes aliases and malformed language values", () => {
    expect(normalizeCodeLanguage(" TS ")).toBe("typescript");
    expect(normalizeCodeLanguage("shell")).toBe("bash");
    expect(normalizeCodeLanguage("")).toBe("text");
  });

  it("provides readable labels and an inert prose plugin", () => {
    expect(languageLabel()).toBe("Plain text");
    expect(languageLabel("go")).toBe("GO");
    expect(remarkProsePaths()()).toBeUndefined();
  });
});
