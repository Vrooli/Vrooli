import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
/**
 * App tests — smoke only.
 *
 * `App` is a tiny composition of `<Providers>` + `<AppRouter>`. Per-route
 * behaviour lives in `app/routes.test.tsx`, shell wiring in
 * `layout/AppShell.test.tsx`, theme persistence in
 * `theme/ThemeProvider.test.tsx`. The smoke test mounts the real `<App>`
 * (browser router at jsdom's "/"), so it renders without the memory-router
 * wrapper; the gamepad test needs a start URL and uses `TestAppRouter`.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import App from "./App";
import { renderWithProviders, stubWire } from "./test-utils";
import { Providers } from "./app/providers";
import { TestAppRouter } from "./app/routes";
import { selectors } from "./consts/selectors";

describe("App composition", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the shell title (smoke: providers + browser routes wire up)", () => {
    stubWire();
    renderWithProviders(<App />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
  });
});

// The application shell must remain navigable with the built-in controller.
it("changes the theme when the focused setting receives gamepad Select", async () => {
  const controller = initSpatialNav({
    isVisible: () => true,
    getGamepads: () => [{ id: 'test-controller', index: 0, connected: true,
      timestamp: 0, mapping: 'standard', axes: [0, 0], vibrationActuator: { type: "dual-rumble", effects: [],
        playEffect: () => Promise.resolve("complete" as const), reset: () => Promise.resolve("complete" as const) },
      buttons: [{ pressed: true, touched: true, value: 1 }],
    } as Gamepad],
  });
  const view = renderWithProviders(
    <Providers><TestAppRouter initialEntries={["/desk"]} /></Providers>,
    { withoutRouter: true },
  );
  try {
    // The theme control is a library Select: gamepad Select opens it on the
    // focused trigger, then chooses the focused option.
    const trigger = screen.getByTestId(selectors.desk.themeSelect);
    trigger.focus();
    window.dispatchEvent(new Event("gamepadconnected"));
    const dark = await screen.findByRole("option", { name: /dark/i });
    dark.focus();
    window.dispatchEvent(new Event("gamepadconnected"));
    await waitFor(() => expect(document.documentElement).toHaveAttribute("data-theme", "dark"));
  } finally {
    view.unmount();
    controller.dispose();
  }
});
