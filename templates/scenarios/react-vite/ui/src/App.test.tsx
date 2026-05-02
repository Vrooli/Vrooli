/**
 * App tests — modeling the three-layer test pattern.
 *
 * 1. **Rendering tests run in cimode.** `t('app.title')` returns the key
 *    `"app.title"`. We assert via the typed `strings.*` registry, so tests
 *    survive any wording change in any locale. The setup file
 *    (`test-setup.ts`) sets cimode before every test.
 *
 * 2. **Selectors are looked up by test ID** via `selectors.*`. Test IDs
 *    are stable identifiers — they don't move when copy or DOM structure
 *    changes.
 *
 * 3. **Locale-switching tests opt back into real locales** via
 *    `await setLocale("en")` in their own `beforeEach`. They validate the
 *    end-to-end i18n pipeline (catalogs → DOM, persistence, html attrs)
 *    using raw catalog references — these tests *should* update when the
 *    canonical English copy changes, because that's what they verify.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("./lib/api", () => ({
  fetchHealth: vi.fn().mockResolvedValue({
    status: "ok",
    service: "test-service",
    timestamp: "2026-05-01T00:00:00Z",
  }),
}));

import App from "./App";
import { selectors } from "./consts/selectors";
import { strings } from "./consts/strings";
import { setLocale } from "./i18n";
import ar from "./i18n/locales/ar.json";
import en from "./i18n/locales/en.json";
import ja from "./i18n/locales/ja.json";
import { interp } from "./test-setup";

const renderApp = () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <App />
    </QueryClientProvider>,
  );
};

describe("App rendering (cimode — copy-independent)", () => {
  // No beforeEach — the setup file already puts us in cimode.

  afterEach(() => {
    cleanup();
  });

  it("renders the title element via its test id", () => {
    renderApp();
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
  });

  it("renders translation keys for the app surface (cimode echoes keys)", async () => {
    renderApp();
    expect(await screen.findByText(strings.app.eyebrow)).toBeInTheDocument();
    expect(screen.getByText(strings.app.description)).toBeInTheDocument();
    expect(screen.getByText(strings.health.title)).toBeInTheDocument();
  });

  it("exposes the refresh button regardless of label copy", () => {
    renderApp();
    expect(screen.getByTestId(selectors.health.refreshButton)).toBeInTheDocument();
  });

  it("renders the locale switcher with toggles for every supported locale", () => {
    renderApp();
    expect(screen.getByTestId(selectors.locale.switcher)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.locale.toggle({ code: "en" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.locale.toggle({ code: "ja" }))).toBeInTheDocument();
  });

  it("shows the refresh count element only after a click", async () => {
    const user = userEvent.setup();
    renderApp();

    const refreshButton = await screen.findByTestId(selectors.health.refreshButton);
    expect(screen.queryByTestId(selectors.health.refreshCount)).not.toBeInTheDocument();

    await user.click(refreshButton);
    // cimode bypasses translation entirely and returns the base key — it
    // does NOT apply CLDR plural logic. Plural-form selection is verified
    // separately in the real-locale block below; here we only assert the
    // refresh-count element appears and renders the registered key path.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.health.refreshCount)).toHaveTextContent(
        strings.health.refreshCount,
      );
    });
  });
});

describe("App locale switching (real locales — end-to-end)", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders English copy by default and reflects it on <html>", async () => {
    renderApp();
    expect(await screen.findByText(en.app.eyebrow)).toBeInTheDocument();
    expect(screen.getByText(en.app.description)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: en.health.refresh }),
    ).toBeInTheDocument();
    expect(document.documentElement.lang).toBe("en");
    expect(document.documentElement.dir).toBe("ltr");
  });

  it("switches to Japanese when the 日本語 toggle is clicked", async () => {
    const user = userEvent.setup();
    renderApp();
    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ja" })));

    await waitFor(() => {
      expect(screen.getByText(ja.app.eyebrow)).toBeInTheDocument();
    });
    expect(screen.getByText(ja.app.description)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: ja.health.refresh }),
    ).toBeInTheDocument();
    expect(document.documentElement.lang).toBe("ja");
  });

  it("flips <html dir> to rtl when an RTL locale (ar) is chosen", async () => {
    // The whole point of ar in the template: prove the LTR→RTL pipeline works
    // end-to-end. `LOCALE_CONFIG.ar.dir === "rtl"` flows through `applyDocumentLocale`
    // on `languageChanged`, and the document's `dir` should flip. Without this
    // assertion the `rtl` branch of the type would be unexercised.
    const user = userEvent.setup();
    renderApp();
    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ar" })));

    await waitFor(() => {
      expect(screen.getByText(ar.app.eyebrow)).toBeInTheDocument();
    });
    expect(document.documentElement.lang).toBe("ar");
    expect(document.documentElement.dir).toBe("rtl");
  });

  it("flips <html dir> back to ltr when returning to a non-RTL locale", async () => {
    // Direction is a stateful attribute; an rtl→ltr round-trip catches the
    // failure mode where `applyDocumentLocale` only ever sets `dir` once.
    const user = userEvent.setup();
    renderApp();
    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ar" })));
    await waitFor(() => {
      expect(document.documentElement.dir).toBe("rtl");
    });

    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "en" })));
    await waitFor(() => {
      expect(document.documentElement.dir).toBe("ltr");
    });
  });

  it("persists the chosen locale to localStorage so returning visits restore it", async () => {
    const user = userEvent.setup();
    renderApp();
    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ja" })));

    await waitFor(() => {
      expect(window.localStorage.getItem("vrooli.locale")).toBe("ja");
    });
  });

  it("marks the active locale's toggle as pressed", async () => {
    const user = userEvent.setup();
    renderApp();

    expect(screen.getByTestId(selectors.locale.toggle({ code: "en" }))).toHaveAttribute(
      "aria-pressed",
      "true",
    );

    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ja" })));

    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.locale.toggle({ code: "ja" })),
      ).toHaveAttribute("aria-pressed", "true");
    });
  });

  it("renders pluralized refresh count in real English (singular at 1)", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(screen.getByTestId(selectors.health.refreshButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.health.refreshCount)).toHaveTextContent(
        en.health.refreshCount_one,
      );
    });
  });

  it("renders pluralized refresh count in real English (plural at 3)", async () => {
    const user = userEvent.setup();
    renderApp();

    const button = screen.getByTestId(selectors.health.refreshButton);
    await user.click(button);
    await user.click(button);
    await user.click(button);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.health.refreshCount)).toHaveTextContent(
        interp(en.health.refreshCount, { count: 3 }),
      );
    });
  });

  // The notifications.summary key exercises a three-way plural shape — base
  // (`_other` fallback), `_zero`, and `_one` — to give scenario authors a
  // worked example of CLDR plurals beyond the simple singular/plural split
  // demoed by refreshCount. The three tests below cover each branch.
  it("renders zero-form plural at count=0 (notifications.summary_zero)", async () => {
    renderApp();

    await waitFor(() => {
      expect(screen.getByTestId(selectors.notifications.summary)).toHaveTextContent(
        en.notifications.summary_zero,
      );
    });
  });

  it("renders one-form plural at count=1 (notifications.summary_one)", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(screen.getByTestId(selectors.health.refreshButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.notifications.summary)).toHaveTextContent(
        en.notifications.summary_one,
      );
    });
  });

  it("renders other-form plural at count=5 (notifications.summary base)", async () => {
    const user = userEvent.setup();
    renderApp();

    const button = screen.getByTestId(selectors.health.refreshButton);
    for (let i = 0; i < 5; i++) {
      await user.click(button);
    }

    await waitFor(() => {
      expect(screen.getByTestId(selectors.notifications.summary)).toHaveTextContent(
        interp(en.notifications.summary, { count: 5 }),
      );
    });
  });
});
