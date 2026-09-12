import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
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
    { $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "List files", command: "ls -la", description: "", agentId: "" },
    { $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "Git status", command: "git status", description: "", agentId: "" },
  ],
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};

/**
 * The uid of the nth rendered command row. Rows are keyed by a stable draft
 * id rather than the array index, so tests address them the same way the
 * component does.
 */
function entryUID(index = 0): string {
  const rows = screen.getAllByTestId(/^shortcut-entry-/);
  const row = rows[index];
  if (!row) throw new Error(`no shortcut entry row at index ${String(index)}`);
  return (row.getAttribute("data-testid") ?? "").replace("shortcut-entry-", "");
}

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
          { label: "List files", command: "ls -la", description: "", agentId: "" },
          { label: "Git status", command: "git status", description: "", agentId: "" },
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

    // Deleting a profile takes every command in it, so it asks first. Before
    // the confirmation step the icon deleted on the first press.
    fireEvent.click(screen.getByTestId("profile-delete-prof-1"));
    expect(shortcutsClient.deleteProfile).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("profile-delete-confirmed-prof-1"));
    await waitFor(() => {
      expect(shortcutsClient.deleteProfile).toHaveBeenCalledWith({ id: "prof-1" });
    });
  });

  // The description lived in the draft type and had no input at all: it was
  // read on load and written on save, so an operator could lose it but never
  // change it.
  it("lets the operator edit a command's description and saves it", async () => {
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [mockProfile] });
    shortcutsClient.upsertProfile.mockResolvedValueOnce({ profile: mockProfile });
    render(<ShortcutProfilesSection />);
    await waitFor(() => { expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy(); });

    fireEvent.change(screen.getByTestId(`entry-description-${entryUID()}`), { target: { value: "Long listing" } });
    fireEvent.click(screen.getByTestId("profile-save-prof-1"));

    await waitFor(() => {
      expect(shortcutsClient.upsertProfile).toHaveBeenCalledWith(
        expect.objectContaining({
          shortcuts: expect.arrayContaining([
            expect.objectContaining({ label: "List files", description: "Long listing" }),
          ]),
        }),
      );
    });
  });

  // This screen is the only one that can tell an operator, before they save,
  // that a command will silently stop their messages being recorded.
  it("warns that a bare agent command may not be captured, and offers the fix", async () => {
    const withCodex: ShortcutProfile = {
      ...mockProfile,
      shortcuts: [
        { $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "Codex", command: "codex --yolo", description: "", agentId: "codex" },
      ],
    };
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [withCodex] });
    render(<ShortcutProfilesSection />);
    await waitFor(() => { expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy(); });

    expect(screen.getByTestId("capture-note-warning")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("capture-use-governed"));

    expect(screen.getByTestId(`entry-command-${entryUID()}`)).toHaveValue("vrooli agent launch --runner codex --arg=--yolo");
    expect(screen.getByTestId("capture-note-governed")).toBeInTheDocument();
    expect(screen.queryByTestId("capture-note-warning")).not.toBeInTheDocument();
  });

  it("says nothing about capture for a command that launches no agent", async () => {
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [mockProfile] });
    render(<ShortcutProfilesSection />);
    await waitFor(() => { expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy(); });
    expect(screen.queryByTestId("capture-note-warning")).not.toBeInTheDocument();
    expect(screen.queryByTestId("capture-note-governed")).not.toBeInTheDocument();
  });

  // Rows were keyed by array index, which makes a reorder reuse the wrong
  // row's state. Reordering is now a feature, so the keys had to become real.
  it("reorders commands from the keyboard and saves the new order", async () => {
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [mockProfile] });
    shortcutsClient.upsertProfile.mockResolvedValueOnce({ profile: mockProfile });
    render(<ShortcutProfilesSection />);
    await waitFor(() => { expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy(); });

    fireEvent.keyDown(screen.getByTestId(`shortcut-grip-${entryUID()}`), { key: "ArrowDown", altKey: true });
    fireEvent.click(screen.getByTestId("profile-save-prof-1"));

    await waitFor(() => {
      expect(shortcutsClient.upsertProfile).toHaveBeenCalledWith(
        expect.objectContaining({
          shortcuts: [
            expect.objectContaining({ label: "Git status" }),
            expect.objectContaining({ label: "List files" }),
          ],
        }),
      );
    });
  });

  it("discards pending edits without touching the server", async () => {
    shortcutsClient.listProfiles.mockResolvedValueOnce({ profiles: [mockProfile] });
    render(<ShortcutProfilesSection />);
    await waitFor(() => { expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy(); });

    fireEvent.change(screen.getByTestId("profile-name-prof-1"), { target: { value: "Scratch" } });
    expect(screen.getByTestId("profile-save-bar-prof-1")).toBeInTheDocument();
    fireEvent.click(screen.getByText(strings.settings.shortcutsSection.discard));

    expect(screen.getByTestId("profile-name-prof-1")).toHaveValue("Dev Shortcuts");
    expect(screen.queryByTestId("profile-save-bar-prof-1")).not.toBeInTheDocument();
    expect(shortcutsClient.upsertProfile).not.toHaveBeenCalled();
  });

  it("creates a new profile via upsertProfile", async () => {
    const newProfile: ShortcutProfile = {
      $typeName: "vrooli.web_console.v1.shortcuts.Profile",
      id: "prof-new",
      scope: "workspace",
      name: "New Profile",
      shortcuts: [{ $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "List files", command: "ls -la", description: "", agentId: "" }],
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
