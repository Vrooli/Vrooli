import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import SessionsPage from "../pages/SessionsPage";
import type { SessionInfo } from "../lib/api";

// [REQ:P0-003a] Session List Management — component rendering & interactions
// [REQ:P1-001a] Session Policy Overview — policy dropdown behavior

const mockSession: SessionInfo = {
  id: "sess-abc123",
  shell: "/bin/bash",
  created_at: "2026-01-15T14:30:00Z",
  cols: 80,
  rows: 24,
  policy: { mode: "never" },
};

const mockSessionWithPolicy: SessionInfo = {
  ...mockSession,
  id: "sess-def456",
  policy: { mode: "preset", duration: "1h" },
};

let mockListSessions: ReturnType<typeof vi.fn>;
let mockDeleteSession: ReturnType<typeof vi.fn>;
let mockUpdateSessionPolicy: ReturnType<typeof vi.fn>;

vi.mock("../lib/api", () => ({
  listSessions: vi.fn(),
  deleteSession: vi.fn(),
  updateSessionPolicy: vi.fn(),
  getShellName: vi.fn((s: string) => s.split("/").pop() || "shell"),
}));

vi.mock("../lib/format", () => ({
  getShellName: vi.fn((s: string) => s.split("/").pop() || "shell"),
  formatSessionTime: vi.fn(() => "2:30:00 PM"),
  truncateId: vi.fn((id: string) => id.slice(0, 8) + "..."),
}));

vi.mock("../hooks/useCountdown", () => ({
  useCountdown: vi.fn(() => null),
}));

describe("SessionsPage", () => {
  const onBack = vi.fn();

  beforeEach(async () => {
    vi.clearAllMocks();
    const api = await import("../lib/api");
    mockListSessions = api.listSessions as ReturnType<typeof vi.fn>;
    mockDeleteSession = api.deleteSession as ReturnType<typeof vi.fn>;
    mockUpdateSessionPolicy = api.updateSessionPolicy as ReturnType<typeof vi.fn>;
  });

  it("renders loading state initially", () => {
    mockListSessions.mockReturnValue(new Promise(() => {})); // never resolves
    render(<SessionsPage onBack={onBack} />);

    expect(screen.getByText("Loading sessions...")).toBeTruthy();
  });

  it("renders empty state when no sessions exist", async () => {
    mockListSessions.mockResolvedValueOnce([]);
    render(<SessionsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByText("No active sessions")).toBeTruthy();
    });
  });

  it("renders session rows after loading", async () => {
    mockListSessions.mockResolvedValueOnce([mockSession]);
    render(<SessionsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("session-row-sess-abc123")).toBeTruthy();
    });

    // Shell name and truncated ID displayed
    expect(screen.getByText("bash")).toBeTruthy();
    expect(screen.getByText("sess-abc...")).toBeTruthy();
    // Dimensions displayed
    expect(screen.getByText("80×24")).toBeTruthy();
    // Session count in header
    expect(screen.getByText("(1)")).toBeTruthy();
  });

  it("calls onBack when back button is clicked", async () => {
    mockListSessions.mockResolvedValueOnce([]);
    render(<SessionsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByText("No active sessions")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("sessions-back"));
    expect(onBack).toHaveBeenCalledOnce();
  });

  it("requires double-click to delete a session (confirm pattern)", async () => {
    mockListSessions.mockResolvedValueOnce([mockSession]);
    mockDeleteSession.mockResolvedValueOnce(undefined);

    render(<SessionsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("session-row-sess-abc123")).toBeTruthy();
    });

    const deleteBtn = screen.getByTestId("session-delete-sess-abc123");

    // First click — arms the confirmation
    fireEvent.click(deleteBtn);
    expect(mockDeleteSession).not.toHaveBeenCalled();

    // Second click — actually deletes
    fireEvent.click(deleteBtn);
    expect(mockDeleteSession).toHaveBeenCalledWith("sess-abc123");
  });

  it("removes session from list after successful delete", async () => {
    mockListSessions.mockResolvedValueOnce([mockSession, mockSessionWithPolicy]);
    mockDeleteSession.mockResolvedValueOnce(undefined);

    render(<SessionsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("session-row-sess-abc123")).toBeTruthy();
      expect(screen.getByTestId("session-row-sess-def456")).toBeTruthy();
    });

    const deleteBtn = screen.getByTestId("session-delete-sess-abc123");

    // Double-click to delete
    fireEvent.click(deleteBtn);
    fireEvent.click(deleteBtn);

    await waitFor(() => {
      expect(screen.queryByTestId("session-row-sess-abc123")).toBeNull();
    });

    // Other session still visible
    expect(screen.getByTestId("session-row-sess-def456")).toBeTruthy();
  });

  it("calls updateSessionPolicy when policy dropdown changes", async () => {
    mockListSessions
      .mockResolvedValueOnce([mockSession])
      .mockResolvedValueOnce([mockSession]); // refresh after policy update
    mockUpdateSessionPolicy.mockResolvedValueOnce({});

    render(<SessionsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("session-policy-sess-abc123")).toBeTruthy();
    });

    const select = screen.getByTestId("session-policy-sess-abc123") as HTMLSelectElement;

    fireEvent.change(select, { target: { value: "preset:1h" } });

    expect(mockUpdateSessionPolicy).toHaveBeenCalledWith("sess-abc123", {
      mode: "preset",
      duration: "1h",
    });
  });

  it("calls updateSessionPolicy with 'never' mode", async () => {
    mockListSessions
      .mockResolvedValueOnce([mockSessionWithPolicy])
      .mockResolvedValueOnce([mockSessionWithPolicy]);
    mockUpdateSessionPolicy.mockResolvedValueOnce({});

    render(<SessionsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByTestId("session-policy-sess-def456")).toBeTruthy();
    });

    const select = screen.getByTestId("session-policy-sess-def456") as HTMLSelectElement;

    fireEvent.change(select, { target: { value: "never" } });

    expect(mockUpdateSessionPolicy).toHaveBeenCalledWith("sess-def456", {
      mode: "never",
    });
  });

  it("refresh button reloads sessions", async () => {
    mockListSessions
      .mockResolvedValueOnce([mockSession])
      .mockResolvedValueOnce([mockSession, mockSessionWithPolicy]);

    render(<SessionsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByText("(1)")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("sessions-refresh"));

    await waitFor(() => {
      expect(screen.getByText("(2)")).toBeTruthy();
    });

    expect(mockListSessions).toHaveBeenCalledTimes(2);
  });

  it("shows empty state on fetch error", async () => {
    mockListSessions.mockRejectedValueOnce(new Error("network error"));
    render(<SessionsPage onBack={onBack} />);

    await waitFor(() => {
      expect(screen.getByText("No active sessions")).toBeTruthy();
    });
  });
});
