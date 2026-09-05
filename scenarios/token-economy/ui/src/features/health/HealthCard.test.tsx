/**
 * HealthCard tests — focused on the health-card surface only.
 *
 * Renders <HealthCard /> directly (not through <App />) so failures
 * point at health-feature behaviour, not shell composition. Plural
 * rendering and locale-driven copy still need real catalogs, so the
 * "real-locale" block opts in via setLocale("en").
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { interp, makeApiMocks, renderWithProviders } from "../../test-utils";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});

import { HealthCard } from "../../components/HealthCard";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { setLocale } from "../../i18n";
import en from "../../i18n/locales/en.json";

describe("HealthCard rendering (cimode — copy-independent)", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the health-card root via its test id", () => {
    renderWithProviders(<HealthCard />);
    expect(screen.getByTestId(selectors.health.card)).toBeInTheDocument();
  });

  it("renders the health title via the strings registry", () => {
    renderWithProviders(<HealthCard />);
    expect(screen.getByText(strings.health.title)).toBeInTheDocument();
  });

  it("exposes the refresh button regardless of label copy", () => {
    renderWithProviders(<HealthCard />);
    expect(screen.getByTestId(selectors.health.refreshButton)).toBeInTheDocument();
  });

  it("shows the refresh count element only after a click", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HealthCard />);

    const refreshButton = await screen.findByTestId(selectors.health.refreshButton);
    expect(screen.queryByTestId(selectors.health.refreshCount)).not.toBeInTheDocument();

    await user.click(refreshButton);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.health.refreshCount)).toHaveTextContent(
        strings.health.refreshCount,
      );
    });
  });
});

describe("HealthCard plurals (real English locale)", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders pluralized refresh count in real English (singular at 1)", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HealthCard />);

    await user.click(screen.getByTestId(selectors.health.refreshButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.health.refreshCount)).toHaveTextContent(
        en.health.refreshCount_one,
      );
    });
  });

  it("renders pluralized refresh count in real English (plural at 3)", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HealthCard />);

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

  // notifications.summary exercises a three-way plural shape — base
  // (`_other` fallback), `_zero`, and `_one` — to give scenario authors
  // a worked example of CLDR plurals beyond simple singular/plural.
  it("renders zero-form plural at count=0 (notifications.summary_zero)", async () => {
    renderWithProviders(<HealthCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.notifications.summary)).toHaveTextContent(
        en.notifications.summary_zero,
      );
    });
  });

  it("renders one-form plural at count=1 (notifications.summary_one)", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HealthCard />);

    await user.click(screen.getByTestId(selectors.health.refreshButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.notifications.summary)).toHaveTextContent(
        en.notifications.summary_one,
      );
    });
  });

  it("renders other-form plural at count=5 (notifications.summary base)", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HealthCard />);

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
