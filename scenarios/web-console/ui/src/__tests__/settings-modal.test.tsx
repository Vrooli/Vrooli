import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import SettingsModal from "../components/SettingsModal";
import type { ShortcutProfile } from "../lib/api";

// Mock workspace store
const mockStoreState = {
  settingsModalOpen: true,
  setSettingsModalOpen: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector(mockStoreState),
}));

// Mock draggable position hook
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
    isDragging: false,
    position: { x: 100, y: 100 },
  }),
}));

// Mock API
let mockListProfiles: ReturnType<typeof vi.fn>;

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

const mockProfile: ShortcutProfile = {
  id: "prof-1",
  scope: "workspace",
  name: "Dev Shortcuts",
  shortcuts: [
    { label: "List files", command: "ls -la" },
  ],
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("SettingsModal", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    mockStoreState.settingsModalOpen = true;
    const api = await import("../lib/api");
    mockListProfiles = api.listShortcutProfiles as ReturnType<typeof vi.fn>;
    mockListProfiles.mockResolvedValue([]);
  });

  it("does not render when settingsModalOpen is false", () => {
    mockStoreState.settingsModalOpen = false;
    render(<SettingsModal />);
    expect(screen.queryByTestId("settings-modal")).toBeNull();
  });

  it("renders modal when settingsModalOpen is true", () => {
    render(<SettingsModal />);
    expect(screen.getByTestId("settings-modal")).toBeTruthy();
    expect(screen.getByTestId("settings-backdrop")).toBeTruthy();
  });

  it("closes on backdrop click", () => {
    render(<SettingsModal />);
    fireEvent.click(screen.getByTestId("settings-backdrop"));
    expect(mockStoreState.setSettingsModalOpen).toHaveBeenCalledWith(false);
  });

  it("closes on X button click", () => {
    render(<SettingsModal />);
    fireEvent.click(screen.getByTestId("settings-close"));
    expect(mockStoreState.setSettingsModalOpen).toHaveBeenCalledWith(false);
  });

  it("renders shortcut profiles section", async () => {
    mockListProfiles.mockResolvedValueOnce([mockProfile]);
    render(<SettingsModal />);

    await waitFor(() => {
      expect(screen.getByTestId("shortcut-profile-prof-1")).toBeTruthy();
    });
  });

  it("renders AI provider section", () => {
    render(<SettingsModal />);
    expect(screen.getByTestId("provider-health-panel")).toBeTruthy();
  });
});
