import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor, within } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { strings } from "../consts/strings";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import type { ConversationEvent, ConversationThreadSummary } from "../api/conversation";

// The Messages view lists the conversation: human and agent messages, with the
// work between them collapsed into one line, injected content as quiet
// notices, and each subagent as a group that opens its own conversation.

const { mockGetConversationWork, mockGetConversationSession } = vi.hoisted(() => ({
  mockGetConversationWork: vi.fn(),
  mockGetConversationSession: vi.fn(),
}));

vi.mock("../hooks/useConversationSession", () => ({
  refreshConversationSession: vi.fn().mockResolvedValue({ ok: true, addedEvents: 0 }),
  loadOlderConversationPage: vi.fn().mockResolvedValue(false),
  loadConversationPageContaining: vi.fn().mockResolvedValue(false),
}));

vi.mock("../api/conversation", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/conversation")>()),
  getConversationWork: mockGetConversationWork,
  getConversationSession: mockGetConversationSession,
  searchConversation: vi.fn().mockResolvedValue({ matches: [], truncated: false, totalMatches: 0, hasMore: false, nextAfterSequence: 0 }),
}));

vi.mock("@vrooli/file-viewer", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@vrooli/file-viewer")>()),
  MarkdownRenderer: ({ content }: { content: string }) => <div data-testid="mock-markdown">{content}</div>,
}));

const at = (seconds: number) => new Date(Date.UTC(2026, 10, 26, 10, 0, seconds)).toISOString();

function row(overrides: Partial<ConversationEvent> & { id: string; sequence: number }): ConversationEvent {
  return {
    sessionId: "sess-1",
    source: "codex_tailer",
    role: "assistant",
    text: `Message ${String(overrides.sequence)}`,
    speechParagraphs: [],
    summarized: false,
    createdAt: at(overrides.sequence),
    occurredAt: at(overrides.sequence),
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
    kind: "message",
    author: "agent",
    ...overrides,
  };
}

function work(id: string, from: number, to: number, counts: Partial<NonNullable<ConversationEvent["work"]>> = {}, threadId?: string): ConversationEvent {
  return row({
    id, sequence: to, kind: "work", text: "", ...(threadId ? { threadId } : {}),
    work: { fromSequence: from, toSequence: to, toolCalls: 0, toolResults: 0, reasoning: 0, context: 0, startedAt: at(from), endedAt: at(to), ...counts },
  });
}

const props = {
  sessionId: "sess-1",
  onPlayFromHere: vi.fn(),
  onPlayEvent: vi.fn(),
  activeSpeakingEventId: null,
  isTtsSpeaking: false,
  summarizeLevel: "moderate" as const,
  selectedVersionForEvent: vi.fn(() => "active" as const),
  summarizingEventId: null,
  getSummarizeError: vi.fn(() => null),
  onClearSummarizeError: vi.fn(),
  onToggleSummarized: vi.fn(),
  onChangeLevel: vi.fn(),
  playbackFocusRequest: null,
  onOpenFilePreview: vi.fn(),
};

