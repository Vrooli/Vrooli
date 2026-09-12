import { describe, expect, it } from "vitest";

import { DICTATION_SCRIPTS, findDictationScript } from "./scripts";

describe("DICTATION_SCRIPTS", () => {
  it("ships a varied built-in prompt pack", () => {
    expect(DICTATION_SCRIPTS.length).toBeGreaterThanOrEqual(12);

    const ids = new Set(DICTATION_SCRIPTS.map((script) => script.id));
    expect(ids.size).toBe(DICTATION_SCRIPTS.length);

    for (const script of DICTATION_SCRIPTS) {
      expect(script.title.trim()).not.toBe("");
      expect(script.purpose.trim()).not.toBe("");
      expect(script.text.trim()).not.toBe("");
      expect(script.tags.length).toBeGreaterThan(0);
    }
  });

  it("finds scripts by stable id", () => {
    expect(findDictationScript(DICTATION_SCRIPTS[0]!.id)).toBe(DICTATION_SCRIPTS[0]);
    expect(findDictationScript("missing")).toBeNull();
  });
});
