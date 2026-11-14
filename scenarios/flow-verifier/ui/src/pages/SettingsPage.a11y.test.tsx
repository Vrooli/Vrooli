import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";

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

describe("SettingsPage accessibility", () => {
  beforeEach(async () => {
    const prefs = await import("../lib/preferences");
    vi.mocked(prefs.fetchSettings).mockResolvedValue(DEFAULT_SETTINGS);
    vi.mocked(prefs.readCache).mockReturnValue(undefined);
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<SettingsPage />);
    await waitFor(() =>
      expect(screen.queryByTestId("settings-loading")).not.toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
