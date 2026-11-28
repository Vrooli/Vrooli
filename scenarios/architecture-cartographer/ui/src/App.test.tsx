/**
 * App tests — smoke only.
 *
 * `App` is a tiny composition of `<Providers>` + `<AppRouter>`. Per-route
 * behaviour lives in `app/routes.test.tsx`, shell wiring in
 * `layout/AppShell.test.tsx`, theme persistence in
 * `theme/ThemeProvider.test.tsx`. This file uses `TestAppRouter` directly
 * because `<App>` mounts `createBrowserRouter`, which doesn't play with the
 * memory-router wrapper inside `renderWithProviders`.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "./test-utils";
import { Providers } from "./app/providers";
import { TestAppRouter } from "./app/routes";
import { selectors } from "./consts/selectors";

// Overview hits listGraphSnapshots + health on mount; stub both so the
// composition smoke test doesn't depend on a live API.
vi.mock("./api/graph", () => ({
  graphClient: {
    listGraphSnapshots: vi.fn().mockResolvedValue({ snapshots: [], nextPageToken: "" }),
    extractGraph: vi.fn().mockResolvedValue({ snapshot: undefined, fromCache: false }),
  },
}));
vi.mock("./api/health", () => ({
  fetchHealth: vi.fn().mockResolvedValue({
    status: "ok",
    service: "architecture-cartographer-api",
    timestamp: new Date(0).toISOString(),
  }),
}));

describe("App composition", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the shell title (smoke: providers + routes wire up)", () => {
    renderWithProviders(
      <Providers>
        <TestAppRouter initialEntries={["/"]} />
      </Providers>,
      { withoutRouter: true },
    );
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
  });
});
