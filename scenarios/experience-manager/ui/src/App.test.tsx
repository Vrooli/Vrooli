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

import App from "./App";
import { renderWithProviders } from "./test-utils";
import { Providers } from "./app/providers";
import { TestAppRouter } from "./app/routes";
import { selectors } from "./consts/selectors";

vi.mock("./app/routes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./app/routes")>();
  return {
    ...actual,
    AppRouter: () => <div data-testid="app-router" />,
  };
});

describe("App composition", () => {
  afterEach(() => {
    cleanup();
  });

  it("mounts the production app composition", () => {
    renderWithProviders(<App />, { withoutRouter: true });
    expect(screen.getByTestId("app-router")).toBeInTheDocument();
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
