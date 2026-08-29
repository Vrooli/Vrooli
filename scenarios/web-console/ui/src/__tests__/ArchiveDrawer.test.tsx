import { renderWithProviders as render } from "../test-utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ArchiveDrawer from "../components/ArchiveDrawer";

const mocks = vi.hoisted(() => ({
  listArchivedSessions: vi.fn(),
  searchArchivedConversations: vi.fn(),
  getConversationRange: vi.fn(),
  deleteSession: vi.fn(),
  reopenSession: vi.fn(),
  listRecoverableSessions: vi.fn(),
  recoverSession: vi.fn(),
  dismissRecoverableSession: vi.fn(),
}));

vi.mock("../api/sessions", () => ({
  listArchivedSessions: mocks.listArchivedSessions,
  deleteSession: mocks.deleteSession,
  reopenSession: mocks.reopenSession,
  listRecoverableSessions: mocks.listRecoverableSessions,
  recoverSession: mocks.recoverSession,
  dismissRecoverableSession: mocks.dismissRecoverableSession,
}));

vi.mock("../api/conversation", () => ({
  searchArchivedConversations: mocks.searchArchivedConversations,
  getConversationRange: mocks.getConversationRange,
}));

vi.mock("../components/MessagesPane", () => ({
  default: (props: { sessionId: string; readOnly?: boolean; focusEventId?: string; focusSequence?: number; onSendToComposer?: (text: string) => void }) => (
    <div
      data-testid="archive-reader-props"
      data-session={props.sessionId}
      data-read-only={String(props.readOnly)}
      data-focus-event={props.focusEventId}
      data-focus-sequence={props.focusSequence}
    >
      <button type="button" data-testid="reader-send" onClick={() => props.onSendToComposer?.("selected archived message")}>send</button>
    </div>
  ),
}));

vi.mock("../components/MessageExportDrawer", () => ({ default: () => null }));

