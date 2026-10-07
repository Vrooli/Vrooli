/**
 * AppShell tests verify the warm-kitchen navigation, appearance control, and
 * landmarks. Page content is exercised in the per-page tests.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import en from "../i18n/locales/en.json";
import ja from "../i18n/locales/ja.json";
import ar from "../i18n/locales/ar.json";
import { TestAppRouter } from "../app/routes";

vi.mock("../api/recipes", () => ({ listRecipes: () => new Promise(() => undefined), createRecipe: vi.fn() }));
vi.mock("../api/workspace", () => ({ ensureWorkspace: () => new Promise(() => undefined) }));

const renderShell = (path = "/") =>
  renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });

describe("AppShell structure (cimode)", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the shell landmarks, the brand, and the main outlet", () => {
    renderShell();
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.navigation)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.tabs)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.main)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.brand)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.layout.skip)).toHaveAttribute("href", `#${selectors.layout.main}`);
  });

  it("keeps preferences out of the shell chrome", () => {
    renderShell();
    expect(screen.queryByTestId(selectors.settingsPage.localeSelect)).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.settingsPage.themeSelect)).not.toBeInTheDocument();
  });

  it("renders primary destinations as header links, settings as a utility link, and five phone tabs", () => {
    renderShell("/settings");
    for (const key of [
      "today",
      "week",
      "groceries",
      "meals",
      "kitchen",
      "settings",
    ] as const) {
      expect(screen.getByTestId(selectors.layout.navLink({ key }))).toBeInTheDocument();
    }
    for (const key of ["today", "week", "groceries", "meals", "kitchen"] as const) expect(screen.getByTestId(selectors.layout.navTab({ key }))).toBeInTheDocument();
    expect(within(screen.getByTestId(selectors.layout.navigation)).getAllByRole("link").map((link) => link.getAttribute("href"))).toEqual(["/", "/week", "/meals", "/groceries", "/kitchen"]);
    expect(within(screen.getByTestId(selectors.layout.tabs)).getAllByRole("link").map((link) => link.getAttribute("href"))).toEqual(["/", "/week", "/meals", "/groceries", "/kitchen"]);
  });

  it("marks the current route on settings and keeps Today unselected", () => {
    renderShell("/settings");
    expect(screen.getByTestId(selectors.layout.navLink({ key: "settings" }))).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId(selectors.layout.navLink({ key: "today" }))).not.toHaveAttribute("aria-current");
  });

  it("marks each selected destination active in both desktop and phone navigation", () => {
    const cases = [
      ["/week", "week"],
      ["/meals", "meals"],
      ["/groceries", "groceries"],
      ["/kitchen", "kitchen"],
    ] as const;
    for (const [path, key] of cases) {
      cleanup();
      renderShell(path);
      expect(screen.getByTestId(selectors.layout.navLink({ key }))).toHaveAttribute("aria-current", "page");
      expect(screen.getByTestId(selectors.layout.navTab({ key }))).toHaveAttribute("aria-current", "page");
    }
  });

  it("opens Settings from the header utility", async () => {
    const user = userEvent.setup();
    renderShell("/");
    await user.click(screen.getByTestId(selectors.layout.navLink({ key: "settings" })));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
    });
  });
});

describe("Locale switching through the shell (real locales)", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders English copy by default and reflects it on <html>", async () => {
    renderShell();
    // The desktop link and the phone tab both render the label, so there will be ≥1 match.
    expect((await screen.findAllByText(en.layout.nav.today)).length).toBeGreaterThan(0);
    expect(document.documentElement.lang).toBe("en");
    expect(document.documentElement.dir).toBe("ltr");
  });

  it("switches to Japanese when 日本語 is selected", async () => {
    const user = userEvent.setup();
    renderShell("/settings");
    await user.click(screen.getByTestId(selectors.settingsPage.localeSelect));
    await user.click(screen.getByRole("option", { name: "日本語" }));

    await waitFor(() => {
      expect(screen.getAllByText(ja.layout.nav.today).length).toBeGreaterThan(0);
    });
    expect(document.documentElement.lang).toBe("ja");
  });

  it("flips <html dir> to rtl when an RTL locale (ar) is chosen", async () => {
    const user = userEvent.setup();
    renderShell("/settings");
    await user.click(screen.getByTestId(selectors.settingsPage.localeSelect));
    await user.click(screen.getByRole("option", { name: "العربية" }));

    await waitFor(() => {
      expect(document.documentElement.dir).toBe("rtl");
      expect(screen.getAllByText(ar.layout.nav.today).length).toBeGreaterThan(0);
    });
  });
});
