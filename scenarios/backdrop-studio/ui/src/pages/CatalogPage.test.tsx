import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { QualityTier } from "@vrooli/proto-types/backdrop-studio/v1/shared/shared_pb";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { makeCandidate, makeStudioMocks, makeStyle, renderWithProviders } from "../test-utils";
import { CatalogPage } from "./CatalogPage";

/**
 * The catalog's three kinds of style: one the procedural lane draws in-process,
 * one a local diffusion model draws (minutes of CPU on a host with no GPU), and
 * one a frontier model draws (billed per render).
 */
const procedural = makeStyle();
const localModel = makeStyle({
  id: "guided-colonnade",
  name: "Guided Colonnade",
  strategy: "guided",
  qualityTier: QualityTier.LOCAL_MODEL,
});
const frontierModel = makeStyle({
  id: "synthesized-atrium",
  name: "Synthesized Atrium",
  strategy: "synthesized",
  qualityTier: QualityTier.FRONTIER_MODEL,
});

async function submittedStyleIds(): Promise<string[]> {
  const { submitRender } = await import("../api/studio");
  return vi.mocked(submitRender).mock.calls.map(([req]) => req.styleId);
}

vi.mock("../api/studio", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/studio")>();
  return { ...actual, ...makeStudioMocks() };
});

describe("CatalogPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a specimen per style once the catalog loads", async () => {
    renderWithProviders(<CatalogPage />);
    expect(await screen.findByTestId("catalog-grid")).toBeInTheDocument();
    expect(screen.getByTestId("style-specimen-cyanotype-arcade")).toBeInTheDocument();
    // The tile shows the style's quality tier (procedural, local model, or
    // frontier model): what an operator needs before rendering or spending.
    expect(screen.getByText(strings.pages.catalog.tierProcedural)).toBeInTheDocument();
  });

  it("announces the loading state before the catalog arrives", () => {
    renderWithProviders(<CatalogPage />);
    expect(screen.getByTestId(selectors.pages.catalog)).toHaveAttribute(
      "data-experience-state",
      "loading",
    );
  });

  it("surfaces an error with a retry rather than an empty grid", async () => {
    const { listStyles } = await import("../api/studio");
    vi.mocked(listStyles).mockRejectedValue(new Error("catalog unreachable"));
    renderWithProviders(<CatalogPage />);
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });

  it("offers to clear filters when a facet combination matches nothing", async () => {
    const { listStyles } = await import("../api/studio");
    vi.mocked(listStyles).mockImplementation((filter) =>
      Promise.resolve(filter?.lineage ? [] : [makeStyle()]),
    );
    renderWithProviders(<CatalogPage />);
    await screen.findByTestId("catalog-grid");

    fireEvent.change(screen.getByTestId("catalog-filter-lineage"), { target: { value: "cyanotype" } });
    await waitFor(() =>
      expect(screen.getByText(strings.pages.catalog.filteredEmptyTitle)).toBeInTheDocument(),
    );
    // Two: one in the facet row, one in the empty state. Both are correct —
    // the operator may be looking at either when they give up on the filter.
    expect(screen.getAllByRole("button", { name: /clearFilters/i })).toHaveLength(2);
  });

  it("filters by an axis and asks the service for the narrowed set", async () => {
    const { listStyles } = await import("../api/studio");
    renderWithProviders(<CatalogPage />);
    await screen.findByTestId("catalog-grid");

    fireEvent.change(screen.getByTestId("catalog-filter-subject"), {
      target: { value: "statuary_architecture" },
    });
    await waitFor(() =>
      expect(vi.mocked(listStyles)).toHaveBeenCalledWith(
        expect.objectContaining({ subject: "statuary_architecture" }),
      ),
    );
  });

  describe("model-backed specimens", () => {
    async function mountMixedCatalog(options: Parameters<typeof renderWithProviders>[1] = {}) {
      const { listStyles } = await import("../api/studio");
      vi.mocked(listStyles).mockResolvedValue([procedural, localModel, frontierModel]);
      const result = renderWithProviders(<CatalogPage />, options);
      await screen.findByTestId("catalog-grid");
      // Let every mount-time query settle before asserting what was submitted.
      await waitFor(async () => expect(await submittedStyleIds()).toContain(procedural.id));
      return result;
    }

    it("opening the catalog renders procedural specimens but never submits a model-backed render", async () => {
      await mountMixedCatalog();

      // A model render is minutes of CPU or a metered charge. Loading the page
      // — by an operator, a screenshot sweep, or a health capture — must not
      // spend either.
      expect(await submittedStyleIds()).toEqual([procedural.id]);
      for (const style of [localModel, frontierModel]) {
        const tile = screen.getByTestId(`style-specimen-${style.id}`);
        expect(within(tile).getByText(strings.pages.catalog.specimenOnRequest)).toBeInTheDocument();
        expect(
          within(tile).getByRole("button", { name: new RegExp(strings.pages.catalog.renderSpecimen) }),
        ).toBeInTheDocument();
      }
    });

    it("renders a model-backed specimen only when the operator asks for that one style", async () => {
      await mountMixedCatalog();

      const tile = screen.getByTestId(`style-specimen-${localModel.id}`);
      fireEvent.click(
        within(tile).getByRole("button", { name: new RegExp(strings.pages.catalog.renderSpecimen) }),
      );

      await waitFor(async () =>
        expect(await submittedStyleIds()).toEqual([procedural.id, localModel.id]),
      );
      expect(await submittedStyleIds()).not.toContain(frontierModel.id);
      expect(await within(tile).findByRole("img")).toBeInTheDocument();
    });

    it("shows a model-backed specimen already rendered this session without submitting again", async () => {
      // Imported here rather than at the top: the hook module imports the
      // mocked `api/studio`, and a top-level import would run the hoisted mock
      // factory before `test-utils` is initialised.
      const { studioKeys } = await import("../hooks/useStudio");
      // A session client that retains renders, as `useRender({ retain })`
      // does for a model-backed result the operator already paid for.
      const queryClient = new QueryClient({
        defaultOptions: { queries: { retry: false, gcTime: Infinity } },
      });
      queryClient.setQueryData(
        studioKeys.render({
          styleId: localModel.id,
          surfaceId: "web.hero",
          placement: localModel.placements[0],
          seed: 7n,
        }),
        {
          id: "job-earlier",
          styleId: localModel.id,
          status: "completed",
          seed: 7n,
          candidates: [makeCandidate({ jobId: "job-earlier", strategy: "guided" })],
        },
      );

      await mountMixedCatalog({ queryClient });

      const tile = screen.getByTestId(`style-specimen-${localModel.id}`);
      expect(within(tile).getByRole("img")).toBeInTheDocument();
      expect(await submittedStyleIds()).toEqual([procedural.id]);
    });
  });
});
