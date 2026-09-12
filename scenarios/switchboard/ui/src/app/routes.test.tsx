/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this
 * file's job is to assert the router config.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

describe("AppRouter", () => {
  // [REQ:SWBD-P0-017]
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response("[]", { status: 200, headers: { "Content-Type": "application/json" } }))));
  });
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("renders the overview at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it.each([
    ["/conversations", "conversations-thread-list-region"],
    ["/agents", "agents-roster-region"],
    ["/agents/new", "agent-new-draft-region"],
    ["/agents/brand-manager", "agent-detail-grant-region"],
    ["/channels", "channels-catalog-region"],
    ["/contacts", "contacts-contact-region"],
  ])("renders the declared region for %s", (path, region) => {
    renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });
    expect(screen.getByTestId(region)).toBeInTheDocument();
  });

  it("redirects the retired welcome route to agent authoring", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/welcome"]} />, { withoutRouter: true });
    expect(screen.getByTestId("agent-new-draft-region")).toBeInTheDocument();
  });
});
