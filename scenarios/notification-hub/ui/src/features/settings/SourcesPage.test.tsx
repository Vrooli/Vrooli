import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { fetchNotificationCatalog, updateNotificationPreference } from "../../api/preferences";
import { strings } from "../../consts/strings";
import { SourcesPage } from "./SourcesPage";

vi.mock("../session/useSession", () => ({
  useSession: () => ({ data: { signedIn: false, subject: "", email: "" }, isPending: false, isError: false, refetch: vi.fn() }),
}));

vi.mock("../../api/preferences", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/preferences")>();
  return { ...actual, fetchNotificationCatalog: vi.fn(), updateNotificationPreference: vi.fn() };
});

describe("SourcesPage signed-out mode", () => {
  beforeEach(() => {
    vi.mocked(fetchNotificationCatalog).mockResolvedValue({
      sources: [{
        source_scenario: "git-control-tower",
        display_name: "Git Control Tower",
        icon: "git",
        deep_link_base: "",
        declared: true,
        categories: [{
          category: "review",
          display_name: "Review reminders",
          description: "Review status changes.",
          declared: true,
          default_enabled: true,
          default_min_urgency: "normal",
        }],
      }],
      preferences: [],
    });
    vi.mocked(updateNotificationPreference).mockClear();
  });

  it("shows saved source preferences as read-only and cannot submit changes", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SourcesPage />);

    await waitFor(() => {
      expect(screen.getAllByRole("status").some((status) => status.textContent.includes(strings.settings.sources.readOnly))).toBe(true);
    });
    const toggle = await screen.findByRole("switch");
    expect(toggle).toBeDisabled();
    await user.click(toggle);
    expect(updateNotificationPreference).not.toHaveBeenCalled();
  });
});