describe("ArchiveDrawer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listRecoverableSessions.mockResolvedValue([]);
    mocks.listArchivedSessions.mockResolvedValue({
      total: 1,
      sessions: [{
        id: "archive-1",
        archived_at: "2026-08-17T12:00:00Z",
        created_at: "2026-08-16T12:00:00Z",
        agent_type: "claude",
        cwd: "/workspace/project",
        pane_name: "receipt signing",
        message_count: 12,
        restore_state: "read_only",
        restore_state_reason: "agent history no longer on disk",
      }],
    });
    mocks.searchArchivedConversations.mockResolvedValue({
      matches: [{ eventId: "event-42", sessionId: "archive-1", sequence: 42, role: "user", createdAt: "2026-08-17T12:10:00Z", excerpt: "receipt signing key" }],
      truncated: false,
      totalMatches: 1,
      distinctSessions: 1,
    });
  });

  it("searches messages, opens the selected transcript read-only, and stages into only the active composer", async () => {
    const onSendToComposer = vi.fn();
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={onSendToComposer} onReopened={vi.fn()} />);

    const input = await screen.findByTestId("archive-search-input");
    fireEvent.change(input, { target: { value: "receipt signing" } });

    await waitFor(() => expect(mocks.searchArchivedConversations).toHaveBeenCalled());
    const reader = await screen.findByTestId("archive-reader-props");
    expect(reader).toHaveAttribute("data-session", "archive-1");
    expect(reader).toHaveAttribute("data-read-only", "true");
    expect(reader).toHaveAttribute("data-focus-event", "event-42");
    expect(reader).toHaveAttribute("data-focus-sequence", "42");

    fireEvent.click(screen.getByTestId("reader-send"));
    expect(onSendToComposer).toHaveBeenCalledWith("selected archived message");
  });

  it("lists all archived sessions before a search and opens one read-only", async () => {
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);

    const row = await screen.findByTestId("archive-session-archive-1");
    expect(row).toHaveTextContent("receipt signing");
    expect(mocks.searchArchivedConversations).not.toHaveBeenCalled();

    fireEvent.click(row);
    const reader = await screen.findByTestId("archive-reader-props");
    expect(reader).toHaveAttribute("data-session", "archive-1");
    expect(reader).toHaveAttribute("data-read-only", "true");
  });

  it("exports and permanently deletes the selected archive behind confirmation", async () => {
    mocks.getConversationRange.mockResolvedValue({ events: [{ id: "event-1" }] });
    mocks.deleteSession.mockResolvedValue(undefined);
    const onClose = vi.fn();
    render(<ArchiveDrawer open onClose={onClose} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);

    fireEvent.click(await screen.findByTestId("archive-session-archive-1"));
    await screen.findByTestId("archive-reader-props");
    fireEvent.click(screen.getByRole("button", { name: "archiveDrawer.export" }));
    await waitFor(() => expect(mocks.getConversationRange).toHaveBeenCalledWith("archive-1", 1, 12));

    fireEvent.click(screen.getByRole("button", { name: "archiveDrawer.delete" }));
    expect(screen.getByTestId("archive-delete-dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("archive-delete-cancel"));
    expect(screen.queryByTestId("archive-delete-dialog")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "archiveDrawer.delete" }));
    fireEvent.click(screen.getByTestId("archive-delete-confirm"));
    await waitFor(() => expect(mocks.deleteSession).toHaveBeenCalledWith("archive-1"));
    expect(onClose).not.toHaveBeenCalled();
  });

  it("honors an archive selected from the sidebar", async () => {
    render(<ArchiveDrawer open initialSessionId="archive-1" onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);

    const reader = await screen.findByTestId("archive-reader-props");
    expect(reader).toHaveAttribute("data-session", "archive-1");
  });

  it("keeps a sidebar selection when crash-recovery sessions also exist", async () => {
    mocks.listArchivedSessions.mockResolvedValue({ total: 2, sessions: [
      {
        id: "archive-1", archived_at: "2026-08-17T12:00:00Z", created_at: "2026-08-16T12:00:00Z",
        agent_type: "claude", pane_name: "receipt signing", message_count: 12, restore_state: "read_only",
      },
      {
        id: "crash-1", archived_at: "2026-08-17T13:00:00Z", created_at: "", agent_type: "codex",
        pane_name: "Crash", message_count: 2, restore_state: "reopenable", awaiting_recovery: true,
      },
    ] });

    render(<ArchiveDrawer open initialSessionId="archive-1" onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);

    const reader = await screen.findByTestId("archive-reader-props");
    expect(reader).toHaveAttribute("data-session", "archive-1");
    expect(screen.getByTestId("archive-orphans-filter")).toHaveAttribute("aria-pressed", "false");
  });

  it("keeps Semantic and non-reopenable restoration disabled with explanations", async () => {
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    fireEvent.change(await screen.findByTestId("archive-search-input"), { target: { value: "receipt" } });

    await screen.findByTestId("archive-reader-props");
    expect(screen.getByRole("button", { name: "archiveDrawer.semanticSearch" })).toBeDisabled();
    const reopen = screen.getByRole("button", { name: "archiveDrawer.reopen" });
    expect(reopen).toBeDisabled();
    expect(reopen).toHaveAttribute("title", "agent history no longer on disk");
  });

  it("retries Reopen with one stable idempotency key and reports the replacement", async () => {
    mocks.listArchivedSessions.mockResolvedValue({
      total: 1,
      sessions: [{
        id: "archive-1", archived_at: "2026-08-17T12:00:00Z", created_at: "2026-08-16T12:00:00Z",
        agent_type: "codex", pane_name: "reopen me", message_count: 3, restore_state: "reopenable",
      }],
    });
    mocks.reopenSession
      .mockRejectedValueOnce(new Error("ambiguous transport failure"))
      .mockResolvedValueOnce({ old_session_id: "archive-1", new_session_id: "live-new" });
    const onReopened = vi.fn();
    const onClose = vi.fn();
    render(<ArchiveDrawer open onClose={onClose} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={onReopened} />);
    fireEvent.change(await screen.findByTestId("archive-search-input"), { target: { value: "receipt" } });
    await screen.findByTestId("archive-reader-props");

    fireEvent.click(screen.getByTestId("archive-reopen"));
    await waitFor(() => expect(mocks.reopenSession).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("ambiguous transport failure"));
    fireEvent.click(screen.getByTestId("archive-reopen"));
    await waitFor(() => expect(mocks.reopenSession).toHaveBeenCalledTimes(2));

    expect(mocks.reopenSession.mock.calls[0]?.[1]).toBe(mocks.reopenSession.mock.calls[1]?.[1]);
    expect(onReopened).toHaveBeenCalledWith(expect.objectContaining({ new_session_id: "live-new" }));
    expect(onClose).toHaveBeenCalled();
  });

  it("defaults to crash orphans and preserves recovery disabled reasons", async () => {
    mocks.listArchivedSessions.mockResolvedValue({ total: 2, sessions: [
      { id: "codex-orphan", archived_at: "2026-08-17T12:00:00Z", created_at: "", agent_type: "codex", pane_name: "Codex", message_count: 2, restore_state: "reopenable", awaiting_recovery: true },
      { id: "grok-orphan", archived_at: "2026-08-17T11:00:00Z", created_at: "", agent_type: "grok", pane_name: "Grok", message_count: 0, restore_state: "nothing_to_restore", awaiting_recovery: true },
    ] });
    mocks.listRecoverableSessions.mockResolvedValue([
      { id: "codex-orphan", agent_type: "codex", recoverable: true },
      { id: "grok-orphan", agent_type: "grok", recoverable: false, not_recoverable_reason: "grok session id is required" },
    ]);

    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);

    expect(await screen.findByTestId("archive-orphan-results")).toBeInTheDocument();
    expect(screen.getByTestId("archive-orphans-filter")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("recoverable-row-codex-orphan-recover")).not.toBeDisabled();
    expect(screen.getByTestId("recoverable-row-grok-orphan-recover")).toBeDisabled();
    expect(screen.getByTestId("recoverable-row-grok-orphan-recover")).toHaveAttribute("title", "grok session id is required");
  });

  it("retries Reattach with one stable idempotency key and reports the replacement", async () => {
    mocks.listArchivedSessions.mockResolvedValue({ total: 1, sessions: [
      { id: "crash-1", archived_at: "2026-08-17T12:00:00Z", created_at: "", agent_type: "opencode", pane_name: "Crash", message_count: 2, restore_state: "reopenable", awaiting_recovery: true },
    ] });
    mocks.listRecoverableSessions.mockResolvedValue([{ id: "crash-1", agent_type: "opencode", agent_session_id: "ses_abc", recoverable: true }]);
    mocks.recoverSession.mockRejectedValueOnce(new Error("ambiguous transport failure")).mockResolvedValueOnce({ old_session_id: "crash-1", new_session_id: "replacement" });
    const onReopened = vi.fn();
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={onReopened} />);

    const button = await screen.findByTestId("recoverable-row-crash-1-recover");
    fireEvent.click(button);
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("ambiguous transport failure"));
    fireEvent.click(button);
    await waitFor(() => expect(mocks.recoverSession).toHaveBeenCalledTimes(2));
    expect(mocks.recoverSession.mock.calls[0]?.[1]).toBe(mocks.recoverSession.mock.calls[1]?.[1]);
    expect(onReopened).toHaveBeenCalledWith(expect.objectContaining({ new_session_id: "replacement" }));
  });

  it("dismisses an orphan and continues reattach-all after an individual failure", async () => {
    const archived = ["first", "second"].map((id) => ({ id, archived_at: "2026-08-17T12:00:00Z", created_at: "", agent_type: "claude" as const, pane_name: id, message_count: 1, restore_state: "reopenable" as const, awaiting_recovery: true }));
    mocks.listArchivedSessions.mockResolvedValue({ total: 2, sessions: archived });
    mocks.listRecoverableSessions.mockResolvedValue(archived.map(({ id }) => ({ id, agent_type: "claude", recoverable: true })));
    mocks.recoverSession.mockRejectedValueOnce(new Error("first failed")).mockResolvedValueOnce({ old_session_id: "second", new_session_id: "replacement" });
    mocks.dismissRecoverableSession.mockResolvedValue(undefined);
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);

    await screen.findByTestId("recoverable-row-first");
    fireEvent.click(screen.getByRole("button", { name: "recoverableSessions.reattachAll" }));
    await waitFor(() => expect(mocks.recoverSession).toHaveBeenCalledTimes(2));
    expect(mocks.recoverSession.mock.calls.map(([id]) => id)).toEqual(["first", "second"]);

    fireEvent.click(screen.getByTestId("recoverable-row-first-dismiss"));
    await waitFor(() => expect(mocks.dismissRecoverableSession).toHaveBeenCalledWith("first"));
  });

  it("keeps a large recovery set in the archive drawer", async () => {
    const archived = Array.from({ length: 20 }, (_, index) => ({
      id: `session-${index}`, archived_at: "2026-08-17T12:00:00Z", created_at: "",
      agent_type: "claude" as const, pane_name: `Session ${index}`, message_count: 1,
      restore_state: "reopenable" as const, awaiting_recovery: true,
    }));
    mocks.listArchivedSessions.mockResolvedValue({ total: archived.length, sessions: archived });
    mocks.listRecoverableSessions.mockResolvedValue(archived.map(({ id }) => ({ id, agent_type: "claude", recoverable: true })));

    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);

    await screen.findByTestId("archive-orphan-results");
    expect(screen.getAllByTestId(/recoverable-row-session-\d+$/)).toHaveLength(20);
  });
});

