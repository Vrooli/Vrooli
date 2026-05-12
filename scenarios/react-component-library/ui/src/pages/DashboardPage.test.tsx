import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders, makeHealthResponse } from "../test-utils";

vi.mock("../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/components")>();
  return {
    ...actual,
    componentsClient: {
      listComponents: vi.fn(),
      getComponent: vi.fn(),
      getComponentByLibraryId: vi.fn(),
      indexComponents: vi.fn(),
      getComponentContent: vi.fn(),
      updateComponentContent: vi.fn(),
    },
  };
});

vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: vi.fn() };
});

import { DashboardPage } from "./DashboardPage";

describe("DashboardPage", () => {
  beforeEach(async () => {
    const { fetchHealth } = await import("../api/health");
    const { componentsClient } = await import("../api/components");
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
    vi.mocked(componentsClient.listComponents).mockResolvedValue({
      components: new Array(7).fill(0).map((_, i) => ({
        id: `cmp-${i}`,
        libraryId: `react-component-library:C${i}`,
        displayName: `C${i}`,
        description: "",
        version: "0.1.0",
        sourcePath: `C${i}.tsx`,
        tags: [],
        updatedAt: "",
      })),
    } as unknown as Awaited<ReturnType<typeof componentsClient.listComponents>>);
  });
  afterEach(() => cleanup());

  it("renders the indexed-components count from listComponents.components.length", async () => {
    renderWithProviders(<DashboardPage />);
    expect(screen.getByTestId("dashboard-page")).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId("dashboard-components-tile").textContent).toContain("7");
    });
  });

  it("links to /components from the components tile", async () => {
    renderWithProviders(<DashboardPage />);
    await waitFor(() => {
      expect(screen.getByTestId("dashboard-components-link")).toHaveAttribute(
        "href",
        "/components",
      );
    });
  });
});
