import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { CatalogPage } from "./CatalogPage";

vi.mock("../../api/tokenEconomy", () => ({
  nextIdempotencyKey: vi.fn(() => "test-key"),
  minterClient: {
    listCatalogEntries: vi.fn().mockResolvedValue({ entries: [
      { id: "movie", title: "Movie night", costAmount: 5n, approvalPosture: 1, retired: false },
      { id: "story", title: "Extra story", costAmount: 3n, approvalPosture: 2, retired: true },
    ] }),
    createCatalogEntry: vi.fn().mockResolvedValue({}),
    retireCatalogEntry: vi.fn().mockResolvedValue({}),
  },
}));

describe("CatalogPage", () => {
  afterEach(() => cleanup());
  it("renders catalog authoring and approval posture accessibly", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<CatalogPage />);
    await waitFor(() => expect(screen.getByTestId("catalog-table")).toBeInTheDocument());
    expect(screen.getByTestId("catalog-approval")).toBeInTheDocument();
    await user.type(screen.getByTestId("catalog-token-type"), "chores");
    await user.type(screen.getByTestId("catalog-title"), "Extra bedtime story");
    await user.type(screen.getByTestId("catalog-description"), "Choose tonight's story");
    await user.clear(screen.getByTestId("catalog-cost"));
    await user.type(screen.getByTestId("catalog-cost"), "3");
    await user.selectOptions(screen.getByTestId("catalog-approval"), "2");
    await user.click(screen.getByTestId("catalog-create"));
    await user.click(screen.getByTestId("catalog-retire-movie"));
    const { minterClient } = await import("../../api/tokenEconomy");
    await waitFor(() => expect(minterClient.createCatalogEntry).toHaveBeenCalledOnce());
    expect(minterClient.retireCatalogEntry).toHaveBeenCalledOnce();
    await expectNoA11yViolations(container);
  });
});
