import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { componentsClient } from "../../api/components";
import { renderWithProviders } from "../../test-utils";
import { ThemeSwitcher, type PreviewKit } from "./ThemeSwitcher";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/components", () => ({
  componentsClient: {
    listDesignStyles: vi.fn().mockResolvedValue({
      styles: [
        { id: "vrooli-default", name: "Vrooli Operational Console" },
        { id: "vrooli-command-display", name: "Vrooli Command Display" },
        { id: "registry-fixture", name: "Registry Fixture" },
      ],
    }),
  },
}));

const filters = {
  visionFilter: "none" as const,
  setVisionFilter: vi.fn(),
  blurPx: 0,
  setBlurPx: vi.fn(),
  blurMin: 0,
  blurMax: 10,
};

function Harness({
  kit = "vrooli-default",
  setKit = vi.fn(),
  setColorScheme = vi.fn(),
}: {
  kit?: PreviewKit;
  setKit?: (kit: PreviewKit) => void;
  setColorScheme?: (scheme: "system" | "light" | "dark") => void;
}) {
  return (
    <ThemeSwitcher
      previewReady
      colorScheme="system"
      setColorScheme={setColorScheme}
      kit={kit}
      setKit={setKit}
      filters={filters}
    />
  );
}

describe("ThemeSwitcher", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("changes the kit without sending a token overlay", async () => {
    const setKit = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<Harness setKit={setKit} />);
    await user.click(screen.getByTestId(selectors.components.themeSwitcher.appearanceToggle));
    await user.click(screen.getByTestId(selectors.components.themeSwitcher.kitSelect));
    const commandDisplay = await screen.findByRole("option", { name: "Vrooli Command Display" });
    await user.click(commandDisplay);
    expect(setKit).toHaveBeenCalledWith("vrooli-command-display");
  });

  it("renders kits returned by the design-style registry", async () => {
    const setKit = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<Harness setKit={setKit} />);
    await user.click(screen.getByTestId(selectors.components.themeSwitcher.appearanceToggle));
    await waitFor(() => {
      expect(componentsClient.listDesignStyles).toHaveBeenCalledWith({});
    });
    await user.click(screen.getByTestId(selectors.components.themeSwitcher.kitSelect));
    const registryFixture = await screen.findByRole("option", { name: "Registry Fixture" });
    expect(registryFixture).toBeInTheDocument();
    await user.click(registryFixture);

    expect(setKit).toHaveBeenCalledWith("registry-fixture");
  });

  it("keeps light and dark as the explicit mode controls", async () => {
    const setColorScheme = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<Harness setColorScheme={setColorScheme} />);
    await user.click(screen.getByTestId(selectors.components.themeSwitcher.appearanceToggle));
    await user.click(screen.getByTestId(selectors.components.themeSwitcher.modeDark));
    expect(setColorScheme).toHaveBeenCalledWith("dark");
  });

  it("portals the appearance panel above the preview dock", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Harness />);
    await user.click(screen.getByTestId(selectors.components.themeSwitcher.appearanceToggle));
    const panel = screen.getByTestId(selectors.components.themeSwitcher.appearancePanel);
    expect(panel.closest("body")).toBe(document.body);
    expect(panel).toHaveAttribute("data-rcl-popover-content");
    expect(panel).toHaveClass("fixed");
  });
});
