import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import ShortcutProfilesSection from "../components/settings/ShortcutProfilesSection";
import type { ShortcutProfile } from "../lib/api";

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

describe("ShortcutProfilesSection", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    const api = await import("../lib/api");
    mockListProfiles = api.listShortcutProfiles as ReturnType<typeof vi.fn>;
    mockUpsertProfile = api.upsertShortcutProfile as ReturnType<typeof vi.fn>;
    mockDeleteProfile = api.deleteShortcutProfile as ReturnType<typeof vi.fn>;
  });

  it("renders loading state initially", () => {
    mockListProfiles.mockReturnValue(new Promise(() => {}));
    render(<ShortcutProfilesSection />);
    expect(screen.getByText("Loading...")).toBeTruthy();
  });

  it("renders empty state when no profiles exist", async () => {
    mockListProfiles.mockResolvedValueOnce([]);
    render(<ShortcutProfilesSection />);
    await waitFor(() => {
      expect(screen.getByText("No shortcut profiles configured")).toBeTruthy();
    });
  });

  it("renders profiles after loading", async () => {
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    render(<ShortcutProfilesSection />);
    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });
  });

  it("shows save button when profile name is edited", async () => {
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    render(<ShortcutProfilesSection />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    fireEvent.change(screen.getByTestId("profile-name-prof-1"), {
      target: { value: "Updated Name" },
    });
    expect(screen.getByTestId("profile-save-prof-1")).toBeTruthy();
  });

  it("calls upsertShortcutProfile when save button is clicked", async () => {
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    mockUpsertProfile.mockResolvedValueOnce({ ...mockProfile, name: "Renamed" });

    render(<ShortcutProfilesSection />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    fireEvent.change(screen.getByTestId("profile-name-prof-1"), {
      target: { value: "Renamed" },
    });
    fireEvent.click(screen.getByTestId("profile-save-prof-1"));

    await waitFor(() => {
      expect(mockUpsertProfile).toHaveBeenCalledWith({
        id: "prof-1",
        scope: "workspace",
        name: "Renamed",
        shortcuts: mockProfile.shortcuts,
      });
    });
  });

  it("deletes profile when delete button is clicked", async () => {
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    mockDeleteProfile.mockResolvedValueOnce(undefined);

    render(<ShortcutProfilesSection />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("profile-delete-prof-1"));
    await waitFor(() => {
      expect(mockDeleteProfile).toHaveBeenCalledWith("prof-1");
    });
  });

  it("creates a new profile", async () => {
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

    render(<ShortcutProfilesSection />);

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
    render(<ShortcutProfilesSection />);
    await waitFor(() => {
      expect(screen.getByTestId("settings-error")).toBeTruthy();
      expect(screen.getByText("Network error")).toBeTruthy();
    });
  });
});
