// @ts-check

import { describe, expect, it, beforeEach, vi } from "vitest";

vi.mock("../../event-feed.js", () => {
  const appendEvent = vi.fn();
  const recordTranscript = vi.fn((tab, entry) => {
    if (!tab) return;
    if (!Array.isArray(tab.transcript)) {
      tab.transcript = [];
    }
    tab.transcript.push(entry);
    tab.transcriptByteSize =
      Number(tab.transcriptByteSize || 0) +
      (typeof entry.data === "string" ? entry.data.length : 0);
  });
  const showError = vi.fn();
  return { appendEvent, recordTranscript, showError };
});

vi.mock("../transport.js", () => ({
  fetchSessionTranscriptRequest: vi.fn(),
}));

vi.mock("../callbacks.js", () => ({
  notifyActiveTabUpdate: vi.fn(),
}));

vi.mock("../../notifications.js", () => ({
  showToast: vi.fn(() => Promise.resolve()),
}));

vi.mock("../../tab-manager.js", () => ({
  refreshTabButton: vi.fn(),
}));

const bufferFrom = (value) => Buffer.from(value, "utf8");

if (typeof globalThis.atob !== "function") {
  globalThis.atob = (input) => Buffer.from(input, "base64").toString("binary");
}

const buildTab = () => ({
  id: "tab-test",
  label: "tab-test",
  defaultLabel: "tab-test",
  colorId: "sky",
  term: /** @type {any} */ ({
    write: vi.fn(),
    reset: vi.fn(),
    focus: vi.fn(),
    scrollToBottom: vi.fn(),
    refresh: vi.fn(),
    rows: 24,
    cols: 80,
  }),
  fitAddon: /** @type {any} */ ({ fit: vi.fn() }),
  transcript: [],
  transcriptByteSize: 0,
  transcriptHydrated: false,
  transcriptHydrating: false,
  events: [],
  suppressed: {},
  telemetry: { typed: 0, queued: 0, sent: 0, batches: 0, lastBatchSize: 0 },
  pendingWrites: [],
  localEchoBuffer: [],
  lastSentSize: { cols: 0, rows: 0 },
  errorMessage: "",
  socketState: "disconnected",
  phase: "idle",
  session: { id: "session-123", command: "bash", args: [] },
  hasReceivedLiveOutput: false,
  hasEverConnected: false,
  replayPending: false,
  replayComplete: false,
  lastReplayCount: 0,
  lastReplayTruncated: false,
  heartbeatInterval: null,
  reconnecting: false,
  wasDetached: false,
  domItem: null,
  domButton: null,
  domClose: null,
  domLabel: null,
  domStatus: null,
});

await import("../stream-handlers.js");
const {
  hydrateTranscriptForTab,
  handleStreamEnvelope,
} = await import("../stream-handlers.js");
const { fetchSessionTranscriptRequest } = await import("../transport.js");
const { showToast } = await import("../../notifications.js");
const { appendEvent } = await import("../../event-feed.js");
const { notifyActiveTabUpdate } = await import("../callbacks.js");
const { state } = await import("../../state.js");

const base64 = (value) => bufferFrom(value).toString("base64");

describe("stream handlers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    state.activeTabId = "tab-test";
  });

  it("hydrates terminal output from persisted transcript", async () => {
    const tab = buildTab();
    state.activeTabId = tab.id;

    const transcriptBody = [
      JSON.stringify({
        timestamp: "2024-05-01T12:00:00Z",
        direction: "stdout",
        encoding: "base64",
        data: base64("hello world\n"),
      }),
      JSON.stringify({
        timestamp: "2024-05-01T12:00:01Z",
        direction: "status",
        message: "closed:client_requested",
      }),
    ].join("\n");

    fetchSessionTranscriptRequest.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(transcriptBody),
    });

    await hydrateTranscriptForTab(tab, { reason: "test" });

    expect(fetchSessionTranscriptRequest).toHaveBeenCalledWith("session-123");
    expect(tab.transcriptHydrated).toBe(true);
    expect(tab.term.write).toHaveBeenCalledWith("hello world\n");
    expect(tab.transcript).toHaveLength(2);
    expect(tab.hasReceivedLiveOutput).toBe(true);
    expect(notifyActiveTabUpdate).toHaveBeenCalled();
  });

  it("displays truncated history warning", () => {
    const tab = buildTab();
    state.activeTabId = tab.id;

    handleStreamEnvelope(tab, {
      type: "output_replay",
      payload: {
        chunks: [],
        truncated: true,
        complete: false,
        count: 0,
        generated: "2024-05-01T12:01:00Z",
      },
    });

    expect(showToast).toHaveBeenCalled();
    expect(appendEvent).toHaveBeenCalledWith(
      tab,
      "output-replay-truncated",
      expect.objectContaining({ truncated: true }),
    );
    expect(tab.lastReplayTruncated).toBe(true);
  });
});
