import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import type { CatalogAsset } from "../../api/components";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings.generated";

const { listCatalogAssets, startWorkflow } = vi.hoisted(() => ({
  listCatalogAssets: vi.fn(),
  startWorkflow: vi.fn(),
}));

vi.mock("../../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/components")>();
  return { ...actual, listCatalogAssets };
});

vi.mock("../../api/workflows", () => ({
  workflowsClient: { startWorkflow },
}));

import { CatalogBrowser } from "./CatalogBrowser";

const component: CatalogAsset = {
  id: "focus-panel",
  libraryId: "react-component-library:FocusTrapPanel",
  displayName: "FocusTrapPanel",
  slot: "layout",
  assetKind: 1,
  metrics: { directAdoptionCount: 3, versionCount: 2 },
} as CatalogAsset;

const hook: CatalogAsset = {
  id: "focus-hook",
  libraryId: "react-component-library:useFocusTrap",
  displayName: "useFocusTrap",
  category: "accessibility",
  assetKind: 2,
  metrics: { directAdoptionCount: 1, versionCount: 1 },
} as CatalogAsset;

describe("CatalogBrowser", () => {
  beforeEach(() => {
    listCatalogAssets.mockImplementation(({ assetKind }: { assetKind: number }) =>
      Promise.resolve({ components: assetKind === 2 ? [hook] : [component] }),
    );
    startWorkflow.mockResolvedValue({ workflow: { id: "workflow-1" } });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows direct adoption counts for components and effective counts only for hooks", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CatalogBrowser />);

    expect(await screen.findByTestId(selectors.catalog.asset)).toHaveTextContent(
      component.displayName,
    );
    expect(screen.getByText(strings.catalog.adoptions)).toBeInTheDocument();
    expect(screen.queryByText(/effective/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: strings.catalog.hooks }));

    await waitFor(() =>
      expect(screen.getByTestId(selectors.catalog.asset)).toHaveTextContent(hook.displayName),
    );
    expect(screen.getByText(strings.catalog.adoptions)).toBeInTheDocument();
    expect(screen.getByText(/effective/i)).toBeInTheDocument();
    await waitFor(() =>
      expect(listCatalogAssets).toHaveBeenLastCalledWith(expect.objectContaining({ assetKind: 2 })),
    );
  });

  it("reports the primary catalog lifecycle without instrumenting the sidebar variant", async () => {
    let resolveCatalog!: (value: { components: CatalogAsset[] }) => void;
    listCatalogAssets.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveCatalog = resolve;
      }),
    );
    renderWithProviders(<CatalogBrowser surfaceId="catalog-results" />);

    const surface = document.querySelector('[data-experience-surface="catalog-results"]');
    expect(surface).toHaveAttribute("data-experience-state", "loading");
    resolveCatalog({ components: [component] });
    await waitFor(() => expect(surface).toHaveAttribute("data-experience-state", "ready"));
  });

  it("aggregates tree adoption counts and changes presentation without changing assets", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CatalogBrowser />);

    await screen.findByTestId(selectors.catalog.asset);
    expect(await screen.findByText("3")).toBeInTheDocument();
    const controls = screen.getAllByTestId(selectors.catalog.presentation);
    await user.click(controls[2]!);

    expect(controls[2]).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId(selectors.catalog.asset).className).toContain("min-h-24");
  });

  it("starts assisted extraction only through the RCL workflow service", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CatalogBrowser />);

    await user.click(screen.getByRole("button", { name: strings.catalog.addAssisted }));
    await user.type(screen.getByLabelText(strings.catalog.sourceScenario), "demo-scenario");
    await user.type(screen.getByLabelText(strings.catalog.sourcePath), "ui/src/Panel.tsx");
    await user.click(screen.getByRole("button", { name: strings.catalog.assistedStart }));

    await waitFor(() =>
      expect(startWorkflow).toHaveBeenCalledWith({
        kind: 1,
        sourceScenario: "demo-scenario",
        sourcePath: "ui/src/Panel.tsx",
        idempotencyKey: "catalog-extract:demo-scenario:ui/src/Panel.tsx",
      }),
    );
  });
});
