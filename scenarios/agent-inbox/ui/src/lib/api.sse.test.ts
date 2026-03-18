/**
 * Tests for API client - SSE parsing edge cases and regenerateMessage
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  completeChat,
  regenerateMessage,
  type StreamingEvent,
} from "./api";
import { mockFetch, createMockResponse, createStreamingResponse } from "./api.test.helpers";

describe("regenerateMessage", () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  it("regenerates with streaming", async () => {
    const events = [
      'data: {"type":"content","content":"Regenerated"}\n\n',
      "data: [DONE]\n\n",
    ];

    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));

    const chunks: string[] = [];

    await regenerateMessage("chat-1", "msg-1", {
      stream: true,
      onChunk: (chunk) => chunks.push(chunk),
    });

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:3000/api/v1/chats/chat-1/messages/msg-1/regenerate?stream=true",
      expect.any(Object)
    );
    expect(chunks).toEqual(["Regenerated"]);
  });

  it("returns message for non-streaming", async () => {
    const mockMessage = { id: "msg-regen", content: "Regenerated" };
    mockFetch.mockResolvedValueOnce(createMockResponse(mockMessage));

    const result = await regenerateMessage("chat-1", "msg-1", { stream: false });

    expect(result).toEqual(mockMessage);
  });
});

describe("SSE parsing edge cases", () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  it("handles events split across chunks", async () => {
    const events = [
      'data: {"type":"content",',
      '"content":"Hello"}\n\ndata: {"type":"content","content":" world"}\n\n',
      "data: [DONE]\n\n",
    ];

    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));

    const receivedEvents: StreamingEvent[] = [];

    await completeChat("chat-1", {
      stream: true,
      onEvent: (event) => receivedEvents.push(event),
    });

    expect(receivedEvents.length).toBeGreaterThan(0);
  });

  it("handles multiple events in single chunk", async () => {
    const events = [
      'data: {"type":"content","content":"One"}\n\ndata: {"type":"content","content":"Two"}\n\ndata: {"type":"content","content":"Three"}\n\n',
      "data: [DONE]\n\n",
    ];

    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));

    const receivedEvents: StreamingEvent[] = [];

    await completeChat("chat-1", {
      stream: true,
      onEvent: (event) => receivedEvents.push(event),
    });

    expect(receivedEvents).toHaveLength(3);
    expect(receivedEvents.map(e => e.content)).toEqual(["One", "Two", "Three"]);
  });

  it("handles empty lines gracefully", async () => {
    const events = [
      "\n\n",
      'data: {"type":"content","content":"Hello"}\n\n',
      "\n",
      "data: [DONE]\n\n",
    ];

    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));

    const receivedEvents: StreamingEvent[] = [];

    await completeChat("chat-1", {
      stream: true,
      onEvent: (event) => receivedEvents.push(event),
    });

    expect(receivedEvents).toHaveLength(1);
  });

  it("handles image_generated events", async () => {
    const events = [
      'data: {"type":"image_generated","image_url":"https://example.com/image.png"}\n\n',
      "data: [DONE]\n\n",
    ];

    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));

    const receivedEvents: StreamingEvent[] = [];

    await completeChat("chat-1", {
      stream: true,
      onEvent: (event) => receivedEvents.push(event),
    });

    expect(receivedEvents).toHaveLength(1);
    expect(receivedEvents[0]!.type).toBe("image_generated");
    expect(receivedEvents[0]!.image_url).toBe("https://example.com/image.png");
  });

  it("handles progress events", async () => {
    const events = [
      'data: {"type":"progress","phase":"executing","message":"Running tool..."}\n\n',
      "data: [DONE]\n\n",
    ];

    mockFetch.mockResolvedValueOnce(createStreamingResponse(events));

    const receivedEvents: StreamingEvent[] = [];

    await completeChat("chat-1", {
      stream: true,
      onEvent: (event) => receivedEvents.push(event),
    });

    expect(receivedEvents).toHaveLength(1);
    expect(receivedEvents[0]!.type).toBe("progress");
    expect(receivedEvents[0]!.phase).toBe("executing");
  });
});
