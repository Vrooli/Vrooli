/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the composition the library shell
 * produces (skip link, navigation landmarks, main). Feature cards keep their
 * own a11y tests.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => ({
      ok: true,
      status: 200,
      statusText: "OK",
      json: async () => String(input).endsWith("/levels")
        ? { levels: [{ id: "table-01", family: "tables", family_name: "Tables", difficulty: 1, title: "Product directory", task: "Extract product records.", tags: [], target_url: "/practice/table-01" }] }
        : { scores: [] },
      text: async () => "",
    })));
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("renders the shell without axe violations in English", async () => {
    const { container } = renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await screen.findByTestId(selectors.range.catalog);
    await expectNoA11yViolations(container);
  });

  it("exposes one primary navigation landmark, a main region, and a skip link", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });

    await screen.findByTestId(selectors.range.catalog);
    expect(screen.getAllByRole("navigation", { name: "Primary navigation" })).toHaveLength(1);
    expect(screen.getByRole("main")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Skip to content" })).toBeInTheDocument();
    // The phone tab bar is a second navigation landmark that the shell hides
    // above the md breakpoint. jsdom has no viewport, so it is present in the
    // DOM but not in the accessibility tree here; its role is asserted by the
    // library's own story contract at phone width.
    expect(screen.getByTestId(selectors.layout.tabs)).toBeInTheDocument();
  });
});
