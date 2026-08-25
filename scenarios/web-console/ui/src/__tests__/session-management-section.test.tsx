import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import SessionManagementSection from "../components/settings/SessionManagementSection";
import { strings } from "../consts/strings";
import type { SessionInfo } from "../api/sessions";

let mockUpdateSessionPolicy: ReturnType<typeof vi.fn>;
let mockGetArchiveRetention: ReturnType<typeof vi.fn>;

vi.mock("../api/sessions", async () => {
  const actual = await vi.importActual<typeof import("../api/sessions")>("../api/sessions");
  return {
    ...actual,
    updateSessionPolicy: vi.fn(),
    getArchiveRetention: vi.fn(),
  };
});

vi.mock("../hooks/useCountdown", () => ({
  useCountdown: vi.fn(() => null),
}));

vi.mock("../hooks/useWorkspaceSync", () => ({
  useWorkspaceSync: () => ({
    syncActivePane: vi.fn(),
    syncPaneOrder: vi.fn(),
    syncPaneUpdate: vi.fn(),
    syncCreateGroup: vi.fn(),
    syncUpdateGroup: vi.fn(),
    syncDeleteGroup: vi.fn(),
  }),
}));

const mockStoreState = {
  panes: [] as Array<{ sessionId: string; name: string; headerColor: string }>,
  movePaneToIndex: vi.fn(),
  setActivePane: vi.fn(),
  setPaneColor: vi.fn(),
  renamePaneById: vi.fn(),
  resetLayout: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) => selector(mockStoreState),
}));

const makeSession = (id: string): SessionInfo => ({
  id,
  shell: "/bin/bash",
  created_at: "2026-01-15T14:30:00Z",
  cols: 80,
  rows: 24,
  backend: "standard",
  survives_restart: false,
  policy: { mode: "never" },
  origin: "ui",
  owner: "",
  display_label: "",
});

describe("SessionManagementSection", () => {
  const onDeleteSession = vi.fn();
  const onRequestClose = vi.fn();

  beforeEach(async () => {
    vi.clearAllMocks();
    mockStoreState.panes = [];
    const api = await import("../api/sessions");
    mockUpdateSessionPolicy = api.updateSessionPolicy as ReturnType<typeof vi.fn>;
    mockGetArchiveRetention = api.getArchiveRetention as ReturnType<typeof vi.fn>;
    mockGetArchiveRetention.mockResolvedValue({
      policy: { message_less_age_days: 0, agent_home_age_days: 0, max_bytes: 0 },
      stats: { entry_count: 0, message_count: 0, transcript_bytes: 0, agent_home_bytes: 0, total_bytes: 0 },
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows empty state when no panes are open", () => {
    render(<SessionManagementSection sessions={[]} onDeleteSession={onDeleteSession} onRequestClose={onRequestClose} />);
    expect(screen.getByText(strings.settings.sessionsSection.noTerminalsOpen)).toBeTruthy();
  });

  it("shows measured archive entry count and total storage", async () => {
    mockGetArchiveRetention.mockResolvedValueOnce({
      policy: { message_less_age_days: 7, agent_home_age_days: 30, max_bytes: 0 },
      stats: { entry_count: 12, message_count: 300, transcript_bytes: 1024, agent_home_bytes: 2048, total_bytes: 3072 },
    });
    render(<SessionManagementSection sessions={[]} onDeleteSession={onDeleteSession} onRequestClose={onRequestClose} />);
    await waitFor(() => {
      expect(screen.getByTestId("archive-storage-summary").getAttribute("data-entry-count")).toBe("12");
      expect(screen.getByTestId("archive-storage-summary").getAttribute("data-total-bytes")).toBe("3072");
    });
  });

  it("renders pane list when panes exist", () => {
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
      { sessionId: "s2", name: "zsh", headerColor: "#ff7a7a" },
    ];
    const sessions = [{ session: makeSession("s1") }, { session: makeSession("s2") }];
    render(<SessionManagementSection sessions={sessions} onDeleteSession={onDeleteSession} onRequestClose={onRequestClose} />);
    expect(screen.getByTestId("sessions-pane-s1")).toBeTruthy();
    expect(screen.getByTestId("sessions-pane-s2")).toBeTruthy();
  });

  it("moves panes up and down", () => {
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
      { sessionId: "s2", name: "zsh", headerColor: "transparent" },
    ];
    const sessions = [{ session: makeSession("s1") }, { session: makeSession("s2") }];
    render(<SessionManagementSection sessions={sessions} onDeleteSession={onDeleteSession} onRequestClose={onRequestClose} />);
    fireEvent.click(screen.getByTestId("sessions-pane-up-s2"));
    fireEvent.click(screen.getByTestId("sessions-pane-down-s1"));
    expect(mockStoreState.movePaneToIndex).toHaveBeenNthCalledWith(1, "s2", 0);
    expect(mockStoreState.movePaneToIndex).toHaveBeenNthCalledWith(2, "s1", 1);
  });

  it("focuses pane and requests close", () => {
    mockStoreState.panes = [{ sessionId: "s1", name: "bash", headerColor: "transparent" }];
    const sessions = [{ session: makeSession("s1") }];
    render(<SessionManagementSection sessions={sessions} onDeleteSession={onDeleteSession} onRequestClose={onRequestClose} />);
    fireEvent.click(screen.getByTestId("sessions-pane-focus-s1"));
    expect(mockStoreState.setActivePane).toHaveBeenCalledWith("s1");
    expect(onRequestClose).toHaveBeenCalledOnce();
  });

  it("calls onDeleteSession when remove button is clicked", () => {
    mockStoreState.panes = [{ sessionId: "s1", name: "bash", headerColor: "transparent" }];
    const sessions = [{ session: makeSession("s1") }];
    render(<SessionManagementSection sessions={sessions} onDeleteSession={onDeleteSession} onRequestClose={onRequestClose} />);
    fireEvent.click(screen.getByTestId("sessions-pane-remove-s1"));
    expect(onDeleteSession).toHaveBeenCalledWith("s1");
  });

  it("calls updateSessionPolicy when policy changes", async () => {
    mockUpdateSessionPolicy.mockResolvedValueOnce({});
    mockStoreState.panes = [{ sessionId: "s1", name: "bash", headerColor: "transparent" }];
    const sessions = [{ session: makeSession("s1") }];
    render(<SessionManagementSection sessions={sessions} onDeleteSession={onDeleteSession} onRequestClose={onRequestClose} />);
    fireEvent.change(screen.getByTestId("sessions-policy-select-s1"), { target: { value: "preset:1h" } });
    await waitFor(() => {
      expect(mockUpdateSessionPolicy).toHaveBeenCalledWith("s1", {
        mode: "preset",
        duration: "1h",
      });
    });
  });
});
