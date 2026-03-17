import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import SettingsModal from "../components/SettingsModal";
import type { ShortcutProfile } from "../lib/api";

// [REQ:P1-002a] Shortcut Profile Management UI — component rendering & interactions
// [REQ:P1-003a] AI Provider Configuration UI — SettingsModal integration

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

vi.mock("../components/IntegrationsPanel", () => ({
  default: () => <div data-testid="integrations-panel">IntegrationsPanel</div>,
}));

// Mock workspace store
const mockStoreState: Record<string, unknown> = {
  panes: [],
  settingsModalOpen: true,
  movePaneToIndex: vi.fn(),
  removePane: vi.fn(),
  setActivePane: vi.fn(),
  setSettingsModalOpen: vi.fn(),
  resetLayout: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector(mockStoreState),
}));

vi.mock("../hooks/useDraggablePosition", () => ({
  useDraggablePosition: () => ({
    elementRef: { current: null },
    floatingStyle: { transform: "translate3d(100px, 100px, 0)" },
    pointerHandlers: {
      onPointerDown: vi.fn(),
      onPointerMove: vi.fn(),
      onPointerUp: vi.fn(),
      onPointerCancel: vi.fn(),
    },
    handleClickCapture: vi.fn(),
    resetPosition: vi.fn(),
    moveTo: vi.fn(),
    isDragging: false,
    position: { x: 100, y: 100 },
  }),
}));

describe("SettingsModal (shortcut profiles)", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    mockStoreState.settingsModalOpen = true;
    mockStoreState.panes = [];
    const api = await import("../lib/api");
    mockListProfiles = api.listShortcutProfiles as ReturnType<typeof vi.fn>;
    mockUpsertProfile = api.upsertShortcutProfile as ReturnType<typeof vi.fn>;
    mockDeleteProfile = api.deleteShortcutProfile as ReturnType<typeof vi.fn>;
  });

  it("renders loading state initially", () => {
    mockListProfiles.mockReturnValue(new Promise(() => {}));
    render(<SettingsModal />);
    expect(screen.getByText("Loading...")).toBeTruthy();
  });

  it("renders empty state when no profiles exist", async () => {
    mockListProfiles.mockResolvedValueOnce([]);
    render(<SettingsModal />);

    await waitFor(() => {
      expect(screen.getByText("No shortcut profiles configured")).toBeTruthy();
    });
  });

  it("renders profiles after loading", async () => {
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    render(<SettingsModal />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    const nameInput = screen.getByTestId("profile-name-prof-1") as HTMLInputElement;
    expect(nameInput.value).toBe("Dev Shortcuts");
    expect(screen.getByTestId("entry-label-prof-1-0")).toBeTruthy();
    expect(screen.getByTestId("entry-command-prof-1-0")).toBeTruthy();
  });

  it("shows save button when profile name is edited (dirty tracking)", async () => {
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    render(<SettingsModal />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    expect(screen.queryByTestId("profile-save-prof-1")).toBeNull();

    const nameInput = screen.getByTestId("profile-name-prof-1");
    fireEvent.change(nameInput, { target: { value: "Updated Name" } });
    expect(screen.getByTestId("profile-save-prof-1")).toBeTruthy();
  });

  it("calls upsertShortcutProfile when save button is clicked", async () => {
    const updatedProfile = { ...mockProfile, name: "Renamed" };
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    mockUpsertProfile.mockResolvedValueOnce(updatedProfile);

    render(<SettingsModal />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    const nameInput = screen.getByTestId("profile-name-prof-1");
    fireEvent.change(nameInput, { target: { value: "Renamed" } });
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

    render(<SettingsModal />);

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

    render(<SettingsModal />);

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
    render(<SettingsModal />);

    await waitFor(() => {
      expect(screen.getByTestId("settings-error")).toBeTruthy();
      expect(screen.getByText("Network error")).toBeTruthy();
    });
  });

  it("renders IntegrationsPanel in the Integrations section", async () => {
    mockListProfiles.mockResolvedValueOnce([]);
    render(<SettingsModal />);

    await waitFor(() => {
      expect(screen.getByTestId("integrations-panel")).toBeTruthy();
    });

    expect(screen.getByText("Integrations")).toBeTruthy();
  });

  it("shows error and refetches when save fails", async () => {
    mockListProfiles
      .mockResolvedValueOnce([mockProfile])
      .mockResolvedValueOnce([mockProfile]);
    mockUpsertProfile.mockRejectedValueOnce(new Error("Save failed"));

    render(<SettingsModal />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });

    fireEvent.change(screen.getByTestId("profile-name-prof-1"), {
      target: { value: "Changed" },
    });
    fireEvent.click(screen.getByTestId("profile-save-prof-1"));

    await waitFor(() => {
      expect(screen.getByTestId("settings-error")).toBeTruthy();
      expect(screen.getByText("Save failed")).toBeTruthy();
    });

    expect(mockListProfiles).toHaveBeenCalledTimes(2);
  });
});
