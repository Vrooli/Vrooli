import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";

vi.mock("../lib/preferences", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/preferences")>();
  return {
    ...actual,
    fetchSettings: vi.fn(),
    putSettings: vi.fn(),
    readCache: vi.fn(),
  };
});

import { SettingsPage } from "./SettingsPage";
import { DEFAULT_SETTINGS } from "../lib/preferences";

describe("SettingsPage", () => {
  beforeEach(async () => {
    const prefs = await import("../lib/preferences");
    vi.mocked(prefs.fetchSettings).mockResolvedValue(DEFAULT_SETTINGS);
    vi.mocked(prefs.putSettings).mockImplementation((p) =>
      Promise.resolve({ ...DEFAULT_SETTINGS, ...p }),
    );
    vi.mocked(prefs.readCache).mockReturnValue(undefined);
  });
  afterEach(() => cleanup());

  it("renders appearance and behavior sections", async () => {
    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByTestId("settings-page")).toBeInTheDocument());
    expect(screen.getByTestId("settings-theme")).toBeInTheDocument();
    expect(screen.getByTestId("settings-density")).toBeInTheDocument();
    expect(screen.getByTestId("settings-font-scale")).toBeInTheDocument();
    expect(screen.getByTestId("settings-reduced-motion")).toBeInTheDocument();
    expect(screen.getByTestId("settings-rtl")).toBeInTheDocument();
    expect(screen.getByTestId("settings-default-root")).toBeInTheDocument();
  });

  it("calls putSettings when the theme is changed", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByTestId("settings-page")).toBeInTheDocument());
    await user.click(screen.getByTestId("settings-theme-dark"));
    const { putSettings } = await import("../lib/preferences");
    expect(vi.mocked(putSettings).mock.calls[0]?.[0]).toEqual(
      expect.objectContaining({ theme: "dark" }),
    );
  });

  it("toggles reduced-motion via the switch", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByTestId("settings-page")).toBeInTheDocument());
    await user.click(screen.getByTestId("settings-reduced-motion"));
    const { putSettings } = await import("../lib/preferences");
    const calls = vi.mocked(putSettings).mock.calls;
    expect(calls.some((c) => (c[0] as { reducedMotion?: boolean }).reducedMotion === true)).toBe(true);
  });
});
