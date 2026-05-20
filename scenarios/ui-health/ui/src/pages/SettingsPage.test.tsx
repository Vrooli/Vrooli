import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { SettingsPage } from "./SettingsPage";

describe("SettingsPage", () => {
  it("renders the theme + locale radio groups", () => {
    renderWithProviders(<SettingsPage />);
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.settingsPage.themeOption({ choice: "light" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.settingsPage.themeOption({ choice: "dark" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.settingsPage.themeOption({ choice: "system" })),
    ).toBeInTheDocument();
  });

  it("flips the aria-checked state when a theme is selected", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);
    const darkBtn = screen.getByTestId(selectors.settingsPage.themeOption({ choice: "dark" }));
    await user.click(darkBtn);
    expect(darkBtn).toHaveAttribute("aria-checked", "true");
  });
});
