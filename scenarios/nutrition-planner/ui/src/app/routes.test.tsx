/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this
 * file's job is to assert the router config. Add one case per route you add.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, fireEvent } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { AppRouter, TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    window.history.replaceState({}, "", "/");
  });

  it("opens and navigates Nooch under the injected Portal gateway path", async () => {
    const base = "/apps/nutrition-planner/proxy";
    vi.stubGlobal("__VROOLI_PROXY_INFO__", {
      generatedAt: Date.now(), hosts: ["localhost"],
      primary: { path: base }, ports: [],
    });
    window.history.replaceState({}, "", base + "/");
    renderWithProviders(<AppRouter />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.today)).toBeInTheDocument();
    const meals = screen.getByTestId(selectors.layout.navLink({ key: "meals" }));
    expect(meals).toHaveAttribute("href", base + "/meals");
    fireEvent.click(meals);
    expect(await screen.findByRole("heading", { name: "Your meals" })).toBeInTheDocument();
    expect(window.location.pathname).toBe(base + "/meals");
  });

  it("keeps the production direct route working without proxy metadata", async () => {
    window.history.replaceState({}, "", "/");
    renderWithProviders(<AppRouter />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.today)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.navLink({ key: "meals" }))).toHaveAttribute("href", "/meals");
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
