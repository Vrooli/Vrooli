import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";

vi.mock("../api/adoptions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/adoptions")>();
  return {
    ...actual,
    adoptionsClient: {
      listAdoptions: vi.fn().mockResolvedValue({ adoptions: [] }),
      createAdoption: vi.fn(),
      deleteAdoption: vi.fn(),
      refreshAdoptions: vi.fn(),
    },
  };
});

import { AdoptionsPage } from "./AdoptionsPage";

describe("AdoptionsPage", () => {
  afterEach(() => cleanup());

  it("renders the page header and the AdoptionsCard", () => {
    renderWithProviders(<AdoptionsPage />);
    expect(screen.getByTestId("adoptions-page")).toBeInTheDocument();
    expect(screen.getByTestId("adoptions-card")).toBeInTheDocument();
  });
});
