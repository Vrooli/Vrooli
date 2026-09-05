/**
 * Streaming completion, SSE parsing, and streaming event types.
 */
import { SSEParser } from "./sse";
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";
import type { Message } from "./api-types";

// =============================================================================
// Streaming Event Types
// =============================================================================

/**
 * StreamingEvent represents a Server-Sent Event (SSE) from the completion endpoint.
 *
 * Events are received during streaming completions and provide real-time updates:
 * - `content` - Text chunks as they're generated
 * - `image_generated` - AI-generated images (multimodal models)
 * - `tool_call_start/result` - Tool execution lifecycle
 * - `tool_pending_approval` - Tool requires user approval
 * - `awaiting_approvals` - Stream paused waiting for approvals
 * - `error/warning` - Issues during completion
 * - `progress` - Status updates during long operations
 *
 * TEMPORAL FLOW: `completion_id` enables client-side correlation of events
 * from the same completion request, helping prevent stale event handling
 * when requests are cancelled or replaced.
 *
 * @see completeChat - Main streaming completion function
 * @see useCompletion - React hook for managing streaming state
 * @see docs/SEAMS.md - Full protocol specification
 */
const STREAMING_EVENT_TYPES = new Set([
  "content", "image_generated", "tool_call_start", "tool_call_result",
  "tool_calls_complete", "tool_pending_approval", "awaiting_approvals",
  "error", "warning", "progress",
]);

function isStreamingEvent(v: unknown): v is StreamingEvent {
  return typeof v === 'object' && v !== null
    && typeof (v as Record<string, unknown>).type === 'string'
    && STREAMING_EVENT_TYPES.has((v as Record<string, unknown>).type as string);
}

export interface StreamingEvent {
  /** Event type discriminator */
  type: "content" | "image_generated" | "tool_call_start" | "tool_call_result" | "tool_calls_complete" | "tool_pending_approval" | "awaiting_approvals" | "error" | "warning" | "progress";
  /** Unique ID for this completion request (for stale event filtering) */
  completion_id?: string;
  /** Text content chunk (type: "content") */
  content?: string;
  /** Generated image URL (type: "image_generated") */
  image_url?: string;
  /** Tool name (type: "tool_call_start", "tool_call_result", "tool_pending_approval") */
  tool_name?: string;
  /** Tool ID (legacy field, prefer tool_call_id) */
  tool_id?: string;
  /** Tool call ID (type: "tool_pending_approval", "tool_call_result") */
  tool_call_id?: string;
  /** JSON-encoded tool arguments */
  arguments?: string;
  /** JSON-encoded tool result */
  result?: string;
  /** Tool execution status ("completed", "failed") */
  status?: string;
  /** Error message (type: "error", "tool_call_result" with failure) */
  error?: string;
  /** Whether auto-continue is happening (type: "tool_calls_complete") */
  continuing?: boolean;
  /** Whether streaming is complete */
  done?: boolean;
  /** Current phase (type: "progress") */
  phase?: string;
  /** Status message (type: "progress", "warning") */
  message?: string;
  /** Error/warning code (type: "error", "warning") */
  code?: string;
  /** Server request ID for debugging */
  request_id?: string;
  /** Signal to deactivate active template (type: "tool_call_result") */
}

/**
 * Process an SSE stream with proper buffering.
 *
 * This uses SSEParser to handle events that may be split across chunk boundaries,
 * preventing data loss that occurs with naive line-based parsing.
 *
 * @param reader - ReadableStream reader
 * @param options - Callbacks for content chunks and events
 */
export async function processSSEStream(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  options?: {
    onChunk?: (content: string) => void;
    onEvent?: (event: StreamingEvent) => void;
    signal?: AbortSignal;
  }
): Promise<void> {
  const decoder = new TextDecoder();
  const parser = new SSEParser({
    onEvent: (sseEvent) => {
      // Skip [DONE] sentinel
      if (sseEvent.data === "[DONE]") return;

      try {
        const parsed: unknown = JSON.parse(sseEvent.data);
        if (!isStreamingEvent(parsed)) {
          console.warn("[SSE] Unexpected event shape:", JSON.stringify(parsed).slice(0, 200));
          return;
        }

        // Legacy callback for content chunks
        if (parsed.content && options?.onChunk) {
          options.onChunk(parsed.content);
        }

        // Event-based callback
        if (options?.onEvent) {
          options.onEvent(parsed);
        }
      } catch {
        // Log parse errors for debugging but don't crash the stream
        console.warn(`Failed to parse SSE event data: ${sseEvent.data.slice(0, 100)}...`);
      }
    },
    onError: (error, rawData) => {
      console.error("SSE parse error:", error.message, rawData.slice(0, 100));
    },
  });

  try {
    for (;;) {
      // Check for abort before each read
      if (options?.signal?.aborted) {
        void reader.cancel();
        throw new DOMException("Aborted", "AbortError");
      }

      const { done, value } = await reader.read();
      if (done) {
        // Flush any remaining buffered data
        parser.flush();
        break;
      }

      // Process chunk with buffered parser
      parser.processChunk(decoder.decode(value, { stream: true }));
    }
  } finally {
    // Ensure reader is released on any exit path
    try {
      reader.releaseLock();
    } catch {
      // Reader may already be released
    }
  }
}

// =============================================================================
// Chat Completion
// =============================================================================

// Chat completion with streaming
// Supports AbortController signal for cancellation on unmount or new request
export interface SkillPayloadForAPI {
  id: string;
  name: string;
  content: string;
  key: string;
  label: string;
  tags?: string[];
  targetToolId?: string;
}

/**
 * Streaming path return type - callbacks deliver content, no message value.
 */
type StreamCompletionResult = ReturnType<() => void>;

export async function completeChat(
  chatId: string,
  options?: {
    stream?: boolean;
    onChunk?: (content: string) => void;
    onEvent?: (event: StreamingEvent) => void;
    signal?: AbortSignal;
    skills?: SkillPayloadForAPI[];
  }
): Promise<Message | StreamCompletionResult> {
  const stream = options?.stream ?? true;
  const params = new URLSearchParams();
  params.set("stream", String(stream));
  const url = buildApiUrl(`/chats/${chatId}/complete?${params.toString()}`, { baseUrl: API_BASE });

  // Build request body with skills if provided
  const body = options?.skills && options.skills.length > 0
    ? JSON.stringify({ skills: options.skills })
    : undefined;

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body,
    signal: options?.signal,
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`Chat completion failed: ${errorText}`);
  }

  if (stream) {
    const reader = res.body?.getReader();
    if (!reader) {
      throw new Error("Streaming not supported");
    }
    await processSSEStream(reader, options);
  } else {
    return jsonResponse<Message>(res);
  }
}
