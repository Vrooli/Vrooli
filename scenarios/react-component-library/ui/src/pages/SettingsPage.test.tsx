import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { SettingsPage } from "./SettingsPage";

describe("SettingsPage", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => cleanup());

  it("renders theme and language segmented controls", () => {
    renderWithProviders(<SettingsPage />);
    expect(screen.getByTestId("settings-page")).toBeInTheDocument();
    expect(screen.getByTestId("settings-theme")).toBeInTheDocument();
    expect(screen.getByTestId("settings-locale")).toBeInTheDocument();
  });

  it("flips the theme selection when a button is clicked", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);
    await user.click(screen.getByTestId("settings-theme-dark"));
    await waitFor(() =>
      expect(screen.getByTestId("settings-theme-dark")).toHaveAttribute("aria-checked", "true"),
    );
  });
});
