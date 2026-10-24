/**
 * App tests — smoke only.
 *
 * Exercise the production composition itself. The shared helper supplies the
 * QueryClient/i18n installed by main.tsx, but no scenario ThemeProvider or router:
 * App must supply those. The local wrapper would mask a missing App provider.
 * Detailed route and theme behavior remains in the corresponding domain tests.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders } from "@vrooli/api-base/testing";

import App from "./App";
import { i18n } from "./i18n";
import { selectors } from "./consts/selectors";

describe("App composition", () => {
  const originalUrl = window.location.href;
  afterEach(() => {
    cleanup();
    window.history.replaceState(null, "", originalUrl);
  });

  it("renders the shell title (smoke: providers + routes wire up)", () => {
    window.history.replaceState(null, "", "/");
    renderWithProviders(<App />, { withoutRouter: true, i18n });
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
  });
});
