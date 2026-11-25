import { renderWithProviders as render } from "../test-utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ArchiveDrawer from "../components/ArchiveDrawer";

// The drawer's footer shows its primary action and one more; the rest live in the
// action bar's "More" menu, whose entries are menu items.
function chooseMoreAction(name: string) {
  fireEvent.click(screen.getByRole("button", { name: /^More/ }));
  fireEvent.click(screen.getByRole("menuitem", { name }));
}

const mocks = vi.hoisted(() => ({
  listArchivedSessions: vi.fn(),
  searchArchivedConversations: vi.fn(),
  searchSemanticConversations: vi.fn(),
  searchNodeArchive: vi.fn(),
  getConversationRange: vi.fn(),
  deleteSession: vi.fn(),
  reopenSession: vi.fn(),
  listRecoverableSessions: vi.fn(),
  recoverSession: vi.fn(),
  dismissRecoverableSession: vi.fn(),
  getContinuityIntegrity: vi.fn(),
  searchLocalContinuity: vi.fn(),
  getContinuityReceipt: vi.fn(),
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
  searchSemanticConversations: mocks.searchSemanticConversations,
  searchNodeArchive: mocks.searchNodeArchive,
  getConversationRange: mocks.getConversationRange,
}));

vi.mock("../api/continuity", () => ({
  getContinuityIntegrity: mocks.getContinuityIntegrity,
  searchLocalContinuity: mocks.searchLocalContinuity,
  getContinuityReceipt: mocks.getContinuityReceipt,
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
    mocks.getContinuityIntegrity.mockResolvedValue({ orphanConversations: 0, orphanCheckpoints: 0, orphanWorkspacePanes: 0, uncatalogedConversations: 0 });
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
    mocks.searchSemanticConversations.mockResolvedValue({
      local: { matches: [], truncated: false, totalMatches: 0, distinctSessions: 0 },
      semanticAvailable: true, semanticError: "", semanticTookMs: 7,
      semanticHits: [{ stableHitId: "semantic-1", eventId: "am-event-1", role: "assistant", occurredAt: "2026-08-17T12:10:00Z", snippet: "semantic-only phrase", harness: "codex", sourceSessionId: "am-session-1", weak: false }],
      coverage: { semanticRatio: 0.94 }, degradations: [],
    });
    mocks.searchNodeArchive.mockResolvedValue({
      nodeId: "node-lab", available: true, degradation: "",
      search: { matches: [{ eventId: "node-event-1", sessionId: "node-session-1", sequence: 8, role: "user", createdAt: "2026-08-17T12:10:00Z", occurredAt: "2026-08-10T08:30:00Z", excerpt: "node-only phrase" }], truncated: false, totalMatches: 1, distinctSessions: 1 },
    });
    mocks.searchLocalContinuity.mockResolvedValue({
      matches: [{ eventId: "event-42", sessionId: "archive-1", sequence: 42, role: "user", createdAt: "2026-08-17T12:10:00Z", excerpt: "receipt signing key", lifecycleState: "archived" }],
      truncated: false,
      totalMatches: 1,
      distinctSessions: 1,
    });
    mocks.getContinuityReceipt.mockResolvedValue({
      operationId: "archive:archive-1", sessionId: "archive-1", command: "archive",
      fromState: "live", toState: "archived", status: "succeeded", errorCode: "",
      createdAt: "2026-08-17T12:00:00Z",
    });
  });

  it("surfaces integrity drift without hiding the archive", async () => {
    mocks.getContinuityIntegrity.mockResolvedValue({ orphanConversations: 2, orphanCheckpoints: 1, orphanWorkspacePanes: 0, uncatalogedConversations: 0 });
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);

    expect(await screen.findByTestId("archive-continuity-warning")).toHaveTextContent("archiveDrawer.continuityWarning");
    expect(screen.getByTestId("archive-session-archive-1")).toBeInTheDocument();
  });

  it("surfaces catalog-only continuity drift", async () => {
    mocks.getContinuityIntegrity.mockResolvedValue({ orphanConversations: 0, orphanCheckpoints: 0, orphanWorkspacePanes: 0, uncatalogedConversations: 2 });
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);

    expect(await screen.findByTestId("archive-continuity-warning")).toHaveTextContent("archiveDrawer.continuityWarning");
  });

  it("shows archive search results when entered from the recovery-only drawer", async () => {
    mocks.listRecoverableSessions.mockResolvedValue([{
      id: "recovery-fixture", recoverable: false, not_recoverable_reason: "synthetic navigation fixture",
    }]);
    mocks.searchLocalContinuity.mockResolvedValue({
      matches: [{ eventId: "antigravity-event", sessionId: "wc-archive-antigravity-fixture-01", sequence: 2, role: "assistant", createdAt: "2026-11-23T15:01:00Z", occurredAt: "2026-11-23T14:58:00Z", excerpt: "synthetic native archive marker", lifecycleState: "archived" }],
      truncated: false,
      totalMatches: 1,
      distinctSessions: 1,
    });
    render(<ArchiveDrawer open preferOrphans onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);

    await screen.findByTestId("archive-orphan-results");
    fireEvent.change(screen.getByTestId("archive-search-input"), { target: { value: "synthetic native archive marker" } });

    expect(await screen.findByTestId("archive-hit-antigravity-event")).toBeInTheDocument();
  });

  it("searches messages, opens the selected transcript read-only, and stages into only the active composer", async () => {
    const onSendToComposer = vi.fn();
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={onSendToComposer} onReopened={vi.fn()} />);

    const input = await screen.findByTestId("archive-search-input");
    fireEvent.change(input, { target: { value: "receipt signing" } });

    await waitFor(() => expect(mocks.searchLocalContinuity).toHaveBeenCalled());
    const reader = await screen.findByTestId("archive-reader-props");
    expect(reader).toHaveAttribute("data-session", "archive-1");
    expect(reader).toHaveAttribute("data-read-only", "true");
    expect(reader).toHaveAttribute("data-focus-event", "event-42");
    expect(reader).toHaveAttribute("data-focus-sequence", "42");

    fireEvent.click(screen.getByTestId("reader-send"));
    expect(onSendToComposer).toHaveBeenCalledWith("selected archived message");
  });

  it("offers an explicit all-lifecycle-state search and keeps live results non-destructive", async () => {
    mocks.searchLocalContinuity.mockResolvedValue({
      matches: [{ eventId: "live-event", sessionId: "live-session", sequence: 1, role: "user", createdAt: "2026-08-17T12:10:00Z", excerpt: "live continuity", lifecycleState: "live" }],
      truncated: false,
      totalMatches: 1,
      distinctSessions: 1,
    });
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("archive-all-states"));
    fireEvent.change(screen.getByTestId("archive-search-input"), { target: { value: "live continuity" } });

    await waitFor(() => expect(mocks.searchLocalContinuity).toHaveBeenCalled());
    const reader = await screen.findByTestId("archive-reader-props");
    expect(reader).toHaveAttribute("data-session", "live-session");
    expect(screen.queryByRole("button", { name: "archiveDrawer.delete" })).toBeNull();
  });

  it("shows catalog-only orphan matches in the normal archive search", async () => {
    mocks.searchLocalContinuity.mockResolvedValue({
      matches: [{ eventId: "orphan-event", sessionId: "orphan-pane", sequence: 88, role: "assistant", createdAt: "2026-08-17T12:10:00Z", excerpt: "recovered plan-manager discussion", lifecycleState: "archived" }],
      truncated: false,
      totalMatches: 1,
      distinctSessions: 1,
    });
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    fireEvent.change(await screen.findByTestId("archive-search-input"), { target: { value: "plan-manager" } });

    const reader = await screen.findByTestId("archive-reader-props");
    expect(reader).toHaveAttribute("data-session", "orphan-pane");
    expect(reader).toHaveAttribute("data-read-only", "true");
  });

  it("opens native archive results with truthful source metadata and no invented receipt", async () => {
    mocks.searchLocalContinuity.mockResolvedValue({
      matches: [{ eventId: "cursor-event", sessionId: "wc-archive-cursor-workspace-123", sequence: 9, role: "assistant", createdAt: "2026-08-17T12:10:00Z", excerpt: "cursor native conversation", lifecycleState: "archived" }],
      truncated: false,
      totalMatches: 1,
      distinctSessions: 1,
    });
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    fireEvent.change(await screen.findByTestId("archive-search-input"), { target: { value: "cursor native" } });

    fireEvent.click(await screen.findByTestId("archive-hit-cursor-event"));
    const reader = await screen.findByTestId("archive-reader-props");
    expect(reader).toHaveAttribute("data-session", "wc-archive-cursor-workspace-123");
    expect(reader).toHaveAttribute("data-read-only", "true");
    expect(screen.getByTestId("native-archive-heading")).toHaveTextContent("Cursor · Native archive");
    expect(screen.getAllByText(/Indexed ·/).length).toBeGreaterThan(0);
    expect(screen.queryByText(/Message ·/)).toBeNull();
    expect(screen.queryByText(/messageCount/)).toBeNull();
    expect(screen.queryByRole("button", { name: "archiveDrawer.receipt" })).toBeNull();
    expect(mocks.getContinuityReceipt).not.toHaveBeenCalled();
  });

  it("labels Qwen Code native archive results", async () => {
    mocks.searchLocalContinuity.mockResolvedValue({
      matches: [{ eventId: "qwen-event", sessionId: "wc-archive-qwen-code-session-123", sequence: 2, role: "assistant", createdAt: "2026-08-17T12:10:00Z", occurredAt: "2026-08-16T09:00:00Z", excerpt: "qwen native conversation", lifecycleState: "archived" }],
      truncated: false,
      totalMatches: 1,
      distinctSessions: 1,
    });
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    fireEvent.change(await screen.findByTestId("archive-search-input"), { target: { value: "qwen native" } });
    fireEvent.click(await screen.findByTestId("archive-hit-qwen-event"));

    expect(await screen.findByTestId("native-archive-heading")).toHaveTextContent("Qwen Code · Native archive");
    expect(screen.getAllByText(/Message ·/).length).toBeGreaterThan(0);
  });

  it("runs optional semantic search, keeps local results, and previews Agent Manager provenance", async () => {
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    fireEvent.change(await screen.findByTestId("archive-search-input"), { target: { value: "semantic-only" } });
    fireEvent.click(screen.getByTestId("archive-search-mode-semantic"));

    const hit = await screen.findByTestId("archive-remote-hit-semantic-semantic-1");
    expect(mocks.searchSemanticConversations).toHaveBeenCalledWith("semantic-only", { agentType: undefined, role: undefined }, 100);
    expect(await screen.findByTestId("archive-search-notice")).toHaveTextContent("archiveDrawer.semanticCoverage");
    fireEvent.click(hit);
    expect(await screen.findByTestId("archive-remote-preview")).toHaveTextContent("am-session-1");
    expect(screen.getByTestId("archive-remote-preview")).toHaveTextContent("semantic-only phrase");
    expect(screen.queryByTestId("archive-reader-props")).toBeNull();
  });

  it("requires an explicit node ID and previews only results from that selected node", async () => {
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("archive-search-mode-node"));
    fireEvent.change(screen.getByTestId("archive-search-input"), { target: { value: "node-only" } });
    expect(await screen.findByTestId("archive-search-notice")).toHaveTextContent("archiveDrawer.nodeIdRequired");
    expect(mocks.searchNodeArchive).not.toHaveBeenCalled();
    expect(screen.queryByTestId("archive-hit-event-42")).toBeNull();
    expect(screen.getByTestId("archive-time-filter")).toBeDisabled();

    fireEvent.change(screen.getByTestId("archive-node-id"), { target: { value: "node-lab" } });
    const hit = await screen.findByTestId("archive-remote-hit-node-node-event-1");
    expect(mocks.searchNodeArchive).toHaveBeenCalledWith("node-lab", "node-only", { agentType: undefined, role: undefined }, 100);
    expect(hit).toHaveTextContent("Message ·");
    fireEvent.click(hit);
    expect(await screen.findByTestId("archive-remote-preview")).toHaveAttribute("data-node-id", "node-lab");
    expect(screen.getByTestId("archive-remote-preview")).toHaveTextContent("node-only phrase");
    expect(screen.queryByTestId("archive-reader-props")).toBeNull();
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

    chooseMoreAction("archiveDrawer.delete");
    expect(screen.getByTestId("archive-delete-dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("archive-delete-cancel"));
    await waitFor(() => expect(screen.queryByTestId("archive-delete-dialog")).toBeNull());
    chooseMoreAction("archiveDrawer.delete");
    fireEvent.click(screen.getByTestId("archive-delete-confirm"));
    await waitFor(() => expect(mocks.deleteSession).toHaveBeenCalledWith("archive-1"));
    expect(onClose).not.toHaveBeenCalled();
  });

  it("keeps the action bar within the component contract and leaves closing to the header", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const onClose = vi.fn();
    render(<ArchiveDrawer open onClose={onClose} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("archive-session-archive-1"));
    fireEvent.click(screen.getByRole("button", { name: /^More/ }));
    expect(screen.getAllByRole("menuitem").map((item) => item.textContent)).toEqual(["archiveDrawer.receipt", "archiveDrawer.delete"]);
    expect(warn.mock.calls.filter(([message]) => String(message).includes("OverlayActionBar"))).toEqual([]);
    warn.mockRestore();
  });

  it("keeps the German phone showcase concise, one-row, and warning-free", async () => {
    window.history.replaceState({}, "", "/?rcl-contract-showcase=1");
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const onClose = vi.fn();
    try {
      render(<ArchiveDrawer open onClose={onClose} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
      fireEvent.click(await screen.findByTestId("archive-session-archive-1"));

      const bar = document.querySelector<HTMLElement>("[data-rcl-overlay-action-bar]");
      expect(bar).not.toBeNull();
      expect(bar).toHaveStyle({ flexWrap: "nowrap" });
      expect(bar?.querySelectorAll(":scope > button")).toHaveLength(2);
      expect(screen.getByRole("button", { name: /^More/ })).toBeInTheDocument();
      expect(screen.getByTestId("archive-reopen")).toHaveAccessibleName("Sitzung wiederherstellen");
      expect(warn.mock.calls.filter(([message]) => String(message).includes("OverlayActionBar"))).toEqual([]);

      fireEvent.click(screen.getByRole("button", { name: /^More/ }));
      expect(screen.getAllByRole("menuitem").map((item) => item.textContent)).toEqual([
        "Beleg anzeigen",
        "Sitzung dauerhaft löschen",
      ]);
      fireEvent.click(screen.getByTestId("archive-drawer.grabber"));
      expect(onClose).toHaveBeenCalledTimes(1);
    } finally {
      warn.mockRestore();
      window.history.replaceState({}, "", "/");
    }
  });

  it("inspects the durable archive receipt without leaving the drawer", async () => {
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    fireEvent.click(await screen.findByTestId("archive-session-archive-1"));
    chooseMoreAction("archiveDrawer.receipt");
    expect(await screen.findByTestId("archive-receipt")).toHaveTextContent("succeeded");
    expect(mocks.getContinuityReceipt).toHaveBeenCalledWith("archive:archive-1");
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

  it("keeps optional Semantic search selectable and non-reopenable restoration disabled with explanations", async () => {
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    fireEvent.change(await screen.findByTestId("archive-search-input"), { target: { value: "receipt" } });

    await screen.findByTestId("archive-reader-props");
    expect(screen.getByTestId("archive-search-mode-semantic")).toBeEnabled();
    const reopen = screen.getByRole("button", { name: "archiveDrawer.reopen" });
    expect(reopen).toBeDisabled();
    // The action bar titles a button with its own label, so the reason is shown as text beside the state.
    expect(screen.getByText(/agent history no longer on disk/)).toBeInTheDocument();
  });

  it("shows why a policy archived a session on its row and beside Reopen", async () => {
    const note = "Archived by the idle policy (6h0m0s): no agent work since 2026-11-20 08:43 UTC (21h0m0s idle); goal paused on thread 01a1f2dd; no turn or sub-agent running.";
    mocks.listArchivedSessions.mockResolvedValue({
      total: 1,
      sessions: [{
        id: "archive-1", archived_at: "2026-11-21T05:47:44Z", created_at: "2026-11-19T08:13:00Z",
        agent_type: "codex", pane_name: "research lead", message_count: 3, restore_state: "reopenable", archive_note: note,
      }],
    });
    render(<ArchiveDrawer open onClose={vi.fn()} activeSessionId="live-1" onSendToComposer={vi.fn()} onReopened={vi.fn()} />);
    expect(await screen.findByTestId("archive-note-archive-1")).toHaveTextContent(note);
    fireEvent.click(screen.getByTestId("archive-session-archive-1"));
    expect(await screen.findByTestId("archive-note")).toHaveTextContent(note);
    expect(screen.getByTestId("archive-reopen")).toBeEnabled();
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
