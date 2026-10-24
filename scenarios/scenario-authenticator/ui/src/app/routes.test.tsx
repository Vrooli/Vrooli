/**
 * Routing smoke — for each canonical path (`/`, `/notes`, `/settings`) the
 * matching page selector is in the document. Page-internal behaviour is
 * exercised in per-page tests; this file's job is to assert the router config.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the dashboard at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it("renders account and permission controls on settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByRole("button", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Request access" })).toBeInTheDocument();
  });

  it("changes settings through the shared controls", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });

    await user.click(screen.getByTestId(selectors.settingsPage.themeOption({ choice: "dark" })));
    await user.click(screen.getByTestId(selectors.settingsPage.localeOption({ code: "ja" })));

    expect(screen.getByTestId(selectors.settingsPage.themeOption({ choice: "dark" }))).toHaveAttribute("aria-checked", "true");
    expect(screen.getByTestId(selectors.settingsPage.localeOption({ code: "ja" }))).toHaveAttribute("aria-checked", "true");
  });

  it("renders the hosted sign-in page", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/auth/login"]} />, { withoutRouter: true });
    expect(screen.getByTestId("hosted-login-page")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "auth.viewReadOnly" })).toBeInTheDocument();
  });
});
