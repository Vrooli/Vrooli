import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { makeHealthResponse, renderWithProviders } from "../test-utils";

const { fetchHealth } = vi.hoisted(() => ({ fetchHealth: vi.fn() }));

vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth };
});

import { SettingsPage } from "./SettingsPage";
import { selectors } from "../consts/selectors";

describe("SettingsPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the appearance, language, owner and session sections", () => {
    fetchHealth.mockResolvedValue(makeHealthResponse());
    renderWithProviders(<SettingsPage />);

    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.settingsPage.themeOption({ choice: "light" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.settingsPage.localeOption({ code: "en" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.owner.panel)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.session.panel)).toBeInTheDocument();
  });

  it("selects a theme via the radiogroup", async () => {
    const user = userEvent.setup();
    fetchHealth.mockResolvedValue(makeHealthResponse());
    renderWithProviders(<SettingsPage />, { initialTheme: "light" });

    const dark = screen.getByTestId(selectors.settingsPage.themeOption({ choice: "dark" }));
    expect(dark).toHaveAttribute("aria-checked", "false");
    await user.click(dark);
    await waitFor(() => expect(dark).toHaveAttribute("aria-checked", "true"));
  });

  it("selects a locale via the radiogroup", async () => {
    const user = userEvent.setup();
    fetchHealth.mockResolvedValue(makeHealthResponse());
    renderWithProviders(<SettingsPage />);

    const ja = screen.getByTestId(selectors.settingsPage.localeOption({ code: "ja" }));
    await user.click(ja);
    await waitFor(() => expect(ja).toHaveAttribute("aria-checked", "true"));
  });
});
