import { describe, expect, it } from "vitest";

import { i18n } from "../i18n";
import type { CoverageReport } from "@vrooli/proto-types/react-component-library/v1/catalog/catalog_pb";

import {
  activityFromCatalog,
  assetDomain,
  assetGroup,
  assetVersion,
  findingStats,
  findingsFromCoverage,
  formatRelative,
  timestampToDate,
} from "./data";

function report(overrides: Partial<CoverageReport> = {}): CoverageReport {
  return {
    rows: [],
    totals: {},
    byDomain: [],
    byPriority: [],
    maturity: undefined,
    compositionScores: {},
    compositionMedian: 0,
    bespokeEscapeCount: 0,
    declaredCapabilityAssetCount: 0,
    declaredUncheckableAssetCount: 0,
    unmeasuredCapabilityAssetCount: 0,
    capabilityDeclarationCount: 0,
    capabilityCoverage: [],
    compositionBlockedAssetCount: 0,
    ...overrides,
  } as CoverageReport;
}

describe("findings projection", () => {
  it("turns real failed gates into ranked findings", () => {
    const findings = findingsFromCoverage(
      report({
        rows: [
          {
            assetId: "controls.button",
            name: "Button",
            domain: "controls",
            failedGates: ["visual", "types"],
            nearestBlockingGate: "visual",
            blocksDownstream: 12,
            assetScore: 70,
            achieved: "implemented",
            target: "production-ready",
            rungName: "component",
            newestEvidence: "2026-11-18T10:00:00Z",
            kind: "component",
          },
          {
            assetId: "overlays.bar",
            name: "OverlayActionBar",
            domain: "overlays",
            failedGates: ["types"],
            nearestBlockingGate: "types",
            blocksDownstream: 2,
            assetScore: 80,
            achieved: "scaffolded",
            target: "verified",
            rungName: "composition",
            newestEvidence: "",
            kind: "component",
          },
        ] as unknown as CoverageReport["rows"],
      }),
    );
    expect(findings).toHaveLength(2);
    expect(findings[0]?.assetName).toBe("Button");
    expect(findings[0]?.severity).toBe("blocking");
    expect(findings[0]?.failedCount).toBe(2);
    expect(findings[0]?.secondaryGates).toEqual(["types"]);
    expect(findings[1]?.assetName).toBe("OverlayActionBar");
    expect(findingStats(findings)).toMatchObject({
      open: 2,
      blocking: 2,
      warning: 0,
      affectedAssets: 2,
    });
  });

  it("returns no findings for an absent report rather than inventing an empty pass", () => {
    expect(findingsFromCoverage(undefined)).toEqual([]);
    expect(findingStats([]).open).toBe(0);
  });
});

describe("catalog projections", () => {
  it("classifies component, hook and support assets", () => {
    expect(assetGroup({ catalogKind: "component" } as never)).toBe("components");
    expect(assetGroup({ catalogKind: "runtime-hook" } as never)).toBe("hooks");
    expect(assetGroup({ catalogKind: "foundation" } as never)).toBe("support");
  });

  it("falls back through domain and version fields", () => {
    expect(assetDomain({ catalogDomain: "controls", slot: "ui-primitive" } as never)).toBe("controls");
    expect(assetDomain({ catalogDomain: "", slot: "ui-primitive" } as never)).toBe("ui-primitive");
    expect(assetVersion({ latestVersion: "2.2.14", version: "2.0.0" } as never)).toBe("2.2.14");
    expect(assetVersion({} as never)).toBe("unversioned");
  });

  it("derives activity from catalog timestamps and labels the actor", () => {
    const events = activityFromCatalog([
      {
        id: "asset-1",
        displayName: "Button",
        slug: "Button",
        libraryId: "react-component-library:Button",
        catalogId: "controls.button",
        latestVersion: "2.2.14",
        updatedAt: { seconds: 1_700_000_000, nanos: 0 },
        indexedAt: "2026-11-18T10:00:00Z",
      } as never,
    ]);
    expect(events.length).toBeGreaterThan(0);
    expect(events[0]?.actor).toBe("catalog");
    expect(events.every((event) => event.at instanceof Date)).toBe(true);
  });
});

describe("timestamp normalization", () => {
  it("handles ISO strings, Dates, protobuf Timestamps and empty values", () => {
    expect(timestampToDate("2026-11-18T10:00:00Z")?.toISOString()).toBe("2026-11-18T10:00:00.000Z");
    expect(timestampToDate({ seconds: 1_700_000_000 })?.getTime()).toBe(1_700_000_000_000);
    expect(timestampToDate(new Date(0))?.getTime()).toBe(0);
    expect(timestampToDate(undefined)).toBeUndefined();
    expect(timestampToDate("not a date")).toBeUndefined();
  });

  it("never formats an invalid date as a relative value", () => {
    // The helper goes through the catalog, so the assertion follows the
    // active locale (cimode in tests) instead of pinning English copy.
    const unknown = i18n.t("common.relativeUnknown");
    expect(formatRelative(new Date(Number.NaN))).toBe(unknown);
    expect(formatRelative(undefined)).toBe(unknown);
    expect(formatRelative(new Date(Number.NaN))).not.toMatch(/\d/);
  });
});
