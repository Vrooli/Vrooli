/**
 * Tests for SSEParser - Basic parsing, chunk boundaries, and multi-line data
 */

import { describe, it, expect, beforeEach } from "vitest";
import { SSEParser, type SSEEvent } from "./SSEParser";

describe("SSEParser - basic parsing", () => {
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

  describe("basic parsing", () => {
    it("handles complete events in a single chunk", () => {
      parser.processChunk('data: {"message": "hello"}\n\n');

      expect(events).toHaveLength(1);
      expect(events[0]).toEqual({
        type: "message",
        data: '{"message": "hello"}',
      });
    });

    it("handles multiple events in a single chunk", () => {
      parser.processChunk(
        'data: first\n\ndata: second\n\ndata: third\n\n'
      );

      expect(events).toHaveLength(3);
      expect(events.map((e) => e.data)).toEqual(["first", "second", "third"]);
    });

    it("parses event type correctly", () => {
      parser.processChunk("event: content\ndata: test data\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]).toEqual({
        type: "content",
        data: "test data",
      });
    });

    it("parses event ID correctly", () => {
      parser.processChunk("id: 123\ndata: test\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]).toEqual({
        type: "message",
        data: "test",
        id: "123",
      });
    });

    it("parses retry interval correctly", () => {
      parser.processChunk("retry: 5000\ndata: test\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]).toEqual({
        type: "message",
        data: "test",
        retry: 5000,
      });
    });

    it("ignores invalid retry values", () => {
      parser.processChunk("retry: invalid\ndata: test\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.retry).toBeUndefined();
    });

    it("ignores comment lines", () => {
      parser.processChunk(": this is a comment\ndata: test\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("test");
    });
  });

  describe("chunk boundary handling (the critical fix)", () => {
    it("buffers incomplete events across chunks", () => {
      parser.processChunk('data: {"mess');
      expect(events).toHaveLength(0);

      parser.processChunk('age": "hello"}\n\n');
      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe('{"message": "hello"}');
    });

    it("handles data split at newline boundary", () => {
      parser.processChunk("data: hello");
      expect(events).toHaveLength(0);

      parser.processChunk("\n\n");
      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("hello");
    });

    it("handles empty line split across chunks", () => {
      parser.processChunk("data: test\n");
      expect(events).toHaveLength(0);

      parser.processChunk("\n");
      expect(events).toHaveLength(1);
    });

    it("handles multiple partial chunks correctly", () => {
      parser.processChunk("da");
      parser.processChunk("ta: ");
      parser.processChunk("part");
      parser.processChunk("ial ");
      parser.processChunk("data");
      parser.processChunk("\n");
      parser.processChunk("\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("partial data");
    });

    it("handles event type split across chunks", () => {
      parser.processChunk("eve");
      parser.processChunk("nt: content\n");
      parser.processChunk("data: test\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.type).toBe("content");
    });
  });

  describe("multi-line data", () => {
    it("concatenates multiple data lines with newlines", () => {
      parser.processChunk("data: line1\ndata: line2\ndata: line3\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("line1\nline2\nline3");
    });

    it("handles multi-line data split across chunks", () => {
      parser.processChunk("data: line1\ndata: li");
      parser.processChunk("ne2\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("line1\nline2");
    });
  });

  describe("flush", () => {
    it("emits remaining buffered event on flush", () => {
      parser.processChunk("data: incomplete");
      expect(events).toHaveLength(0);

      parser.flush();
      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("incomplete");
    });

    it("does not emit empty event on flush with empty buffer", () => {
      parser.processChunk("data: test\n\n");
      expect(events).toHaveLength(1);

      parser.flush();
      expect(events).toHaveLength(1);
    });

    it("handles flush after complete event", () => {
      parser.processChunk("data: test\n\n");
      parser.flush();

      expect(events).toHaveLength(1);
    });
  });

  describe("reset", () => {
    it("clears buffer and current event", () => {
      parser.processChunk("data: partial");
      expect(events).toHaveLength(0);

      parser.reset();
      parser.flush();

      expect(events).toHaveLength(0);
    });

    it("allows fresh parsing after reset", () => {
      parser.processChunk("data: old");
      parser.reset();
      parser.processChunk("data: new\n\n");

      expect(events).toHaveLength(1);
      expect(events[0]!.data).toBe("new");
    });
  });
});
