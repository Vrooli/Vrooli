/**
 * AppShell tests: the configuration the desk feeds the library shell (nav
 * items, phone tabs, the Needs-you badge, router adapter).
 */
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TestAppRouter } from "../app/routes";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import en from "../i18n/locales/en.json";
import { renderWithProviders, stubWire } from "../test-utils";

const renderShell = (path = "/") => renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });

const ITEM = { id: "AV4", title: "Choose how AV4 handles yt-dlp", commissionId: "programme-five-20261112", options: [], answerable: true };

describe("AppShell", () => {
  beforeEach(() => {
    stubWire((path) => (path.endsWith("/ListNeedsYou") ? { items: [ITEM, { ...ITEM, id: "AV5" }] } : {}));
  });
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("renders the shell landmarks, the brand, and the main outlet", () => {
    renderShell();
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.navigation)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.main)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.skip)).toHaveAttribute("href", `#${selectors.layout.main}`);
  });

  it("puts four destinations in the phone tabs and keeps Launch, Reviews and Desk in the menu", () => {
    renderShell();
    for (const key of ["now", "needsYou", "topics", "commissions", "launch", "reviews", "desk"] as const) {
      expect(screen.getByTestId(selectors.layout.navLink({ key }))).toBeInTheDocument();
    }
    for (const key of ["now", "needsYou", "topics", "commissions"] as const) {
      expect(screen.getByTestId(selectors.layout.navTab({ key }))).toBeInTheDocument();
    }
    for (const key of ["launch", "reviews", "desk"] as const) {
      expect(screen.queryByTestId(selectors.layout.navTab({ key }))).not.toBeInTheDocument();
    }
  });

  it("marks a drill-in as inside its destination", () => {
    renderShell("/commissions/programme-four-20261110");
    expect(screen.getByTestId(selectors.layout.navLink({ key: "commissions" }))).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId(selectors.layout.navLink({ key: "now" }))).not.toHaveAttribute("aria-current");
  });

  it("shows how many items wait for the operator on the Needs-you destination", async () => {
    await setLocale("en");
    renderShell();
    await waitFor(() => expect(screen.getByTestId(selectors.layout.navLink({ key: "needsYou" }))).toHaveTextContent("2"));
    expect(screen.getAllByText(en.layout.nav.needsYou).length).toBeGreaterThan(0);
  });

  it("navigates through the router when a phone tab is selected", async () => {
    const user = userEvent.setup();
    renderShell("/");
    await user.click(screen.getByTestId(selectors.layout.navTab({ key: "topics" })));
    await waitFor(() => expect(screen.getByTestId(selectors.pages.topics)).toBeInTheDocument());
  });
});
