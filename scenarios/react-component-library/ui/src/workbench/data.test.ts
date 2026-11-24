import { beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "../i18n";

const apiMocks = vi.hoisted(() => ({
  getCatalogAsset: vi.fn(),
  getComponentExperience: vi.fn(),
  listComponentStories: vi.fn(),
  listComponentTestReports: vi.fn(),
  listComponentVersions: vi.fn(),
}));

vi.mock("../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/components")>();
  return {
    ...actual,
    componentsClient: {
      ...actual.componentsClient,
      listComponentVersions: apiMocks.listComponentVersions,
    },
    getCatalogAsset: apiMocks.getCatalogAsset,
    getComponentExperience: apiMocks.getComponentExperience,
    listComponentStories: apiMocks.listComponentStories,
  };
});

vi.mock("../api/componentTests", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/componentTests")>();
  return { ...actual, listComponentTestReports: apiMocks.listComponentTestReports };
});

import {
  assetDomain,
  assetAdoptions,
  assetHasDraft,
  assetGroup,
  assetVersion,
  formatDay,
  formatNumber,
  formatPercent,
  formatRelative,
  formatTime,
  isCoverageWarming,
  loadComponentBundle,
  timestampToDate,
} from "./data";

beforeEach(() => {
  vi.resetAllMocks();
  apiMocks.getCatalogAsset.mockResolvedValue({
    component: { id: "cmp-button", latestVersion: "2.0.0", version: "2.0.0" },
  });
  apiMocks.listComponentVersions.mockResolvedValue({ versions: [] });
  apiMocks.listComponentStories.mockResolvedValue({ stories: [] });
  apiMocks.getComponentExperience.mockResolvedValue({});
  apiMocks.listComponentTestReports.mockResolvedValue([]);
});

describe("version-pinned component bundles", () => {
  it("requests and returns stories from the selected release rather than latest", async () => {
    apiMocks.getCatalogAsset.mockResolvedValue({
      component: { id: "cmp-button", latestVersion: "2.0.0", version: "2.0.0" },
    });
    apiMocks.listComponentVersions.mockResolvedValue({ versions: [] });
    apiMocks.listComponentStories.mockResolvedValue({
      stories: [
        {
          id: "story-contract-1.4.0",
          version: "1.4.0",
          storiesJson: '[{"id":"primary","name":"Primary"}]',
          argsJson: "{}",
          sourcePath: "Button.tsx",
        },
      ],
    });
    apiMocks.getComponentExperience.mockResolvedValue({});
    apiMocks.listComponentTestReports.mockResolvedValue([]);

    const bundle = await loadComponentBundle("cmp-button", "1.4.0");

    expect(apiMocks.listComponentStories).toHaveBeenCalledWith({
      componentId: "cmp-button",
      version: "1.4.0",
      limit: 200,
    });
    expect(bundle.stories).toHaveLength(1);
    expect(bundle.stories[0]).toMatchObject({ id: "primary", version: "1.4.0" });
  });

  it("keeps an unavailable pinned release empty instead of substituting latest", async () => {
    apiMocks.getCatalogAsset.mockResolvedValue({
      component: { id: "cmp-button", latestVersion: "2.0.0", version: "2.0.0" },
    });
    apiMocks.listComponentVersions.mockResolvedValue({ versions: [] });
    apiMocks.listComponentStories.mockResolvedValue({
      stories: [
        {
          id: "story-contract-2.0.0",
          version: "2.0.0",
          storiesJson: '[{"id":"primary","name":"Primary"}]',
          argsJson: "{}",
          sourcePath: "Button.tsx",
        },
      ],
    });
    apiMocks.getComponentExperience.mockResolvedValue({});
    apiMocks.listComponentTestReports.mockResolvedValue([]);

    const bundle = await loadComponentBundle("cmp-button", "1.4.0");

    expect(bundle.stories).toEqual([]);
    expect(bundle.partial).toContain("stories");
  });

  it("reports independently failed sources and preserves available data", async () => {
    apiMocks.listComponentVersions.mockRejectedValue(new Error("versions unavailable"));
    apiMocks.listComponentStories.mockRejectedValue(new Error("stories unavailable"));
    apiMocks.getComponentExperience.mockRejectedValue(new Error("experience unavailable"));
    apiMocks.listComponentTestReports.mockRejectedValue(new Error("tests unavailable"));

    const bundle = await loadComponentBundle("cmp-button");

    expect(bundle.partial).toEqual(["versions", "stories", "experience", "tests"]);
    expect(bundle.versions).toEqual([]);
    expect(bundle.stories).toEqual([]);
    expect(bundle.experience).toBeUndefined();
    expect(bundle.testReports).toEqual([]);
  });

  it("fails clearly when the catalog entry has no component", async () => {
    apiMocks.getCatalogAsset.mockResolvedValue({});
    await expect(loadComponentBundle("missing")).rejects.toThrow("Component not found");
    expect(apiMocks.listComponentStories).not.toHaveBeenCalled();
  });

  it("uses the newest available release and ignores malformed or duplicate declarations", async () => {
    apiMocks.getCatalogAsset.mockResolvedValue({ component: { id: "cmp-button", version: "" } });
    apiMocks.listComponentVersions.mockResolvedValue({
      versions: [{ version: "1.9.0" }, { version: "1.10.0" }] as never,
    });
    apiMocks.listComponentStories.mockResolvedValue({
      stories: [
        { id: "invalid-json", version: "1.10.0", storiesJson: "{", argsJson: "{}" },
        { id: "not-an-array", version: "1.10.0", storiesJson: "{}", argsJson: "{}" },
        {
          id: "latest-contract",
          version: "1.10.0",
          storiesJson:
            '[null,4,{"name":"missing id"},{"id":"primary","args":[],"role":2},{"id":"primary","name":"Duplicate"},{"id":"secondary","name":"Secondary","role":"example","args":{"size":"sm"}}]',
          argsJson: '{"size":"sm"}',
          sourcePath: "Button.tsx",
        },
        {
          id: "older-contract",
          version: "1.9.0",
          storiesJson: '[{"id":"older"}]',
          argsJson: "{}",
        },
      ],
    });

    const bundle = await loadComponentBundle("cmp-button");

    expect(bundle.versions.map(({ version }) => version)).toEqual(["1.10.0", "1.9.0"]);
    expect(bundle.stories).toEqual([
      expect.objectContaining({ id: "older", version: "1.9.0" }),
      expect.objectContaining({
        key: "1.10.0:primary",
        id: "primary",
        name: "primary",
        role: "story",
        args: {},
        contractId: "latest-contract",
      }),
      expect.objectContaining({
        key: "1.10.0:secondary",
        name: "Secondary",
        role: "example",
        args: { size: "sm" },
      }),
    ]);
    expect(bundle.partial).toEqual([]);
  });
});

