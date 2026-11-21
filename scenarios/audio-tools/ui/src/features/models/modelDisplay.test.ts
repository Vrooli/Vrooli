import { describe, it, expect } from "vitest";
import {
  commercialLabelKey,
  commercialTone,
  languageSummary,
  readinessLabelKey,
  readinessTone,
} from "./modelDisplay";
import { strings } from "../../consts/strings";

describe("readiness tone mapping", () => {
  it.each([
    ["available", "success"],
    ["candidate", "info"],
    ["experimental", "warning"],
    ["watch", "neutral"],
    ["unspecified", "neutral"],
  ] as const)("maps %s to %s", (level, tone) => {
    expect(readinessTone(level)).toBe(tone);
    expect(readinessLabelKey(level)).toBe(strings.models.readiness[level]);
  });
});

describe("commercial tone mapping", () => {
  it.each([
    ["allowed", "success"],
    ["disallowed", "danger"],
    ["unknown", "warning"],
    ["unspecified", "neutral"],
  ] as const)("maps %s to %s", (level, tone) => {
    expect(commercialTone(level)).toBe(tone);
    expect(commercialLabelKey(level)).toBe(strings.models.commercial[level]);
  });
});

describe("languageSummary", () => {
  it("keeps the first few languages and counts the overflow", () => {
    expect(languageSummary(["en", "es", "fr", "de", "it"])).toEqual({
      shown: ["en", "es", "fr"],
      overflow: 2,
    });
  });

  it("reports no overflow when the list fits", () => {
    expect(languageSummary(["en", "es"])).toEqual({
      shown: ["en", "es"],
      overflow: 0,
    });
  });
});
