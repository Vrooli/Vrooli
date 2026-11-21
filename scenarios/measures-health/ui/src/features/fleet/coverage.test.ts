import { describe, expect, it } from "vitest";

import { DomainStatus, Tier } from "../../api/fleet";
import { compareDomainStatus, domainStatusMeta, tierMeta } from "./coverage";
import { strings } from "../../consts/strings";

describe("coverage presentation metadata", () => {
  it("maps each domain status to a stable label + chip", () => {
    expect(domainStatusMeta(DomainStatus.UNCOVERED).labelKey).toBe(strings.fleet.status.uncovered);
    expect(domainStatusMeta(DomainStatus.WAIVED).labelKey).toBe(strings.fleet.status.waived);
    expect(domainStatusMeta(DomainStatus.COVERED).labelKey).toBe(strings.fleet.status.covered);
    expect(domainStatusMeta(DomainStatus.NOT_EXPECTED).labelKey).toBe(strings.fleet.status.notExpected);
  });

  it("falls back to the neutral chip for an unspecified status", () => {
    expect(domainStatusMeta(DomainStatus.UNSPECIFIED).labelKey).toBe(strings.fleet.status.notExpected);
  });

  it("maps each tier to a stable label", () => {
    expect(tierMeta(Tier.FULL).labelKey).toBe(strings.fleet.tier.full);
    expect(tierMeta(Tier.PARTIAL).labelKey).toBe(strings.fleet.tier.partial);
    expect(tierMeta(Tier.FALLBACK).labelKey).toBe(strings.fleet.tier.fallback);
    expect(tierMeta(Tier.UNSPECIFIED).labelKey).toBe(strings.fleet.tier.none);
  });

  it("orders domains by urgency (UNCOVERED first, NOT_EXPECTED last)", () => {
    const ordered = [
      DomainStatus.NOT_EXPECTED,
      DomainStatus.COVERED,
      DomainStatus.UNCOVERED,
      DomainStatus.WAIVED,
    ].sort(compareDomainStatus);

    expect(ordered).toEqual([
      DomainStatus.UNCOVERED,
      DomainStatus.WAIVED,
      DomainStatus.COVERED,
      DomainStatus.NOT_EXPECTED,
    ]);
  });

  it("grades tiers worst-last so the strongest sorts first", () => {
    expect(tierMeta(Tier.FULL).order).toBeLessThan(tierMeta(Tier.PARTIAL).order);
    expect(tierMeta(Tier.PARTIAL).order).toBeLessThan(tierMeta(Tier.FALLBACK).order);
  });
});
