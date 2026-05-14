import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import type { Profile as ShortcutProfile } from "@vrooli/proto-types/web-console/v1/shortcuts/shortcuts_pb";
import { asMockedClient } from "../test-utils";
import { strings } from "../consts/strings";

// [REQ:P1-002a] Shortcut Profile Management UI — exercises ShortcutsService
// via the Connect-Web client. The domain module is mocked so tests assert
// against client method calls directly, without a transport.

vi.mock("../api/shortcuts", () => ({
  shortcutsClient: {
    getEffective: vi.fn(),
    listProfiles: vi.fn(),
    upsertProfile: vi.fn(),
    deleteProfile: vi.fn(),
  },
}));

import { shortcutsClient as _shortcutsClient } from "../api/shortcuts";
const shortcutsClient = asMockedClient(_shortcutsClient);

import ShortcutProfilesSection from "../components/settings/ShortcutProfilesSection";

const mockProfile: ShortcutProfile = {
  $typeName: "vrooli.web_console.v1.shortcuts.Profile",
  id: "prof-1",
  scope: "workspace",
  name: "Dev Shortcuts",
  shortcuts: [
    { $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "List files", command: "ls -la", description: "" },
    { $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "Git status", command: "git status", description: "" },
  ],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};

describe("ShortcutProfilesSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state initially", () => {
    shortcutsClient.listProfiles.mockReturnValue(new Promise(() => {}));
    render(<ShortcutProfilesSection />);
    expect(screen.getByText(strings.settings.shortcutsSection.loading)).toBeTruthy();
  });

  it("renders empty state when no profiles exist", async () => {
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [] });
    render(<ShortcutProfilesSection />);
    await waitFor(() => {
      expect(screen.getByText(strings.settings.shortcutsSection.empty)).toBeTruthy();
    });
  });

  it("renders profiles after loading", async () => {
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [mockProfile] });
    render(<ShortcutProfilesSection />);
    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });
  });

  it("shows save button when profile name is edited", async () => {
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [mockProfile] });
    render(<ShortcutProfilesSection />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    fireEvent.change(screen.getByTestId("profile-name-prof-1"), {
      target: { value: "Updated Name" },
    });
    expect(screen.getByTestId("profile-save-prof-1")).toBeTruthy();
  });

  it("calls upsertProfile with draft fields when save is clicked", async () => {
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [mockProfile] });
    shortcutsClient.upsertProfile.mockResolvedValueOnce({
      profile: { ...mockProfile, name: "Renamed" },
    });

    render(<ShortcutProfilesSection />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    fireEvent.change(screen.getByTestId("profile-name-prof-1"), {
      target: { value: "Renamed" },
    });
    fireEvent.click(screen.getByTestId("profile-save-prof-1"));

    await waitFor(() => {
      expect(shortcutsClient.upsertProfile).toHaveBeenCalledWith({
        id: "prof-1",
        scope: "workspace",
        name: "Renamed",
        shortcuts: [
          { label: "List files", command: "ls -la", description: "" },
          { label: "Git status", command: "git status", description: "" },
        ],
      });
    });
  });

  it("deletes profile when delete button is clicked", async () => {
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [mockProfile] });
    shortcutsClient.deleteProfile.mockResolvedValueOnce({});

    render(<ShortcutProfilesSection />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("profile-delete-prof-1"));
    await waitFor(() => {
      expect(shortcutsClient.deleteProfile).toHaveBeenCalledWith({ id: "prof-1" });
    });
  });

  it("creates a new profile via upsertProfile", async () => {
    const newProfile: ShortcutProfile = {
      $typeName: "vrooli.web_console.v1.shortcuts.Profile",
      id: "prof-new",
      scope: "workspace",
      name: "New Profile",
      shortcuts: [{ $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "List files", command: "ls -la", description: "" }],
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
    };
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [] });
    shortcutsClient.upsertProfile.mockResolvedValueOnce({ profile: newProfile });

    render(<ShortcutProfilesSection />);

    await waitFor(() => {
      expect(screen.getByText(strings.settings.shortcutsSection.empty)).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("create-profile"));

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-new")).toBeTruthy();
    });
    expect(shortcutsClient.upsertProfile).toHaveBeenCalled();
  });

  it("shows error banner when profile load fails", async () => {
    shortcutsClient.listProfiles.mockRejectedValueOnce(new Error("Network error"));
    render(<ShortcutProfilesSection />);
    await waitFor(() => {
      expect(screen.getByTestId("settings-error")).toBeTruthy();
      expect(screen.getByText("Network error")).toBeTruthy();
    });
  });
});
