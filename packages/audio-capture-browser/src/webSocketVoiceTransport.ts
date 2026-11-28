import type {
  VoiceAudioChunk,
  VoiceStreamConnection,
  VoiceStreamHandlers,
  VoiceStreamStart,
} from "./transport";
import type { StreamMessage } from "./streamMessages";

export interface VoiceWebSocketCodec<Request, Response> {
  start(value: VoiceStreamStart): Request;
  audio(value: VoiceAudioChunk): Request;
  end(): Request;
  encode(value: Request): Uint8Array;
  decode(value: Uint8Array): Response;
  /** True only for the server's successful admission response to StreamStart. */
  isReady(value: Response): boolean;
  event(value: Response, start: VoiceStreamStart): StreamMessage | null;
}

const MAX_BUFFERED_AUDIO_BYTES = 2 * 1024 * 1024;

/**
 * Adapts the shared v2 voice contract to a browser WebSocket carrying binary
 * protobuf envelopes. Audio is never accepted by this adapter until the server
 * confirms session admission; a send failure closes the stream so the shared
 * provider can retain and replay uncommitted audio.
 */
export function openWebSocketVoiceStream<Request, Response>(
  url: string,
  codec: VoiceWebSocketCodec<Request, Response>,
  start: VoiceStreamStart,
  handlers: VoiceStreamHandlers,
): VoiceStreamConnection {
  let socket: WebSocket | null = null;
  let readyNotified = false;
  let serverCompleted = false;
  let requestOpen = true;
  let closed = false;

  const notifyClose = (error?: unknown) => {
    if (closed) return;
    closed = true;
    requestOpen = false;
    handlers.onClose(error);
  };

  const fail = (error: unknown) => {
    notifyClose(error);
    try {
      socket?.close(1011, "stream transport failed");
    } catch {
      // The close callback is advisory after the shared provider was notified.
    }
  };

  const send = (request: Request, isAudio: boolean) => {
    if (closed) return false;
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      fail(new Error("voice stream WebSocket is not open"));
      return false;
    }
    try {
      const encoded = codec.encode(request);
      if (isAudio && socket.bufferedAmount + encoded.byteLength > MAX_BUFFERED_AUDIO_BYTES) {
        fail(new Error("voice stream transport backpressure limit exceeded"));
        return false;
      }
      socket.send(encoded);
      return true;
    } catch (error) {
      fail(error);
      return false;
    }
  };

  try {
    socket = new WebSocket(url);
    socket.binaryType = "arraybuffer";
    socket.onopen = () => {
      if (!closed) send(codec.start(start), false);
    };
    socket.onmessage = (message) => {
      if (closed) return;
      try {
        const data = message.data;
        let bytes: Uint8Array;
        if (Object.prototype.toString.call(data) === "[object ArrayBuffer]") bytes = new Uint8Array(data as ArrayBuffer);
        else if (ArrayBuffer.isView(data)) bytes = new Uint8Array(data.buffer, data.byteOffset, data.byteLength);
        else throw new Error("voice stream expected a binary protobuf frame");

        const response = codec.decode(bytes);
        if (!readyNotified && codec.isReady(response)) {
          readyNotified = true;
          handlers.onReady();
        }
        const event = codec.event(response, start);
        if (event) {
          if (event.type === "final") serverCompleted = true;
          handlers.onEvent(event);
        }
      } catch (error) {
        fail(error);
      }
    };
    socket.onerror = () => fail(new Error("voice stream WebSocket failed"));
    socket.onclose = (event) => {
      notifyClose(event.code === 1000 && serverCompleted ? undefined : new Error("voice stream WebSocket closed before successful completion"));
    };
  } catch (error) {
    queueMicrotask(() => notifyClose(error));
  }

  return {
    isOpen: () => requestOpen && !closed && readyNotified && socket?.readyState === WebSocket.OPEN,
    sendAudio: (chunk) => {
      if (requestOpen && !closed && readyNotified) send(codec.audio(chunk), true);
    },
    end: () => {
      if (!requestOpen || closed || !readyNotified) return;
      if (!send(codec.end(), false)) return;
      requestOpen = false;
    },
    close: () => {
      if (closed) return;
      requestOpen = false;
      try {
        socket?.close(1000, "client cancelled");
      } finally {
        notifyClose();
      }
    },
  };
}
