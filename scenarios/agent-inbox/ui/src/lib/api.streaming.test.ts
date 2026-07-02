/**
 * Tests for API client - completeChat streaming functionality
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  completeChat,
  type StreamingEvent,
} from "./api";
import { mockFetch, createMockResponse, createStreamingResponse } from "./api.test.helpers";

describe("completeChat", () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("streams content events", async () => {
    const events = [
      'data: {"type":"content","content":"Hello"}\n\n',
      'data: {"type":"content","content":" world"}\n\n',
      "data: [DONE]\n\n",
    ];

    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));

    const receivedEvents: StreamingEvent[] = [];
    const receivedChunks: string[] = [];

    await completeChat("chat-1", {
      stream: true,
      onEvent: (event) => receivedEvents.push(event),
      onChunk: (chunk) => receivedChunks.push(chunk),
    });

    expect(receivedEvents).toHaveLength(2);
    expect(receivedEvents[0]).toEqual({ type: "content", content: "Hello" });
    expect(receivedEvents[1]).toEqual({ type: "content", content: " world" });
    expect(receivedChunks).toEqual(["Hello", " world"]);
  });

  it("handles tool_call_start events", async () => {
    const events = [
      'data: {"type":"tool_call_start","tool_id":"call_123","tool_name":"run-agent","arguments":"{}"}\n\n',
      "data: [DONE]\n\n",
    ];
    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));
    const receivedEvents: StreamingEvent[] = [];
    await completeChat("chat-1", { stream: true, onEvent: (event) => receivedEvents.push(event) });
    expect(receivedEvents).toHaveLength(1);
    expect(receivedEvents[0]!.type).toBe("tool_call_start");
    expect(receivedEvents[0]!.tool_name).toBe("run-agent");
  });

  it("handles tool_call_result events", async () => {
    const events = [
      'data: {"type":"tool_call_result","tool_id":"call_123","status":"completed","result":"{\\"success\\":true}"}\n\n',
      "data: [DONE]\n\n",
    ];
    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));
    const receivedEvents: StreamingEvent[] = [];
    await completeChat("chat-1", { stream: true, onEvent: (event) => receivedEvents.push(event) });
    expect(receivedEvents).toHaveLength(1);
    expect(receivedEvents[0]!.type).toBe("tool_call_result");
    expect(receivedEvents[0]!.status).toBe("completed");
  });

  it("handles error events", async () => {
    const events = [
      'data: {"type":"error","error":"Something went wrong"}\n\n',
      "data: [DONE]\n\n",
    ];
    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));
    const receivedEvents: StreamingEvent[] = [];
    await completeChat("chat-1", { stream: true, onEvent: (event) => receivedEvents.push(event) });
    expect(receivedEvents).toHaveLength(1);
    expect(receivedEvents[0]!.type).toBe("error");
    expect(receivedEvents[0]!.error).toBe("Something went wrong");
  });

  it("handles pending approval events", async () => {
    const events = [
      'data: {"type":"tool_pending_approval","tool_call_id":"call_456","tool_name":"dangerous-tool"}\n\n',
      'data: {"type":"awaiting_approvals"}\n\n',
      "data: [DONE]\n\n",
    ];
    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));
    const receivedEvents: StreamingEvent[] = [];
    await completeChat("chat-1", { stream: true, onEvent: (event) => receivedEvents.push(event) });
    expect(receivedEvents).toHaveLength(2);
    expect(receivedEvents[0]!.type).toBe("tool_pending_approval");
    expect(receivedEvents[1]!.type).toBe("awaiting_approvals");
  });

  it("ignores invalid JSON gracefully", async () => {
    const events = [
      'data: {"type":"content","content":"Valid"}\n\n',
      "data: {invalid json}\n\n",
      'data: {"type":"content","content":" chunk"}\n\n',
      "data: [DONE]\n\n",
    ];
    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));
    const receivedEvents: StreamingEvent[] = [];
    await completeChat("chat-1", { stream: true, onEvent: (event) => receivedEvents.push(event) });
    expect(receivedEvents).toHaveLength(2);
  });

  it("handles abort signal", async () => {
    const controller = new AbortController();
    let readCount = 0;
    const reader: ReadableStreamDefaultReader<Uint8Array> = {
      read: async () => {
        readCount++;
        if (readCount === 1) {
          return { done: false, value: new TextEncoder().encode('data: {"type":"content","content":"Hello"}\n\n') };
        }
        await new Promise((_, reject) => {
          controller.signal.addEventListener("abort", () => { reject(new DOMException("Aborted", "AbortError")); });
        });
        return { done: true, value: undefined };
      },
      releaseLock: () => {},
      cancel: async () => {},
      closed: Promise.resolve(undefined),
    };
    const body = { getReader: () => reader } as ReadableStream<Uint8Array>;
    mockFetch.mockResolvedValueOnce({ ok: true, status: 200, body, headers: new Headers() } as Response);
    const completionPromise = completeChat("chat-1", { stream: true, signal: controller.signal });
    controller.abort();
    await expect(completionPromise).rejects.toThrow("Aborted");
  });

  it("includes skills in request body", async () => {
    mockFetch.mockResolvedValueOnce(createStreamingResponse(["data: [DONE]\n\n"]));
    const skills = [{ id: "skill-1", name: "Test", content: "Do X", key: "test", label: "Test" }];
    await completeChat("chat-1", { stream: true, skills });
    expect(mockFetch).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ body: JSON.stringify({ skills }) }));
  });

  it("returns message for non-streaming", async () => {
    const mockMessage = { id: "msg-1", content: "Response" };
    mockFetch.mockResolvedValueOnce(createMockResponse(mockMessage));
    const result = await completeChat("chat-1", { stream: false });
    expect(result).toEqual(mockMessage);
  });

  it("throws on HTTP error", async () => {
    mockFetch.mockResolvedValueOnce(createMockResponse("Internal server error", { status: 500, ok: false }));
    await expect(completeChat("chat-1")).rejects.toThrow("Chat completion failed");
  });

  it("throws when streaming not supported", async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, status: 200, body: null, headers: new Headers() } as Response);
    await expect(completeChat("chat-1", { stream: true })).rejects.toThrow("Streaming not supported");
  });
});
