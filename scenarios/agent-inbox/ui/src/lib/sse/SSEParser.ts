/**
 * SSEParser - Buffered Server-Sent Events parser.
 *
 * This parser handles SSE events that may be split across chunk boundaries,
 * ensuring no events are lost when network delivers partial data.
 *
 * PROBLEM SOLVED: Naive line-based parsing (text.split("\n")) loses events
 * when JSON payloads are split across ReadableStream chunks.
 *
 * SEAM: Pure functions for event parsing - easily unit tested.
 */

/**
 * Represents a parsed SSE event.
 */
export interface SSEEvent {
  /** The event type (from "event:" field, defaults to "message") */
  type: string;
  /** The event data (from "data:" field) */
  data: string;
  /** Optional event ID (from "id:" field) */
  id?: string;
  /** Optional retry interval in ms (from "retry:" field) */
  retry?: number;
}

/**
 * Options for creating an SSEParser instance.
 */
export interface SSEParserOptions {
  /** Called when a complete event is parsed */
  onEvent: (event: SSEEvent) => void;
  /** Called when an error occurs during parsing (e.g., malformed JSON) */
  onError?: (error: Error, rawData: string) => void;
}

/**
 * Buffered SSE parser that handles events split across chunk boundaries.
 *
 * Usage:
 * ```typescript
 * const parser = new SSEParser({
 *   onEvent: (event) => console.log("Event:", event),
 *   onError: (error, raw) => console.error("Parse error:", error, raw),
 * });
 *
 * // In streaming loop:
 * while (true) {
 *   const { done, value } = await reader.read();
 *   if (done) {
 *     parser.flush();
 *     break;
 *   }
 *   parser.processChunk(decoder.decode(value, { stream: true }));
 * }
 * ```
 */
export class SSEParser {
  private buffer = "";
  private currentEvent: Partial<SSEEvent> = {};
  private readonly onEvent: (event: SSEEvent) => void;
  private readonly onError?: (error: Error, rawData: string) => void;

  constructor(options: SSEParserOptions) {
    this.onEvent = options.onEvent;
    this.onError = options.onError;
  }

  /**
   * Process an incoming chunk of data.
   *
   * This method buffers incomplete lines and processes complete ones.
   * A complete event is emitted when an empty line is encountered,
   * following the SSE specification.
   *
   * @param chunk - Raw text chunk from the stream
   */
  processChunk(chunk: string): void {
    this.buffer += chunk;

    // Split on newlines, keeping incomplete lines in buffer
    const lines = this.buffer.split("\n");

    // The last element may be incomplete (no trailing newline)
    this.buffer = lines.pop() || "";

    for (const line of lines) {
      this.processLine(line);
    }
  }

  /**
   * Flush any remaining buffered data at stream end.
   *
   * Call this when the stream is done to ensure any final
   * partial event is processed.
   */
  flush(): void {
    // Process any remaining buffer content
    if (this.buffer.trim()) {
      this.processLine(this.buffer);
      this.buffer = "";
    }

    // Emit any pending event
    this.emitCurrentEvent();
  }

  /**
   * Process a single line of SSE data.
   *
   * SSE format:
   * - "event: <type>" - sets event type
   * - "data: <data>" - sets event data (can be multiple lines)
   * - "id: <id>" - sets event ID
   * - "retry: <ms>" - sets reconnection interval
   * - ":" prefix - comment, ignored
   * - empty line - dispatches the event
   */
  private processLine(line: string): void {
    // Remove carriage return if present (Windows-style line endings)
    const cleanLine = line.replace(/\r$/, "");

    // Empty line triggers event dispatch
    if (cleanLine === "") {
      this.emitCurrentEvent();
      return;
    }

    // Comment lines start with colon
    if (cleanLine.startsWith(":")) {
      return;
    }

    // Parse field:value format
    const colonIndex = cleanLine.indexOf(":");
    if (colonIndex === -1) {
      // Line with no colon is treated as field name with empty value
      this.setField(cleanLine, "");
      return;
    }

    const field = cleanLine.slice(0, colonIndex);
    // Value starts after colon, with optional leading space
    let value = cleanLine.slice(colonIndex + 1);
    if (value.startsWith(" ")) {
      value = value.slice(1);
    }

    this.setField(field, value);
  }

  /**
   * Set a field on the current event being built.
   */
  private setField(field: string, value: string): void {
    switch (field) {
      case "event":
        this.currentEvent.type = value;
        break;
      case "data":
        // Data can span multiple lines - concatenate with newlines
        if (this.currentEvent.data === undefined) {
          this.currentEvent.data = value;
        } else {
          this.currentEvent.data += "\n" + value;
        }
        break;
      case "id":
        // Per SSE spec, ignore IDs containing null character
        if (!value.includes("\u0000")) {
          this.currentEvent.id = value;
        }
        break;
      case "retry": {
        const retry = parseInt(value, 10);
        if (!isNaN(retry) && retry >= 0) {
          this.currentEvent.retry = retry;
        }
        break;
      }
      // Unknown fields are ignored per SSE spec
    }
  }

  /**
   * Emit the current event if it has data.
   */
  private emitCurrentEvent(): void {
    // Only emit if we have data (required field)
    if (this.currentEvent.data !== undefined) {
      const event: SSEEvent = {
        type: this.currentEvent.type || "message",
        data: this.currentEvent.data,
      };

      if (this.currentEvent.id !== undefined) {
        event.id = this.currentEvent.id;
      }
      if (this.currentEvent.retry !== undefined) {
        event.retry = this.currentEvent.retry;
      }

      try {
        this.onEvent(event);
      } catch (error) {
        this.onError?.(
          error instanceof Error ? error : new Error(String(error)),
          this.currentEvent.data
        );
      }
    }

    // Reset for next event
    this.currentEvent = {};
  }

  /**
   * Reset the parser state.
   * Useful for reconnection scenarios.
   */
  reset(): void {
    this.buffer = "";
    this.currentEvent = {};
  }
}