function seed(events: ConversationEvent[], threads: ConversationThreadSummary[] = []) {
  useConversationStore.setState({
    sessions: {
      "sess-1": createConversationSessionState({ events, threads, cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
    },
  });
}

describe("Messages view projection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useConversationStore.setState({ sessions: {} });
    vi.stubGlobal("IntersectionObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), unobserve: vi.fn(), disconnect: vi.fn() })));
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), unobserve: vi.fn(), disconnect: vi.fn() })));
  });

  it("collapses the work between two messages into one line that opens to the tool calls", async () => {
    seed([
      row({ id: "u1", sequence: 1, role: "user", author: "human", text: "Fix the release flow" }),
      work("work::2", 2, 4, { toolCalls: 2, toolResults: 1 }),
      row({ id: "a5", sequence: 5, text: "Done: the release flow passes." }),
    ]);
    mockGetConversationWork.mockResolvedValue([
      row({ id: "t2", sequence: 2, kind: "tool_call", label: "exec", text: "[Tool call: exec]\n\"make test\"\nCall ID: call_1" }),
      row({ id: "t3", sequence: 3, kind: "tool_result", label: "tool", text: "[Tool result: tool]\nok\nCall ID: call_1", textTruncated: true }),
      row({ id: "t4", sequence: 4, kind: "tool_call", label: "apply_patch", text: "[Tool call: apply_patch]\n*** Begin Patch" }),
    ]);
    render(<MessagesPane {...props} />);

    // The conversation is two messages; the tool calls are not rows.
    expect(screen.getByTestId("msg-card-u1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-card-a5")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-card-work::2")).toBeNull();
    const line = screen.getByTestId("work-row-work::2");
    expect(line).not.toHaveAttribute("data-expanded");
    expect(within(line).getByTestId("work-summary-work::2").textContent).toBe(
      [strings.messagesPane.work.worked, strings.messagesPane.work.toolCalls].join(" · "),
    );
    expect(mockGetConversationWork).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("work-toggle-work::2"));
    expect(screen.getByTestId("work-row-work::2")).toHaveAttribute("data-expanded", "true");
    await waitFor(() => expect(screen.getByTestId("work-item-t2")).toBeInTheDocument());
    expect(mockGetConversationWork).toHaveBeenCalledWith("sess-1", expect.objectContaining({ fromSequence: 2, toSequence: 4 }), "");
    expect(["t2", "t3", "t4"].map((id) => screen.getByTestId(`work-item-${id}`).getAttribute("data-kind"))).toEqual(["tool_call", "tool_result", "tool_call"]);
    expect(within(screen.getByTestId("work-item-t2")).getByText("exec")).toBeInTheDocument();
    expect(within(screen.getByTestId("work-item-t2")).getByText("make test")).toBeInTheDocument();
  });

  it("names only the operator 'You': injected content is a quiet notice and a peer's message is the peer's", () => {
    seed([
      row({ id: "n1", sequence: 1, role: "user", author: "harness", kind: "notice", label: "task_notification", text: "<task-notification>\n<task-id>b1</task-id>\n<summary>Goal check-in: background work still running</summary>\n</task-notification>" }),
      row({ id: "p2", sequence: 2, role: "user", author: "peer", authorName: "serving-fabric", text: "WS-F question for the lead" }),
      row({ id: "l3", sequence: 3, role: "user", author: "automation", authorName: "agent-manager", label: "launch_prompt", text: "Read and execute the brief." }),
      row({ id: "h4", sequence: 4, role: "user", author: "human", text: "Thanks, keep going." }),
    ]);
    render(<MessagesPane {...props} />);

    const notice = screen.getByTestId("msg-card-n1");
    expect(notice).toHaveAttribute("data-notice", "task_notification");
    expect(screen.queryByTestId("msg-speaker-n1")).toBeNull();
    expect(screen.getByTestId("notice-summary-n1").textContent).toBe("Goal check-in: background work still running");
    fireEvent.click(screen.getByTestId("notice-toggle-n1"));
    expect(screen.getByTestId("notice-text-n1").textContent).toContain("<task-id>b1</task-id>");

    expect(screen.getByTestId("msg-speaker-p2").textContent).toBe(strings.messagesPane.speaker.peerNamed);
    expect(screen.getByTestId("msg-speaker-l3").textContent).toBe(strings.messagesPane.speaker.automationNamed);
    expect(screen.getByTestId("msg-speaker-h4").textContent).toBe(strings.messagesPane.speaker.you);
  });

  it("groups a subagent's conversation at the point it began and opens it in place of the list", async () => {
    const thread: ConversationThreadSummary = {
      threadId: "01a217a4", kind: "subagent", label: "Hilbert", description: "/root/sc9_owner",
      parentThreadId: "", firstSequence: 3, lastSequence: 9, messageCount: 2, workCount: 5, startedAt: at(3), endedAt: at(9),
    };
    // The main thread's own work between 2 and 10 is nothing but the
    // subagent's rows: the line is empty, the group carries the span.
    seed([
      row({ id: "u1", sequence: 1, role: "user", author: "human", text: "Spawn Hilbert for SC9" }),
      work("work::2", 2, 9),
      row({ id: "a10", sequence: 10, text: "Hilbert finished SC9." }),
    ], [thread]);
    mockGetConversationSession.mockResolvedValue({
      sessionId: "sess-1",
      events: [
        row({ id: "b3", sequence: 3, role: "user", author: "peer", threadId: "01a217a4", text: "Own SC9 end to end." }),
        work("work:01a217a4:4", 4, 8, { toolCalls: 3 }, "01a217a4"),
        row({ id: "r9", sequence: 9, threadId: "01a217a4", text: "SC9 is green." }),
      ],
      cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
      hasMore: false,
      oldestSequence: 1,
      capture: {},
    });
    render(<MessagesPane {...props} />);

    const group = screen.getByTestId("thread-group-01a217a4");
    expect(group).toHaveAttribute("data-thread-kind", "subagent");
    expect(group).toHaveAttribute("data-message-count", "2");
    expect(screen.getByTestId("thread-name-01a217a4").textContent).toBe("Hilbert");
    // The empty span draws no line of its own; the group stands for it.
    expect(screen.queryByTestId("work-row-work::2")).toBeNull();
    // It sits between the prompt and the reply.
    const order = [...document.querySelectorAll("[data-testid^='msg-card-'], [data-testid^='thread-group-']")].map((node) => node.getAttribute("data-testid"));
    expect(order).toEqual(["msg-card-u1", "thread-group-01a217a4", "msg-card-a10"]);

    fireEvent.click(group);
    const view = await screen.findByTestId("thread-view");
    expect(view).toHaveAttribute("data-thread-id", "01a217a4");
    expect(mockGetConversationSession).toHaveBeenCalledWith("sess-1", expect.objectContaining({ projection: "messages", threadId: "01a217a4" }));
    await waitFor(() => expect(screen.getByTestId("thread-msg-r9")).toBeInTheDocument());
    expect(screen.getByTestId("thread-msg-b3")).toBeInTheDocument();
    expect(screen.getByTestId("work-row-work:01a217a4:4")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("thread-back"));
    expect(screen.queryByTestId("thread-view")).toBeNull();
    expect(screen.getByTestId("msg-card-a10")).toBeInTheDocument();
  });
});
