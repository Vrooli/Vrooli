/**
 * Tests for SSEParser - Error handling, edge cases, and real-world scenarios
 */

import { describe, it, expect, beforeEach } from "vitest";
import { SSEParser, type SSEEvent } from "./SSEParser";

describe("SSEParser - errors and edge cases", () => {
  let events: SSEEvent[];
  let errors: Array<{ error: Error; rawData: string }>;
  let parser: SSEParser;

  beforeEach(() => {
    events = [];
    errors = [];
    parser = new SSEParser({
      onEvent: (event) => events.push(event),
      onError: (error, rawData) => errors.push({ error, rawData }),
    });
  });

  describe("error handling", () => {
    it("calls onError when event handler throws", () => {
      const errorParser = new SSEParser({
        onEvent: () => {
          throw new Error("Handler error");
        },
        onError: (error, rawData) => errors.push({ error, rawData }),
      });

      errorParser.processChunk("data: test\n\n");

      expect(errors).toHaveLength(1);
      expect(errors[0]!.error.message).toBe("Handler error");
      expect(errors[0]!.rawData).toBe("test");
    });

    it("continues parsing after error", () => {
      let callCount = 0;
      const errorParser = new SSEParser({
        onEvent: (event) => {
          callCount++;
          if (callCount === 1) throw new Error("First error");
          events.push(event);
        },
        onError: (error, rawData) => errors.push({ error, rawData }),
      });

      errorParser.processChunk("data: first\n\ndata: second\n\n");

      expect(errors).toHaveLength(1);
      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("second");
    });
  });

  describe("edge cases", () => {
    it("handles Windows-style line endings (CRLF)", () => {
      parser.processChunk("data: test\r\n\r\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("test");
    });

    it("handles empty data field", () => {
      parser.processChunk("data:\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("");
    });

    it("handles data with leading space (space is stripped)", () => {
      parser.processChunk("data: hello\n\n");
      expect(events[0]!.data).toBe("hello");

      parser.reset();
      events = [];

      parser.processChunk("data:hello\n\n");
      expect(events[0]!.data).toBe("hello");
    });

    it("handles field with no colon (empty value)", () => {
      parser.processChunk("data\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("");
    });

    it("ignores IDs containing null character", () => {
      parser.processChunk("id: bad\x00id\ndata: test\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.id).toBeUndefined();
    });

    it("handles very long data", () => {
      const longData = "x".repeat(100000);
      parser.processChunk(`data: ${longData}\n\n`);

      expect(events).toHaveLength(1);
      expect(events[0]!.data.length).toBe(100000);
    });

    it("handles [DONE] sentinel value", () => {
      parser.processChunk("data: [DONE]\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("[DONE]");
    });
  });

  describe("real-world scenarios", () => {
    it("handles typical OpenAI-style streaming response", () => {
      const chunks = [
        'data: {"id":"1","choices":[{"delta":{"content":"Hello"}}]}\n\n',
        'data: {"id":"1","choices":[{"delta":{"content":" world"}}]}\n\n',
        "data: [DONE]\n\n",
      ];

      for (const chunk of chunks) {
        parser.processChunk(chunk);
      }

      expect(events).toHaveLength(3);
      expect(events[2]!.data).toBe("[DONE]");
    });

    it("handles chunked JSON split unpredictably", () => {
      const fullData = 'data: {"type":"content","text":"Hello, I am Claude."}\n\n';
      const splitPoint = fullData.indexOf("Claude");

      parser.processChunk(fullData.slice(0, splitPoint));
      parser.processChunk(fullData.slice(splitPoint));

      expect(events).toHaveLength(1);
      expect(JSON.parse(events[0]!.data)).toEqual({
        type: "content",
        text: "Hello, I am Claude.",
      });
    });
  });
});
