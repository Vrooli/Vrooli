import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { makeStudioMocks, renderWithProviders } from "../test-utils";
import { PlacementsPage } from "./PlacementsPage";

vi.mock("../api/studio", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/studio")>();
  return { ...actual, ...makeStudioMocks() };
});

describe("PlacementsPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders its own surface", async () => {
    renderWithProviders(<PlacementsPage />);
    expect(await screen.findByTestId(selectors.pages.placements)).toBeInTheDocument();
  });
});
