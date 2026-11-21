import { afterEach, describe, expect, it, vi } from "vitest";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { screen } from "@testing-library/react";

import { selectors } from "../consts/selectors";
import { en } from "../i18n/catalog";
import { setLocale } from "../i18n";
import { renderWithProviders } from "../test-utils";
import { RouteError } from "./RouteError";

function Broken(): never {
  throw new Error("route failed to render");
}

let consoleError: ReturnType<typeof vi.spyOn> | undefined;
// React and React Router report the throw they are about to hand to the error element; the
// shared setup treats console output as a failure, so this file silences it and restores it,
// as the ErrorBoundary tests do.
afterEach(() => consoleError?.mockRestore());

describe("RouteError", () => {
  it("replaces the library's raw error page with the app's own, when a route throws", async () => {
    await setLocale("en");
    consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined);
    const router = createMemoryRouter([{ path: "/", element: <Broken />, errorElement: <RouteError /> }], {
      future: { v7_relativeSplatPath: true },
    });
    renderWithProviders(<RouterProvider router={router} future={{ v7_startTransition: true }} />, { withoutRouter: true });
    const alert = await screen.findByTestId(selectors.errorBoundary.root);
    expect(alert).toHaveTextContent(en.errorBoundary.title);
    expect(screen.getByTestId(selectors.errorBoundary.retryButton)).toHaveTextContent(en.errorBoundary.retry);
  });
});
