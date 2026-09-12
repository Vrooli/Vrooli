import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { makeStudioMocks, renderWithProviders } from "../test-utils";
import { CandidatesPage } from "./CandidatesPage";

vi.mock("../api/studio", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/studio")>();
  return { ...actual, ...makeStudioMocks() };
});

describe("CandidatesPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders its own surface", async () => {
    renderWithProviders(<CandidatesPage />);
    expect(await screen.findByTestId(selectors.pages.candidates)).toBeInTheDocument();
  });
});
