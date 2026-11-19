/**
 * App tests — route resolution and shell continuity for the Nocturne
 * workbench. The page-level contracts are covered by page stories; this file
 * proves the shell mounts once and each public route resolves to its page.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "./test-utils";

const emitShortcutIntentMock = vi.fn();

vi.mock("@vrooli/iframe-bridge", () => ({
  emitShortcutIntent: (...args: unknown[]) => emitShortcutIntentMock(...args),
}));

const fixtures = vi.hoisted(() => ({ asset: {
  id: "asset-button",
  libraryId: "react-component-library:Button",
  displayName: "Button",
  description: "The primary action control.",
  version: "2.2.14",
  latestVersion: "2.2.14",
  slug: "Button",
  slot: "ui-primitive",
  catalogId: "controls.button",
  catalogKind: "component",
  catalogDomain: "controls",
  assetKind: "ASSET_KIND_COMPONENT",
  tags: ["control"],
  metrics: { directAdoptionCount: 8, effectiveAdoptionCount: 80, versionCount: 2 },
  dependencies: [],
  designStyles: [],
  updatedAt: "2026-11-18T12:00:00Z",
  indexedAt: "2026-11-18T11:00:00Z",
} }));

vi.mock("./api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/components")>();
  return {
    ...actual,
    listCatalogAssets: vi.fn().mockResolvedValue({ components: [fixtures.asset] }),
    getCatalogAsset: vi.fn().mockResolvedValue({ component: fixtures.asset }),
    listComponentStories: vi.fn().mockResolvedValue({ stories: [] }),
    getComponentExperience: vi.fn().mockRejectedValue(new Error("not available")),
    componentsClient: {
      listComponents: vi.fn().mockResolvedValue({ components: [fixtures.asset] }),
      getComponent: vi.fn().mockResolvedValue({ component: fixtures.asset }),
      getComponentContent: vi.fn().mockRejectedValue(new Error("not available")),
    },
  };
});

vi.mock("./api/catalog", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/catalog")>();
  return {
    ...actual,
    getCatalogCoverage: vi.fn().mockResolvedValue({
      rows: [
        {
          assetId: "controls.button",
          name: "Button",
          domain: "controls",
          failedGates: ["visual"],
          nearestBlockingGate: "visual",
          blocksDownstream: 4,
          assetScore: 80,
          achieved: "implemented",
          target: "production-ready",
          rungName: "component",
          newestEvidence: "2026-11-18T10:00:00Z",
          kind: "component",
        },
      ],
      maturity: {
        total: 1,
        atOrAboveTarget: 0,
        byRung: { implemented: 1 },
        productionReadyCoverage: { numerator: 0, denominator: 1, ratio: 0 },
        passEvidence: 3,
        failEvidence: 1,
        unmeasuredEvidence: 0,
        byGate: [],
      },
      byDomain: [],
      byPriority: [],
      totals: {},
    }),
    listCatalogNextWork: vi.fn().mockResolvedValue({ rows: [] }),
    describeCapabilities: vi.fn().mockResolvedValue({ definitions: [], states: [] }),
  };
});

vi.mock("./api/componentTests", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/componentTests")>();
  return {
    ...actual,
    listComponentTestReports: vi.fn().mockResolvedValue([]),
    runComponentTest: vi.fn().mockResolvedValue({ id: "report-1", verdict: "passed", results: [] }),
  };
});

vi.mock("./api/versions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/versions")>();
  return {
    ...actual,
    versionsClient: {
      ...actual.versionsClient,
      listVersions: vi.fn().mockResolvedValue({ versions: [] }),
    },
  };
});

import App from "./App";

describe("workbench routes", () => {
  afterEach(() => {
    cleanup();
    emitShortcutIntentMock.mockClear();
  });

  it("mounts the Nocturne shell and Home at /", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await waitFor(() => {
      expect(screen.getByTestId("workbench-shell")).toBeInTheDocument();
      expect(screen.getByTestId("home-page")).toBeInTheDocument();
    });
    expect(screen.getAllByRole("link", { name: /components/i }).length).toBeGreaterThan(0);
  });

  it("redirects the legacy catalog route to Components", async () => {
    renderWithProviders(<App />, { routerEntries: ["/catalog"] });
    await waitFor(() => {
      expect(screen.getByTestId("components-page")).toBeInTheDocument();
    });
  });

  it("resolves Components, Review and Settings", async () => {
    renderWithProviders(<App />, { routerEntries: ["/components"] });
    await waitFor(() => expect(screen.getByTestId("components-page")).toBeInTheDocument());
    cleanup();

    renderWithProviders(<App />, { routerEntries: ["/review"] });
    await waitFor(() => expect(screen.getByTestId("review-page")).toBeInTheDocument());
    cleanup();

    renderWithProviders(<App />, { routerEntries: ["/settings"] });
    await waitFor(() => expect(screen.getByTestId("settings-page")).toBeInTheDocument());
  });

  it("keeps one shell while navigating between pages", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await waitFor(() => expect(screen.getByTestId("home-page")).toBeInTheDocument());
    const shell = screen.getByTestId("workbench-shell");
    const componentsLink = screen.getAllByRole("link", { name: /components/i })[0];
    if (!componentsLink) throw new Error("the shell must expose a components link");
    fireEvent.click(componentsLink);
    await waitFor(() => expect(screen.getByTestId("components-page")).toBeInTheDocument());
    expect(screen.getByTestId("workbench-shell")).toBe(shell);
  });

  it("renders an unknown route as the workbench not-found page", async () => {
    renderWithProviders(<App />, { routerEntries: ["/does-not-exist"] });
    await waitFor(() => {
      expect(screen.getByTestId("not-found-page")).toBeInTheDocument();
    });
  });
});
