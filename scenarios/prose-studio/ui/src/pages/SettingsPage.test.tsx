import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { SettingsPage } from "./SettingsPage";
import { selectors } from "../consts/selectors";

describe("SettingsPage", () => {
  afterEach(() => cleanup());

  it("supports theme and locale choices", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);
    const dark = screen.getByTestId(selectors.settingsPage.themeOption({ choice: "dark" }));
    await user.click(dark);
    expect(dark).toHaveAttribute("aria-checked", "true");
    await user.click(screen.getByTestId(selectors.settingsPage.localeOption({ code: "ja" })));
    expect(screen.getByTestId(selectors.settingsPage.localeOption({ code: "ja" }))).toHaveAttribute("aria-checked", "true");
  });
});
