import { createElement } from "react";
import { Outlet, RouterProvider, createMemoryRouter } from "react-router-dom";
import { act, screen, waitFor } from "@testing-library/react";
import { expect } from "vitest";
import { renderWithProviders } from "./index";
import { useSessionExpiryBoundary } from "../features/session/useSession";
import { settingsRoute } from "../features/settings/routes";
import { WIDE_SETTINGS_QUERY } from "../features/settings/layoutMode";

/** What a phone and a desktop answer to the media queries the settings use. */
export function setViewport(kind: "phone" | "desktop"): void {
  const desktop = kind === "desktop";
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (media: string) => ({
      media,
      matches: desktop || (!media.includes("min-width") && !media.includes(WIDE_SETTINGS_QUERY)),
      onchange: null,
      addListener() {},
      removeListener() {},
      addEventListener() {},
      removeEventListener() {},
      dispatchEvent: () => false,
    }),
  });
}

/** The part of the shell the settings depend on: a refused request signs the page out. */
function ShellStandIn() {
  useSessionExpiryBoundary();
  return createElement(Outlet);
}

/**
 * Render the real settings routes in an in-memory data router, as the shell
 * would mount them, starting at `path`.
 */
export function renderSettings(path: string, viewport: "phone" | "desktop" = "phone") {
  setViewport(viewport);
  const router = createMemoryRouter([{ path: "/", element: createElement(ShellStandIn), children: [settingsRoute] }], {
    initialEntries: [path],
    future: { v7_relativeSplatPath: true },
  });
  const view = renderWithProviders(createElement(RouterProvider, { router, future: { v7_startTransition: true } }), { withoutRouter: true });
  return { ...view, router };
}

/** Wait for the page's heading, the sign that the route and its data have settled. */
export async function findPageHeading(name: string | RegExp) {
  return screen.findByRole("heading", { level: 1, name });
}

export async function waitForPath(router: { state: { location: { pathname: string } } }, pathname: string) {
  await waitFor(() => expect(router.state.location.pathname).toBe(pathname));
}

/** Navigate the way a link or the back button would, inside `act` so React settles. */
export async function go(router: { navigate: (to: string) => Promise<void> }, to: string) {
  await act(async () => { await router.navigate(to); });
}
