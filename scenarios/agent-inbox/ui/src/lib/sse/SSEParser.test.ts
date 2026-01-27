import { describe, it, expect, beforeEach } from "vitest";
import { SSEParser, type SSEEvent } from "./SSEParser";

describe("SSEParser", () => {
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
      // Split JSON across chunk boundary
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
      expect(events).toHaveLength(1); // No additional event
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
      // Per SSE spec, a field with no colon is treated as field name with empty value
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
      // Simulate chunks arriving from network
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
      // Real network conditions - JSON split in middle
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
