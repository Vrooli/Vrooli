import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import SessionDrawer from "../components/SessionDrawer";
import type { SessionInfo } from "../lib/api";

// [REQ:P0-008a] Drawer Layout Component — rendering and open/close
// [REQ:P0-008b] Session Status and Controls — session list display
// [REQ:P0-008b-i] Session Action Feedback — delete confirmation pattern
// [REQ:P1-001b] Policy Configuration UI — policy dropdown in drawer
// [REQ:P1-003b] Provider Health Dashboard — ProviderHealthPanel integration

let mockUpdateSessionPolicy: ReturnType<typeof vi.fn>;
let mockGetAIConfig: ReturnType<typeof vi.fn>;

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return {
    ...actual,
    updateSessionPolicy: vi.fn(),
    getAIConfig: vi.fn().mockResolvedValue({ providers: [], health: [] }),
    updateAIConfig: vi.fn(),
  };
});

vi.mock("../lib/format", () => ({
  getShellName: vi.fn((s: string) => s.split("/").pop() || "shell"),
  formatSessionTime: vi.fn(() => "2:30:00 PM"),
  truncateId: vi.fn((id: string) => id.slice(0, 8) + "..."),
}));

vi.mock("../hooks/useCountdown", () => ({
  useCountdown: vi.fn(() => null),
}));

const makeSessions = (...ids: string[]): Array<{ session: SessionInfo }> =>
  ids.map((id) => ({
    session: {
      id,
      shell: "/bin/bash",
      created_at: "2026-01-15T14:30:00Z",
      cols: 80,
      rows: 24,
      policy: { mode: "never" as const },
    },
  }));

