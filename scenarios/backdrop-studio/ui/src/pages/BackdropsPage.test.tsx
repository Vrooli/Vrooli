import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { makeStudioMocks, renderWithProviders } from "../test-utils";
import { BackdropsPage } from "./BackdropsPage";

vi.mock("../api/studio", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/studio")>();
  return { ...actual, ...makeStudioMocks() };
});

describe("BackdropsPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders its own surface", async () => {
    renderWithProviders(<BackdropsPage />);
    expect(await screen.findByTestId(selectors.pages.backdrops)).toBeInTheDocument();
  });
});
