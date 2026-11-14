import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { makeStudioMocks, renderWithProviders } from "../test-utils";
import { SweepPage } from "./SweepPage";

vi.mock("../api/studio", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/studio")>();
  return { ...actual, ...makeStudioMocks() };
});

describe("SweepPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("asks for a style before rendering anything", async () => {
    renderWithProviders(<SweepPage />, { routerEntries: ["/sweep"] });
    expect(await screen.findByText(strings.pages.sweep.empty)).toBeInTheDocument();
  });

  it("renders one cell per seed in the range", async () => {
    renderWithProviders(<SweepPage />, { routerEntries: ["/sweep?style=cyanotype-arcade"] });
    expect(await screen.findByTestId("sweep-grid")).toBeInTheDocument();
    // Six seeds by default, starting at 7 — the same seed the catalog
    // specimens use, so the first cell is the tile the operator clicked from.
    expect(screen.getByTestId("sweep-cell-7")).toBeInTheDocument();
    expect(screen.getByTestId("sweep-cell-12")).toBeInTheDocument();
  });

  it("lets the operator select a candidate from the grid", async () => {
    renderWithProviders(<SweepPage />, { routerEntries: ["/sweep?style=cyanotype-arcade"] });
    await screen.findByTestId("sweep-grid");
    const cell = screen.getByTestId("sweep-cell-9");
    fireEvent.click(cell.querySelector("button") as HTMLButtonElement);
    expect(screen.getByTestId("sweep-cell-9")).toHaveAttribute("data-selected", "true");
  });

  it("announces loading while the catalog is in flight", () => {
    renderWithProviders(<SweepPage />, { routerEntries: ["/sweep"] });
    expect(screen.getByTestId(selectors.pages.sweep)).toHaveAttribute(
      "data-experience-state",
      "loading",
    );
  });

  it("marks a cell whose render failed rather than showing an empty frame", async () => {
    const { submitRender } = await import("../api/studio");
    vi.mocked(submitRender).mockRejectedValue(new Error("out of device memory"));
    renderWithProviders(<SweepPage />, { routerEntries: ["/sweep?style=cyanotype-arcade"] });
    expect(await screen.findAllByText(strings.pages.catalog.specimenUnavailable)).not.toHaveLength(0);
  });
});
