/**
 * App composition smoke test. Verifies that the shell + dashboard route mount
 * without crashing through the in-memory router. Per-page behaviour lives in
 * the feature/page-specific tests.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders, makeHealthResponse } from "./test-utils";
import { selectors } from "./consts/selectors";

vi.mock("./api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/health")>();
  return {
    ...actual,
    fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()),
  };
});

import App from "./App";

describe("App composition", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the shell and the dashboard at /", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
    });
  });
});
