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

// `BaseStyles` injects the library's canonical CSS (including `@layer` and
// `color-mix()`), which jsdom's CSS parser rejects. Its only job is to publish
// design tokens to the real browser, so stub it here; the layout contract is
// asserted in AppShell.test.tsx.
vi.mock("@vrooli/react-component-library/BaseStyles", () => ({ BaseStyles: () => null }));
import App from "./App";
import { Providers } from "./app/providers";
import { TestAppRouter } from "./app/routes";
import { selectors } from "./consts/selectors";

describe("App composition", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the shell title (smoke: providers + routes wire up)", async () => {
    renderWithProviders(
      <Providers>
        <TestAppRouter initialEntries={["/"]} />
      </Providers>,
      { withoutRouter: true },
    );
    expect(await screen.findByTestId(selectors.app.title)).toBeInTheDocument();
  });

  it("mounts the production browser-router composition", async () => {
    renderWithProviders(<App />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.app.title)).toBeInTheDocument();
  });
});
