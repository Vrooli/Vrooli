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
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "./test-utils";
import { Providers } from "./app/providers";
import { TestAppRouter } from "./app/routes";
import { selectors } from "./consts/selectors";
import { setLocale } from "./i18n";

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

it("changes the theme from the settings control", async () => {
  const user = userEvent.setup();
  await setLocale("en");
  renderWithProviders(
    <Providers><TestAppRouter initialEntries={["/settings"]} /></Providers>,
    { withoutRouter: true },
  );
  await user.click(screen.getByTestId(selectors.settingsPage.themeSelect));
  await user.click(screen.getByRole("option", { name: "Dark" }));
  expect(document.documentElement).toHaveAttribute("data-theme", "dark");
});
