import { describe, expect, it } from "vitest";

import { i18n } from "../i18n";

import {
  assetDomain,
  assetGroup,
  assetVersion,
  formatRelative,
  timestampToDate,
} from "./data";

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
