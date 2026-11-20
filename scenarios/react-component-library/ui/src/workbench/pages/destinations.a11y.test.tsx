import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { DesignIndexPage } from "./DesignIndexPage";
import { LibraryPage } from "./LibraryPage";
import { ReviewPage } from "./ReviewPage";
import { StartPage } from "./StartPage";
import { StatsPage } from "./StatsPage";
import { WorkspaceStudioPage } from "./WorkspaceStudioPage";

const fixtures = vi.hoisted(() => ({ workspace: {
  $typeName: "vrooli.react_component_library.v1.workspace.Workspace",
  id: "ws_harbor",
  name: "Harbor onboarding",
  summary: "Four-step onboarding",
  provenance: 2,
  tags: ["onboarding"],
  bindingScenario: "",
  bindingPage: "",
  themeJson: "{}",
  optionsJson: "{}",
  revision: 2,
  createdAt: "2026-11-19T10:00:00Z",
  updatedAt: "2026-11-19T11:00:00Z",
  originWorkspaceId: "",
  pages: [
    {
      $typeName: "vrooli.react_component_library.v1.workspace.WorkspacePage",
      id: "pg_welcome",
      name: "Welcome",
      route: "/onboarding",
      pageTypeId: "",
      sortOrder: 0,
      specJson: "{}",
      displays: [
        {
          $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceDisplay",
          id: "dp_phone",
          device: 3,
          width: 390,
          height: 844,
          sortOrder: 0,
          states: [
            {
              $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceState",
              id: "st_default",
              name: "Default",
              argsJson: "{}",
              scenarioDataJson: "{}",
              status: 2,
              captureId: "",
              sortOrder: 0,
            },
          ],
        },
      ],
    },
  ],
} }));

vi.mock("../../workbench/data", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../workbench/data")>();
  const asset = {
    id: "asset-button",
    libraryId: "react-component-library:Button",
    displayName: "Button",
    description: "Primary action.",
    version: "2.2.14",
    latestVersion: "2.2.14",
    slug: "Button",
    slot: "ui-primitive",
    catalogId: "controls.button",
    catalogKind: "component",
    catalogDomain: "controls",
    tags: ["control"],
    metrics: { directAdoptionCount: 8, effectiveAdoptionCount: 80, versionCount: 2 },
    dependencies: [],
    designStyles: [],
    updatedAt: "2026-11-18T12:00:00Z",
  };
  return {
    ...actual,
    useCatalog: () => ({ data: [asset], isLoading: false, isError: false }),
    useCoverage: () => ({
      data: {
        rows: [],
        totals: {},
        byDomain: [],
        byPriority: [],
        maturity: { productionReadyCoverage: { numerator: 0, denominator: 0, ratio: 0 } },
      },
      isLoading: false,
      isError: false,
    }),
    useCapabilities: () => ({ data: { states: [] }, isLoading: false, isError: false }),
    useNextWork: () => ({ data: { rows: [] }, isLoading: false, isError: false }),
    useDesignScenarios: () => ({ data: [], isLoading: false, isError: false }),
    useDesignPages: () => ({ data: [], isLoading: false, isError: false }),
  };
});

vi.mock("../../api/workspaces", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/workspaces")>();
  return {
    ...actual,
    listWorkspaces: vi.fn().mockResolvedValue([
      {
        id: "ws_harbor",
        name: "Harbor onboarding",
        summary: "Four-step onboarding",
        provenance: 2,
        tags: ["onboarding"],
        pageCount: 2,
        displayCount: 3,
        stateCount: 4,
        revision: 2,
        createdAt: "2026-11-19T10:00:00Z",
        updatedAt: "2026-11-19T11:00:00Z",
      },
    ]),
    getWorkspace: vi.fn().mockResolvedValue(fixtures.workspace),
    listWorkspaceAssets: vi.fn().mockResolvedValue([]),
    forkWorkspace: vi.fn(),
    createWorkspace: vi.fn(),
    saveWorkspace: vi.fn(),
  };
});

vi.mock("../../api/review", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/review")>();
  return {
    ...actual,
    getReviewSummary: vi.fn().mockResolvedValue({
      open: 0,
      blockers: 0,
      high: 0,
      acknowledged: 0,
      accepted: 0,
      fixed: 0,
    }),
    listFindings: vi.fn().mockResolvedValue([]),
    listFindingsPage: vi.fn().mockResolvedValue({ findings: [], total: 0 }),
  };
});

vi.mock("../../api/activity", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/activity")>();
  return {
    ...actual,
    listActivity: vi.fn().mockResolvedValue([]),
    getActivitySummary: vi.fn().mockResolvedValue({
      total: 0,
      empty: true,
      byDay: [],
      byActor: [],
      byAction: [],
    }),
  };
});

vi.mock("../../api/libraryHealth", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/libraryHealth")>();
  return {
    ...actual,
    getLibraryHealth: vi.fn().mockResolvedValue({
      generatedAt: "2026-11-19T12:00:00Z",
      counts: { assets: 1, published: 1, workspaces: 1, findingsOpen: 0, coverageRows: 0 },
      overview: { state: 1, metrics: [] },
      quality: { state: 1, metrics: [] },
      activity: { state: 1, metrics: [] },
      services: { state: 1, metrics: [] },
    }),
  };
});

vi.mock("../../api/search", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/search")>();
  return {
    ...actual,
    searchWorkbench: vi.fn().mockResolvedValue({
      query: "",
      results: [],
      facets: [],
      total: 0,
      tookMs: 1n,
      status: { indexed: true, documentCount: 1, semanticLeg: 3, stale: false, lastError: "" },
    }),
  };
});



beforeEach(async () => {
  await setLocale("en");
});

/**
 * Data hooks resolve one microtask after render. Flushing inside `act` keeps
 * the first paint settled before axe inspects it and before the suite's
 * console-error guard checks for unwrapped updates.
 */
async function settle() {
  await act(async () => {
    await new Promise((resolve) => globalThis.setTimeout(resolve, 0));
  });
}

afterEach(() => cleanup());

describe("workbench destinations accessibility", () => {
  it("Start has no structural axe violations", async () => {
    const { container } = renderWithProviders(<StartPage />);
    // The shared search debounces; wait for its settled empty state so axe
    // never inspects the surface while a request is queued.
    await screen.findByText(/No matches/i);
    await expectNoA11yViolations(container);
  });

  it("Library has no structural axe violations", async () => {
    const { container } = renderWithProviders(<LibraryPage />);
    await settle();
    await expectNoA11yViolations(container);
  });

  it("Stats has no structural axe violations", async () => {
    const { container } = renderWithProviders(<StatsPage />, { route: "/stats" });
    await settle();
    await expectNoA11yViolations(container);
  });

  it("Review has no structural axe violations", async () => {
    const { container } = renderWithProviders(<ReviewPage />, { route: "/review" });
    await settle();
    await expectNoA11yViolations(container);
  });

  it("Design index has no structural axe violations", async () => {
    const { container } = renderWithProviders(<DesignIndexPage />);
    // The shelf loads each template's real composition. Wait for at least one
    // thumbnail to settle so axe sees the rendered shelf, not placeholders.
    await waitFor(() =>
      expect(container.querySelector(".wb-template-thumb:not(.is-pending)")).toBeTruthy(),
    );
    await settle();
    await expectNoA11yViolations(container);
  });

  it("workspace studio has no structural axe violations", async () => {
    const { container } = renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    await waitFor(() =>
      expect(container.querySelector('[data-testid="workspace-studio"]')).toBeTruthy(),
    );
    await settle();
    await expectNoA11yViolations(container);
  });
});
