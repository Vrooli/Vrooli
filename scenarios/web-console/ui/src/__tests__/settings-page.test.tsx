import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import SettingsPage from "../pages/SettingsPage";
import type { ShortcutProfile } from "../lib/api";

// [REQ:P1-002a] Shortcut Profile Management UI — component rendering & interactions
// [REQ:P1-003a] AI Provider Configuration UI — SettingsPage integration

const mockProfile: ShortcutProfile = {
  id: "prof-1",
  scope: "workspace",
  name: "Dev Shortcuts",
  shortcuts: [
    { label: "List files", command: "ls -la" },
    { label: "Git status", command: "git status" },
  ],
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

let mockListProfiles: ReturnType<typeof vi.fn>;
let mockUpsertProfile: ReturnType<typeof vi.fn>;
let mockDeleteProfile: ReturnType<typeof vi.fn>;

vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return {
    ...actual,
    listShortcutProfiles: vi.fn(),
    upsertShortcutProfile: vi.fn(),
    deleteShortcutProfile: vi.fn(),
  };
});

vi.mock("../components/ProviderHealthPanel", () => ({
  default: () => <div data-testid="provider-health-panel">ProviderHealthPanel</div>,
}));

describe("SettingsPage", () => {
  const onBack = vi.fn();

  beforeEach(async () => {
    vi.clearAllMocks();
    const api = await import("../lib/api");
    mockListProfiles = api.listShortcutProfiles as ReturnType<typeof vi.fn>;
    mockUpsertProfile = api.upsertShortcutProfile as ReturnType<typeof vi.fn>;
    mockDeleteProfile = api.deleteShortcutProfile as ReturnType<typeof vi.fn>;
  });

  it("renders loading state initially", () => {
    mockListProfiles.mockReturnValue(new Promise(() => {}));
    render(<SettingsPage onBack={onBack} />);

    expect(screen.getByText("Loading...")).toBeTruthy();
  });

  it("renders empty state when no profiles exist", async () => {
    mockListProfiles.mockResolvedValueOnce([]);
    render(<SettingsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByText("No shortcut profiles configured")).toBeTruthy();
    });
  });

  it("renders profiles after loading", async () => {
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    render(<SettingsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    // Profile name is displayed
    const nameInput = screen.getByTestId("profile-name-prof-1") as HTMLInputElement;
    expect(nameInput.value).toBe("Dev Shortcuts");

    // Shortcut entries are displayed
    expect(screen.getByTestId("entry-label-prof-1-0")).toBeTruthy();
    expect(screen.getByTestId("entry-command-prof-1-0")).toBeTruthy();
  });

  it("calls onBack when back button is clicked", async () => {
    mockListProfiles.mockResolvedValueOnce([]);
    render(<SettingsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByText("No shortcut profiles configured")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("settings-back"));
    expect(onBack).toHaveBeenCalledOnce();
  });

  it("shows save button when profile name is edited (dirty tracking)", async () => {
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    render(<SettingsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    // Save button should NOT be visible initially (not dirty)
    expect(screen.queryByTestId("profile-save-prof-1")).toBeNull();

    // Edit the profile name
    const nameInput = screen.getByTestId("profile-name-prof-1");
    fireEvent.change(nameInput, { target: { value: "Updated Name" } });

    // Save button should appear
    expect(screen.getByTestId("profile-save-prof-1")).toBeTruthy();
  });

  it("calls upsertShortcutProfile when save button is clicked", async () => {
    const updatedProfile = { ...mockProfile, name: "Renamed" };
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    mockUpsertProfile.mockResolvedValueOnce(updatedProfile);

    render(<SettingsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    // Make a change to trigger dirty state
    const nameInput = screen.getByTestId("profile-name-prof-1");
    fireEvent.change(nameInput, { target: { value: "Renamed" } });

    // Click save
    fireEvent.click(screen.getByTestId("profile-save-prof-1"));

    expect(mockUpsertProfile).toHaveBeenCalledWith({
      id: "prof-1",
      scope: "workspace",
      name: "Renamed",
      shortcuts: mockProfile.shortcuts,
    });
  });

  it("deletes profile when delete button is clicked", async () => {
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    mockDeleteProfile.mockResolvedValueOnce(undefined);

    render(<SettingsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("profile-delete-prof-1"));

    expect(mockDeleteProfile).toHaveBeenCalledWith("prof-1");

    await waitFor(() => {
      expect(screen.queryByTestId("shortcut-profile-prof-1")).toBeNull();
    });
  });

  it("creates new profile when 'New Profile' button is clicked", async () => {
    const newProfile: ShortcutProfile = {
      id: "prof-new",
      scope: "workspace",
      name: "New Profile",
      shortcuts: [{ label: "List files", command: "ls -la" }],
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };

    mockListProfiles.mockResolvedValueOnce([]);
    mockUpsertProfile.mockResolvedValueOnce(newProfile);

    render(<SettingsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByText("No shortcut profiles configured")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("create-profile"));

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-new")).toBeTruthy();
    });
  });

  it("shows error banner when profile load fails", async () => {
    mockListProfiles.mockRejectedValueOnce(new Error("Network error"));
    render(<SettingsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("settings-error")).toBeTruthy();
      expect(screen.getByText("Network error")).toBeTruthy();
    });
  });

  it("renders ProviderHealthPanel in the AI Providers section", async () => {
    mockListProfiles.mockResolvedValueOnce([]);
    render(<SettingsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("provider-health-panel")).toBeTruthy();
    });

    expect(screen.getByText("AI Providers")).toBeTruthy();
  });

  it("shows error and refetches when save fails", async () => {
    mockListProfiles
      .mockResolvedValueOnce([mockProfile])
      .mockResolvedValueOnce([mockProfile]); // refetch after error
    mockUpsertProfile.mockRejectedValueOnce(new Error("Save failed"));

    render(<SettingsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    // Make a change and save
    fireEvent.change(screen.getByTestId("profile-name-prof-1"), {
      target: { value: "Changed" },
    });
    fireEvent.click(screen.getByTestId("profile-save-prof-1"));

    await waitFor(() => {
      expect(screen.getByTestId("settings-error")).toBeTruthy();
      expect(screen.getByText("Save failed")).toBeTruthy();
    });

    // Should have refetched profiles
    expect(mockListProfiles).toHaveBeenCalledTimes(2);
  });
});
