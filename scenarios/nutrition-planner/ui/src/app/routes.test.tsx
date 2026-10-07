/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this
 * file's job is to assert the router config. Add one case per route you add.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders Today as the normal root entry", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.today)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.navLink({ key: "today" }))).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId(selectors.layout.navTab({ key: "today" }))).toHaveAttribute("aria-current", "page");
    expect(screen.queryByText("Quick capture")).not.toBeInTheDocument();
  });

  it("redirects the legacy Today URL to the normal root route", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/today"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.today)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.navLink({ key: "today" }))).toHaveAttribute("aria-current", "page");
  });

  it("routes the selected Meals destination", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/meals"]} />, { withoutRouter: true });
    expect(await screen.findByRole("heading", { name: "Your meals" })).toBeInTheDocument();
  });

  it("routes Explore under Meals", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/meals/explore"]} />, { withoutRouter: true });
    expect(await screen.findByRole("heading", { name: "Meals worth making" })).toBeInTheDocument();
  });

  it("routes the selected Week destination", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/week"]} />, { withoutRouter: true });
    expect(await screen.findByRole("heading", { name: "Your week" })).toBeInTheDocument();
  });

  it("routes the selected Groceries destination", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/groceries"]} />, { withoutRouter: true });
    expect(await screen.findByRole("heading", { name: "Your groceries" })).toBeInTheDocument();
  });

  it("routes the selected Kitchen destination", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/kitchen"]} />, { withoutRouter: true });
    expect(await screen.findByRole("heading", { name: "Your kitchen" })).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
