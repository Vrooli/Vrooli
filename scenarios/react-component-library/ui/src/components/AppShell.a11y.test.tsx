import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

import { expectNoA11yViolations, renderWithProviders, makeHealthResponse } from "../test-utils";

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
vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: vi.fn() };
});

import { AppShell } from "./AppShell";

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<section aria-label="content">page</section>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );
    expect(screen.getByRole("main")).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });
});
