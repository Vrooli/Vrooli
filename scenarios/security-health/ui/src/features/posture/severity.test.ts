import { describe, expect, it } from "vitest";

import { Severity } from "../../api/validation";
import { compareSeverity, severityMeta } from "./severity";

describe("severity metadata", () => {
  it("orders ERROR before WARNING before INFO", () => {
    const order = [Severity.INFO, Severity.ERROR, Severity.WARNING].sort(compareSeverity);
    expect(order).toEqual([Severity.ERROR, Severity.WARNING, Severity.INFO]);
  });

  it("maps every severity to a distinct chip + label key", () => {
    expect(severityMeta(Severity.ERROR).chipClass).toContain("red");
    expect(severityMeta(Severity.WARNING).chipClass).toContain("amber");
    expect(severityMeta(Severity.INFO).chipClass).toContain("slate");
  });

  it("falls back to the INFO label for an unspecified severity", () => {
    expect(severityMeta(Severity.UNSPECIFIED).labelKey).toBe(severityMeta(Severity.INFO).labelKey);
  });
});