describe("coverage recovery and display helpers", () => {
  it("recognizes warming errors without treating unrelated errors as transient", () => {
    expect(isCoverageWarming(new Error("Corpus is being prepared"))).toBe(true);
    expect(isCoverageWarming("Coverage warming up")).toBe(true);
    expect(isCoverageWarming(new Error("network unavailable"))).toBe(false);
    expect(isCoverageWarming({ message: "preparing" })).toBe(false);
  });

  it("projects draft and adoption metadata without inventing values", () => {
    expect(assetHasDraft({ draftVersion: "2.1.0", latestVersion: "2.0.0" } as never)).toBe(true);
    expect(assetHasDraft({ draftVersion: "2.0.0", latestVersion: "2.0.0" } as never)).toBe(false);
    expect(assetHasDraft({} as never)).toBe(false);
    expect(assetAdoptions({ metrics: { directAdoptionCount: 7 } } as never)).toBe(7);
    expect(assetAdoptions({} as never)).toBe(0);
  });

  it("formats day, time, numbers, percentages, and relative dates", () => {
    const today = new Date();
    expect(formatDay(today)).toMatch(/\d|today/i);
    expect(formatDay(new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1))).toMatch(
      /\d|yesterday/i,
    );
    expect(formatDay(new Date(2020, 0, 2))).toMatch(/\d/);
    expect(formatTime(new Date("2026-11-18T10:05:00Z"))).toMatch(/\d{2}:\d{2}/);
    expect(formatNumber(1234)).toMatch(/1.*234/);
    expect(formatPercent(0.125)).toBe("12.5%");
    expect(formatRelative(new Date(Date.now() + 5_000))).toBe(i18n.t("common.justNow"));
    expect(formatRelative(new Date(Date.now() - 120_000))).toMatch(/\d/);
  });
});

describe("catalog projections", () => {
  it("classifies component, hook and support assets", () => {
    expect(assetGroup({ catalogKind: "component" } as never)).toBe("components");
    expect(assetGroup({ catalogKind: "runtime-hook" } as never)).toBe("hooks");
    expect(assetGroup({ catalogKind: "foundation" } as never)).toBe("support");
  });

  it("falls back through domain and version fields", () => {
    expect(assetDomain({ catalogDomain: "controls", slot: "ui-primitive" } as never)).toBe(
      "controls",
    );
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
