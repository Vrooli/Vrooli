import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";

vi.mock("../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/components")>();
  return {
    ...actual,
    componentsClient: {
      listComponents: vi.fn().mockResolvedValue({ components: [] }),
      getComponent: vi.fn(),
      getComponentByLibraryId: vi.fn(),
      indexComponents: vi.fn(),
      getComponentContent: vi.fn(),
      updateComponentContent: vi.fn(),
    },
  };
});

import { ComponentsPage } from "./ComponentsPage";

describe("ComponentsPage", () => {
  afterEach(() => cleanup());

  it("renders the page header and the ComponentsCard list", () => {
    renderWithProviders(<ComponentsPage />);
    expect(screen.getByTestId("components-page")).toBeInTheDocument();
    expect(screen.getByTestId("components-card")).toBeInTheDocument();
  });
});
