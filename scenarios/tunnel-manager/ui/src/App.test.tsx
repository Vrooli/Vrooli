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
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "./test-utils";
import { Providers } from "./app/providers";
import { TestAppRouter } from "./app/routes";
import { createAppQueryClient } from "./app/queryClient";
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

  it("uses immediate, visible capability failures in production", () => {
    const defaults = createAppQueryClient().getDefaultOptions();

    expect(defaults.queries?.retry).toBe(false);
    expect(defaults.queries?.refetchOnWindowFocus).toBe(false);
    expect(defaults.mutations?.retry).toBe(false);
  });
});
