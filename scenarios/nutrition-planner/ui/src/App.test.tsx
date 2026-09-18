import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
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
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "./test-utils";
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

// The application shell must remain navigable with the built-in controller.
it("changes the theme when the focused setting receives gamepad Select", async () => {
  const controller = initSpatialNav({
    isVisible: () => true,
    getGamepads: () => [{ id: 'test-controller', index: 0, connected: true,
      timestamp: 0, mapping: 'standard', axes: [0, 0], vibrationActuator: { type: "dual-rumble", effects: [],
        playEffect: async () => "complete", reset: async () => "complete" },
      buttons: [{ pressed: true, touched: true, value: 1 }],
    } as Gamepad],
  });
  const view = renderWithProviders(
    <Providers><TestAppRouter initialEntries={["/settings"]} /></Providers>,
    { withoutRouter: true },
  );
  try {
    const dark = screen.getByTestId(selectors.settingsPage.themeOption({ choice: "dark" }));
    expect(dark).toHaveAttribute("aria-checked", "false");
    dark.focus();
    window.dispatchEvent(new Event("gamepadconnected"));
    await waitFor(() => expect(dark).toHaveAttribute("aria-checked", "true"));
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
  } finally {
    view.unmount();
    controller.dispose();
  }
});
