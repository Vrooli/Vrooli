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
import { makeScanFleetResponse } from "./features/storage/mocks/factories";

// The index route is the data-backed dashboard; stub the client so this smoke
// never reaches a live transport.
vi.mock("./api/storage", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/storage")>();
  return {
    ...actual,
    storageClient: {
      ...actual.storageClient,
      getInventory: vi.fn().mockResolvedValue(makeScanFleetResponse({ scenarioCount: 0 })),
      scanFleet: vi.fn().mockResolvedValue(makeScanFleetResponse({ scenarioCount: 0 })),
    },
  };
});

import { Providers } from "./app/providers";
import { TestAppRouter } from "./app/routes";
import { selectors } from "./consts/selectors";

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