describe("ArchiveDrawer swipe actions", () => {
  const REOPENABLE = {
    id: "archive-2",
    archived_at: "2026-08-17T12:00:00Z",
    created_at: "2026-08-16T12:00:00Z",
    agent_type: "claude",
    cwd: "/workspace/project",
    pane_name: "release planning",
    message_count: 4,
    restore_state: "reopenable" as const,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listRecoverableSessions.mockResolvedValue([]);
    // The gesture is touch-only, and jsdom reports no matches by default.
    window.matchMedia = ((query: string) => ({
      matches: query.includes("max-width: 767px"),
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    })) as unknown as typeof window.matchMedia;
  });

  const renderDrawer = () =>
    render(
      <ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />,
    );

  it("offers a reopen gesture on a reopenable row", async () => {
    mocks.listArchivedSessions.mockResolvedValue({ total: 1, sessions: [REOPENABLE] });
    renderDrawer();
    expect(await screen.findByTestId("archive-swipe-archive-2")).toBeInTheDocument();
  });

  // A row that cannot be reopened gets no gesture rather than a disabled one,
  // so the affordance never promises something it cannot do.
  it("offers no gesture on a row that cannot be reopened", async () => {
    mocks.listArchivedSessions.mockResolvedValue({
      total: 1,
      sessions: [{ ...REOPENABLE, id: "archive-3", restore_state: "read_only" as const }],
    });
    renderDrawer();
    await screen.findByTestId("archive-session-archive-3");
    expect(screen.queryByTestId("archive-swipe-archive-3")).toBeNull();
  });

  it("reopens the row the gesture was performed on, not the selected one", async () => {
    mocks.listArchivedSessions.mockResolvedValue({ total: 1, sessions: [REOPENABLE] });
    mocks.reopenSession.mockResolvedValue({ id: "archive-2" });
    renderDrawer();
    await screen.findByTestId("archive-swipe-archive-2");

    const face = screen.getByTestId("archive-swipe-archive-2.face");
    fireEvent.pointerDown(face, {
      pointerId: 1, clientX: 0, clientY: 0, button: 0, pointerType: "touch", timeStamp: 1,
    });
    fireEvent.pointerMove(window, {
      pointerId: 1, clientX: 200, clientY: 0, pointerType: "touch", timeStamp: 300,
    });
    fireEvent.pointerUp(window, {
      pointerId: 1, clientX: 200, clientY: 0, pointerType: "touch", timeStamp: 400,
    });

    fireEvent.click(screen.getByTestId("archive-swipe-archive-2.action.reopen"));
    // The row the gesture ran on, not whichever row happens to be selected.
    await waitFor(() => {
      expect(mocks.reopenSession).toHaveBeenCalledWith("archive-2", expect.any(String));
    });
  });
});
