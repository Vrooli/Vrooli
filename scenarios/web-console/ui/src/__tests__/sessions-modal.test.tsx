import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import SessionsModal from "../components/SessionsModal";
import type { SessionInfo } from "../lib/api";

let mockUpdateSessionPolicy: ReturnType<typeof vi.fn>;

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return {
    ...actual,
    updateSessionPolicy: vi.fn(),
  };
});

vi.mock("../hooks/useCountdown", () => ({
  useCountdown: vi.fn(() => null),
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

// Mock workspace store
const mockStoreState = {
  sessionsModalOpen: true,
  panes: [] as Array<{ sessionId: string; name: string; headerColor: string }>,
  setSessionsModalOpen: vi.fn(),
  movePaneToIndex: vi.fn(),
  setActivePane: vi.fn(),
  setPaneColor: vi.fn(),
  renamePaneById: vi.fn(),
  resetLayout: vi.fn(),
};

vi.mock("../stores/useWorkspaceStore", () => ({
  useWorkspaceStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector(mockStoreState),
}));

const makeSession = (id: string): SessionInfo => ({
  id,
  shell: "/bin/bash",
  created_at: "2026-01-15T14:30:00Z",
  cols: 80,
  rows: 24,
  policy: { mode: "never" },
});

describe("SessionsModal", () => {
  const onDeleteSession = vi.fn();

  beforeEach(async () => {
    vi.clearAllMocks();
    mockStoreState.sessionsModalOpen = true;
    mockStoreState.panes = [];
    const api = await import("../lib/api");
    mockUpdateSessionPolicy = api.updateSessionPolicy as ReturnType<typeof vi.fn>;
  });

  it("does not render when sessionsModalOpen is false", () => {
    mockStoreState.sessionsModalOpen = false;
    render(<SessionsModal sessions={[]} onDeleteSession={onDeleteSession} />);
    expect(screen.queryByTestId("sessions-modal")).toBeNull();
  });

  it("renders modal when open", () => {
    render(<SessionsModal sessions={[]} onDeleteSession={onDeleteSession} />);
    expect(screen.getByTestId("sessions-modal")).toBeTruthy();
    expect(screen.getByTestId("sessions-backdrop")).toBeTruthy();
  });

  it("closes on backdrop click", () => {
    render(<SessionsModal sessions={[]} onDeleteSession={onDeleteSession} />);
    fireEvent.click(screen.getByTestId("sessions-backdrop"));
    expect(mockStoreState.setSessionsModalOpen).toHaveBeenCalledWith(false);
  });

  it("closes on X button click", () => {
    render(<SessionsModal sessions={[]} onDeleteSession={onDeleteSession} />);
    fireEvent.click(screen.getByTestId("sessions-close"));
    expect(mockStoreState.setSessionsModalOpen).toHaveBeenCalledWith(false);
  });

  it("shows empty state when no panes", () => {
    render(<SessionsModal sessions={[]} onDeleteSession={onDeleteSession} />);
    expect(screen.getByText("No terminals open")).toBeTruthy();
  });

  it("renders pane list when panes exist", () => {
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
      { sessionId: "s2", name: "zsh", headerColor: "#ff7a7a" },
    ];
    const sessions = [{ session: makeSession("s1") }, { session: makeSession("s2") }];
    render(<SessionsModal sessions={sessions} onDeleteSession={onDeleteSession} />);
    expect(screen.getByTestId("sessions-pane-s1")).toBeTruthy();
    expect(screen.getByTestId("sessions-pane-s2")).toBeTruthy();
    expect(screen.getByText("bash")).toBeTruthy();
    expect(screen.getByText("zsh")).toBeTruthy();
  });

  it("disables up button for first pane", () => {
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
      { sessionId: "s2", name: "zsh", headerColor: "transparent" },
    ];
    const sessions = [{ session: makeSession("s1") }, { session: makeSession("s2") }];
    render(<SessionsModal sessions={sessions} onDeleteSession={onDeleteSession} />);
    expect(screen.getByTestId("sessions-pane-up-s1")).toHaveProperty("disabled", true);
  });

  it("disables down button for last pane", () => {
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
      { sessionId: "s2", name: "zsh", headerColor: "transparent" },
    ];
    const sessions = [{ session: makeSession("s1") }, { session: makeSession("s2") }];
    render(<SessionsModal sessions={sessions} onDeleteSession={onDeleteSession} />);
    expect(screen.getByTestId("sessions-pane-down-s2")).toHaveProperty("disabled", true);
  });

  it("calls movePaneToIndex on up button click", () => {
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
      { sessionId: "s2", name: "zsh", headerColor: "transparent" },
    ];
    const sessions = [{ session: makeSession("s1") }, { session: makeSession("s2") }];
    render(<SessionsModal sessions={sessions} onDeleteSession={onDeleteSession} />);
    fireEvent.click(screen.getByTestId("sessions-pane-up-s2"));
    expect(mockStoreState.movePaneToIndex).toHaveBeenCalledWith("s2", 0);
  });

  it("calls movePaneToIndex on down button click", () => {
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
      { sessionId: "s2", name: "zsh", headerColor: "transparent" },
    ];
    const sessions = [{ session: makeSession("s1") }, { session: makeSession("s2") }];
    render(<SessionsModal sessions={sessions} onDeleteSession={onDeleteSession} />);
    fireEvent.click(screen.getByTestId("sessions-pane-down-s1"));
    expect(mockStoreState.movePaneToIndex).toHaveBeenCalledWith("s1", 1);
  });

  it("calls resetLayout on reset button click", () => {
    render(<SessionsModal sessions={[]} onDeleteSession={onDeleteSession} />);
    fireEvent.click(screen.getByTestId("sessions-reset-layout"));
    expect(mockStoreState.resetLayout).toHaveBeenCalledOnce();
  });

  it("focuses pane and closes modal on focus button click", () => {
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
    ];
    const sessions = [{ session: makeSession("s1") }];
    render(<SessionsModal sessions={sessions} onDeleteSession={onDeleteSession} />);
    fireEvent.click(screen.getByTestId("sessions-pane-focus-s1"));
    expect(mockStoreState.setActivePane).toHaveBeenCalledWith("s1");
    expect(mockStoreState.setSessionsModalOpen).toHaveBeenCalledWith(false);
  });

  it("calls onDeleteSession on remove button click", () => {
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
    ];
    const sessions = [{ session: makeSession("s1") }];
    render(<SessionsModal sessions={sessions} onDeleteSession={onDeleteSession} />);
    fireEvent.click(screen.getByTestId("sessions-pane-remove-s1"));
    expect(onDeleteSession).toHaveBeenCalledWith("s1");
  });

  it("renders policy dropdown for each session", () => {
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
    ];
    const sessions = [{ session: makeSession("s1") }];
    render(<SessionsModal sessions={sessions} onDeleteSession={onDeleteSession} />);
    expect(screen.getByTestId("sessions-policy-select-s1")).toBeTruthy();
  });

  it("calls updateSessionPolicy when policy changes", async () => {
    mockUpdateSessionPolicy.mockResolvedValueOnce({});
    mockStoreState.panes = [
      { sessionId: "s1", name: "bash", headerColor: "transparent" },
    ];
    const sessions = [{ session: makeSession("s1") }];
    render(<SessionsModal sessions={sessions} onDeleteSession={onDeleteSession} />);
    const select = screen.getByTestId("sessions-policy-select-s1");
    fireEvent.change(select, { target: { value: "preset:1h" } });

    await waitFor(() => {
      expect(mockUpdateSessionPolicy).toHaveBeenCalledWith("s1", {
        mode: "preset",
        duration: "1h",
      });
    });
  });
});
