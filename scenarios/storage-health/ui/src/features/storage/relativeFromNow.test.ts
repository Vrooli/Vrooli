import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { setLocale } from "../../i18n";
import { relativeFromNow } from "./relativeFromNow";

describe("relativeFromNow", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(async () => {
    await setLocale("en");
  });

  it("returns an empty string for an empty value", () => {
    expect(relativeFromNow("")).toBe("");
  });

  it("returns an empty string for an unparseable value", () => {
    expect(relativeFromNow("not-a-date")).toBe("");
  });

  it("renders seconds-ago for a recent timestamp", () => {
    const out = relativeFromNow(new Date(Date.now() - 5_000).toISOString());
    expect(out.toLowerCase()).toContain("second");
  });

  it("renders minutes-ago", () => {
    const out = relativeFromNow(new Date(Date.now() - 5 * 60_000).toISOString());
    expect(out.toLowerCase()).toContain("minute");
  });

  it("renders hours-ago", () => {
    const out = relativeFromNow(new Date(Date.now() - 5 * 3_600_000).toISOString());
    expect(out.toLowerCase()).toContain("hour");
  });

  it("renders days-ago", () => {
    const out = relativeFromNow(new Date(Date.now() - 3 * 86_400_000).toISOString());
    expect(out.toLowerCase()).toContain("day");
  });
});
