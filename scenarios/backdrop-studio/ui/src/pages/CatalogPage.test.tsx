import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { makeStudioMocks, makeStyle, renderWithProviders } from "../test-utils";
import { CatalogPage } from "./CatalogPage";

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
    // The tile shows the style's cost lane, which is what an operator needs
    // before spending anything.
    expect(screen.getByText(strings.pages.catalog.free)).toBeInTheDocument();
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
});