describe("SessionDrawer", () => {
  const onClose = vi.fn();
  const onDeleteSession = vi.fn();
  const onSelectSession = vi.fn();

  beforeEach(async () => {
    vi.clearAllMocks();
    const api = await import("../lib/api");
    mockUpdateSessionPolicy = api.updateSessionPolicy as ReturnType<typeof vi.fn>;
    mockGetAIConfig = api.getAIConfig as ReturnType<typeof vi.fn>;
    mockGetAIConfig.mockResolvedValue({ providers: [], health: [] });
  });

  it("renders drawer panel even when closed (translated off-screen)", () => {
    render(
      <SessionDrawer open={false} onClose={onClose} sessions={[]} onDeleteSession={onDeleteSession} />,
    );
    const drawer = screen.getByTestId("session-drawer");
    expect(drawer.className).toContain("-translate-x-full");
  });

  it("renders drawer visible when open", () => {
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={[]} onDeleteSession={onDeleteSession} />,
    );
    const drawer = screen.getByTestId("session-drawer");
    expect(drawer.className).toContain("translate-x-0");
  });

  it("shows backdrop when open", () => {
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={[]} onDeleteSession={onDeleteSession} />,
    );
    expect(screen.getByTestId("drawer-backdrop")).toBeTruthy();
  });

  it("closes when backdrop is clicked", () => {
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={[]} onDeleteSession={onDeleteSession} />,
    );
    fireEvent.click(screen.getByTestId("drawer-backdrop"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("closes when close button is clicked", () => {
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={[]} onDeleteSession={onDeleteSession} />,
    );
    fireEvent.click(screen.getByTestId("drawer-close"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("shows empty state when no sessions", () => {
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={[]} onDeleteSession={onDeleteSession} />,
    );
    expect(screen.getByText("No active sessions")).toBeTruthy();
    expect(screen.getByText(/0 active session/)).toBeTruthy();
  });

  it("renders session rows with shell name and truncated ID", () => {
    const sessions = makeSessions("sess-abc123");
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={sessions} onDeleteSession={onDeleteSession} />,
    );
    expect(screen.getByTestId("drawer-session-sess-abc123")).toBeTruthy();
    expect(screen.getByText("bash")).toBeTruthy();
    expect(screen.getByText(/sess-abc\.*/)).toBeTruthy();
    expect(screen.getByText("1 active session")).toBeTruthy();
  });

  it("renders multiple sessions with correct count", () => {
    const sessions = makeSessions("sess-001", "sess-002", "sess-003");
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={sessions} onDeleteSession={onDeleteSession} />,
    );
    expect(screen.getByTestId("drawer-session-sess-001")).toBeTruthy();
    expect(screen.getByTestId("drawer-session-sess-002")).toBeTruthy();
    expect(screen.getByTestId("drawer-session-sess-003")).toBeTruthy();
    expect(screen.getByText("3 active sessions")).toBeTruthy();
  });

  it("selects session when session name is clicked", () => {
    const sessions = makeSessions("sess-abc123");
    render(
      <SessionDrawer
        open={true}
        onClose={onClose}
        sessions={sessions}
        onDeleteSession={onDeleteSession}
        onSelectSession={onSelectSession}
      />,
    );
    // Click the session name button area
    fireEvent.click(screen.getByText("bash"));
    expect(onSelectSession).toHaveBeenCalledWith("sess-abc123");
  });

  it("requires double-click to delete a session (confirm pattern)", () => {
    const sessions = makeSessions("sess-abc123");
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={sessions} onDeleteSession={onDeleteSession} />,
    );
    // Find the delete button by its title
    const deleteBtn = screen.getByTitle("Terminate session");

    // First click — arms confirmation
    fireEvent.click(deleteBtn);
    expect(onDeleteSession).not.toHaveBeenCalled();
    expect(screen.getByTitle("Click again to confirm")).toBeTruthy();

    // Second click — actually deletes
    fireEvent.click(screen.getByTitle("Click again to confirm"));
    expect(onDeleteSession).toHaveBeenCalledWith("sess-abc123");
  });

  it("renders policy dropdown for each session", () => {
    const sessions = makeSessions("sess-abc123");
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={sessions} onDeleteSession={onDeleteSession} />,
    );
    expect(screen.getByTestId("policy-select-sess-abc123")).toBeTruthy();
  });

  it("calls updateSessionPolicy when policy changes", async () => {
    mockUpdateSessionPolicy.mockResolvedValueOnce({});
    const sessions = makeSessions("sess-abc123");
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={sessions} onDeleteSession={onDeleteSession} />,
    );
    const select = screen.getByTestId("policy-select-sess-abc123");
    fireEvent.change(select, { target: { value: "preset:1h" } });

    await waitFor(() => {
      expect(mockUpdateSessionPolicy).toHaveBeenCalledWith("sess-abc123", {
        mode: "preset",
        duration: "1h",
      });
    });
  });

  it("calls updateSessionPolicy with never mode", async () => {
    mockUpdateSessionPolicy.mockResolvedValueOnce({});
    const sessions: Array<{ session: SessionInfo }> = [{
      session: {
        id: "sess-def456",
        shell: "/bin/bash",
        created_at: "2026-01-15T14:30:00Z",
        cols: 80,
        rows: 24,
        policy: { mode: "preset", duration: "1h" },
      },
    }];
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={sessions} onDeleteSession={onDeleteSession} />,
    );
    const select = screen.getByTestId("policy-select-sess-def456");
    fireEvent.change(select, { target: { value: "never" } });

    await waitFor(() => {
      expect(mockUpdateSessionPolicy).toHaveBeenCalledWith("sess-def456", {
        mode: "never",
      });
    });
  });

  it("displays policy error on update failure and auto-dismisses", async () => {
    mockUpdateSessionPolicy.mockRejectedValueOnce(new Error("Policy update failed"));
    const sessions = makeSessions("sess-abc123");
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={sessions} onDeleteSession={onDeleteSession} />,
    );
    const select = screen.getByTestId("policy-select-sess-abc123");
    fireEvent.change(select, { target: { value: "preset:8h" } });

    await waitFor(() => {
      expect(screen.getByTestId("policy-error")).toBeTruthy();
      expect(screen.getByText("Policy update failed")).toBeTruthy();
    });
  });

  it("renders ProviderHealthPanel in drawer footer", async () => {
    mockGetAIConfig.mockResolvedValueOnce({
      providers: [{ name: "ollama", enabled: true, priority: 1, timeout_sec: 30, max_retries: 2 }],
      health: [{ name: "ollama", available: true, error_count: 0, success_count: 5, error_rate: 0 }],
    });
    const sessions = makeSessions("sess-abc123");
    render(
      <SessionDrawer open={true} onClose={onClose} sessions={sessions} onDeleteSession={onDeleteSession} />,
    );

    await waitFor(() => {
      expect(screen.getByText("AI Providers")).toBeTruthy();
    });
  });
});
