/**
 * SettingsPage accessibility regression test.
 *
 * The settings page owns several radiogroups (theme, locale, and the new
 * display/accessibility preferences) plus an embedded query-backed card, so a
 * full-page axe scan guards the labelling of all of them at once.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { makeModelsMocks } from "../features/models/mocks/models";

vi.mock("../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

import { SettingsPage } from "./SettingsPage";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";

describe("SettingsPage accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    window.localStorage.clear();
  });

  afterEach(async () => {
    cleanup();
    vi.clearAllMocks();
    await setLocale("en");
  });

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<SettingsPage />);

    // Wait for the embedded ModelDefaultsCard to settle into its empty state so
    // the scan runs against the resolved DOM.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.defaults.empty)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });
});
