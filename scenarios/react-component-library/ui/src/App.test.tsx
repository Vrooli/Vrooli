/**
 * App tests — route resolution and shell continuity for the Nocturne
 * workbench. The page-level contracts are covered by page stories; this file
 * proves the shell mounts once and each public route resolves to its page.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "./test-utils";
import { selectors } from "./consts/selectors";
import { setLocale } from "./i18n";

const emitShortcutIntentMock = vi.fn();
const useWorkspacesMock = vi.hoisted(() => vi.fn());

vi.mock("@vrooli/iframe-bridge", () => ({
  emitShortcutIntent: (...args: unknown[]) => emitShortcutIntentMock(...args),
}));

vi.mock("./workbench/workspaces/useWorkspaces", () => ({
  useWorkspaces: useWorkspacesMock,
}));

vi.mock("./workbench/pages/WorkspaceStudioPage", () => ({
  WorkspaceStudioPage: () => <div data-testid="bound-workspace-studio" />,
}));

const fixtures = vi.hoisted(() => ({
  asset: {
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
  },
}));

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

import App from "./App";

describe("workbench routes", () => {
  beforeEach(async () => {
    await setLocale("en");
    useWorkspacesMock.mockReturnValue({ data: [], isLoading: true, isError: false });
  });

  afterEach(() => {
    cleanup();
    emitShortcutIntentMock.mockClear();
  });

  it("mounts the Nocturne shell and Start at /", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await waitFor(() => {
      expect(screen.getByTestId("workbench-shell")).toBeInTheDocument();
      expect(screen.getByTestId("start-page")).toBeInTheDocument();
    });
    expect(screen.getAllByRole("link", { name: /library/i }).length).toBeGreaterThan(0);
  });

  it("opens the bookmarked component-context route when its query key is present", async () => {
    renderWithProviders(<App />, { routerEntries: ["/design?component="] });
    await screen.findByRole("alert");
  });

  it("routes scenario and page query parameters through the experience binding", async () => {
    useWorkspacesMock.mockReturnValue({ data: [], isLoading: false, isError: false });
    renderWithProviders(<App />, {
      routerEntries: ["/design?scenario=example&page=overview"],
    });
    await screen.findByTestId("adopt-experience-page");
  });

  it("opens the workspace already bound to the requested experience page", async () => {
    useWorkspacesMock.mockReturnValue({
      data: [{ id: "ws-bound", bindingScenario: "example", bindingPage: "overview" }],
      isLoading: false,
      isError: false,
    });
    renderWithProviders(<App />, {
      routerEntries: ["/design?scenario=example&page=overview"],
    });
    await screen.findByTestId("bound-workspace-studio");
  });

  it("redirects the legacy catalog route to Library", async () => {
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
    await waitFor(() => expect(screen.getByTestId("start-page")).toBeInTheDocument());
    const shell = screen.getByTestId("workbench-shell");
    const libraryLink = screen.getByTestId(selectors.workbench.nav.library);
    fireEvent.click(libraryLink);
    await waitFor(() => expect(screen.getByTestId("components-page")).toBeInTheDocument());
    expect(screen.getByTestId("workbench-shell")).toBe(shell);
  });

  it("keeps all five primary destinations available in desktop and mobile navigation", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await screen.findByTestId("start-page");

    for (const destination of Object.values(selectors.workbench.nav)) {
      expect(screen.getByTestId(destination)).toBeInTheDocument();
      expect(screen.getByTestId(`${destination}-tab`)).toBeInTheDocument();
    }
    expect(screen.getByTestId(`${selectors.app.shell}-header`)).toBeInTheDocument();
    expect(screen.getByTestId(`${selectors.app.shell}-tabs`)).toBeInTheDocument();
  });

  it("preserves active navigation for legacy route aliases", async () => {
    renderWithProviders(<App />, { routerEntries: ["/catalog"] });
    await screen.findByTestId("components-page");

    expect(screen.getByTestId(selectors.workbench.nav.library)).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByTestId(`${selectors.workbench.nav.library}-tab`)).toHaveAttribute(
      "aria-current",
      "page",
    );

    cleanup();
    renderWithProviders(<App />, { routerEntries: ["/coverage"] });
    await screen.findByTestId("stats-page");
    expect(screen.getByTestId(selectors.workbench.nav.stats)).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByTestId(`${selectors.workbench.nav.stats}-tab`)).toHaveAttribute(
      "aria-current",
      "page",
    );
  });

  it("supports a keyboard-only journey across the primary workbench destinations", async () => {
    const user = userEvent.setup();
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await screen.findByTestId("start-page");

    const visit = async (href: string, page: string) => {
      const navId = Object.entries(selectors.workbench.nav).find(([, testId]) => {
        const link = screen.queryByTestId(testId);
        return link?.getAttribute("href") === href;
      })?.[1];
      const link = navId ? screen.getByTestId(navId) : null;
      if (!link) throw new Error(`Missing workbench destination: ${href}`);
      link.focus();
      await user.keyboard("{Enter}");
      await screen.findByTestId(page);
    };

    await visit("/library", "library-page");
    await visit("/design", "design-index-page");
    await visit("/review", "review-page");
    await visit("/stats", "stats-page");
  });

  it("opens the command palette from the keyboard and traps focus until Escape", async () => {
    const user = userEvent.setup();
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await screen.findByTestId("start-page");

    fireEvent.keyDown(window, { key: "k", ctrlKey: true });
    const dialog = await screen.findByRole("dialog", { name: "Workbench search" });
    const search = within(dialog).getByRole("searchbox", { name: "Search the library" });
    await waitFor(() => expect(search).toHaveFocus());
    await user.keyboard("{Shift>}{Tab}{/Shift}");
    expect(dialog).toContainElement(document.activeElement as HTMLElement);
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Workbench search" })).not.toBeInTheDocument();
  });

  it("routes the legacy measurement URLs to their Stats views", async () => {
    renderWithProviders(<App />, { routerEntries: ["/coverage"] });
    await waitFor(() => expect(screen.getByTestId("stats-page")).toBeInTheDocument());
    expect(screen.getByRole("tab", { name: /Quality/i })).toHaveAttribute("aria-selected", "true");
    cleanup();

    renderWithProviders(<App />, { routerEntries: ["/capabilities"] });
    await waitFor(() => expect(screen.getByTestId("stats-page")).toBeInTheDocument());
    expect(screen.getByRole("tab", { name: /Services/i })).toHaveAttribute("aria-selected", "true");
  });

  it("renders an unknown route as the workbench not-found page", async () => {
    renderWithProviders(<App />, { routerEntries: ["/does-not-exist"] });
    await waitFor(() => {
      expect(screen.getByTestId("not-found-page")).toBeInTheDocument();
    });
  });
});
