/**
 * App per-route ErrorBoundary test.
 *
 * Mocks the inventory page to throw at render. Asserts the shell + sidebar
 * remain mounted while the error boundary contains the crash to the route.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "./test-utils";
import { selectors } from "./consts/selectors";

vi.mock("./pages/InventoryPage", () => ({
  InventoryPage() {
    throw new Error("simulated render-time blowup");
  },
}));

import App from "./App";

describe("App per-route ErrorBoundary", () => {
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("contains a render-time crash in the route without dismounting the shell", async () => {
    renderWithProviders(<App />, { routerEntries: ["/flows"] });
    expect(await screen.findByTestId(selectors.errorBoundary.root)).toBeInTheDocument();
    expect(screen.getByTestId("app-shell")).toBeInTheDocument();
  });
});
