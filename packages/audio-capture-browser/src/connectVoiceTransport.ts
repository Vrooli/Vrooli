import { PushableAsyncIterable } from "./pushableAsyncIterable";
import type {
  VoiceAudioChunk,
  VoiceStreamConnection,
  VoiceStreamHandlers,
  VoiceStreamStart,
} from "./transport";
import type { StreamMessage } from "./streamMessages";

export interface BidiConnectClient<Request, Response> {
  transcribeStream(
    requests: AsyncIterable<Request>,
    options: { signal: AbortSignal },
  ): AsyncIterable<Response>;
}

/** Generated-message operations required to adapt a Connect STT bidi stream. */
export interface VoiceConnectCodec<Request, Response> {
  start(value: VoiceStreamStart): Request;
  audio(value: VoiceAudioChunk): Request;
  end(): Request;
  isStart(value: Request): boolean;
  event(value: Response, start: VoiceStreamStart): StreamMessage | null;
}

/**
 * Adapts the generated Connect client to the shared browser provider contract.
 * This owns the request queue and response pump, leaving only protobuf mapping
 * in each scenario adapter.
 */
export function openConnectVoiceStream<Request, Response>(
  client: BidiConnectClient<Request, Response>,
  codec: VoiceConnectCodec<Request, Response>,
  start: VoiceStreamStart,
  handlers: VoiceStreamHandlers,
): VoiceStreamConnection {
  let readyNotified = false;
  let requestOpen = true;
  let closed = false;
  const abort = new AbortController();
  const queue = new PushableAsyncIterable<Request>((request) => {
    if (!readyNotified && codec.isStart(request)) {
      readyNotified = true;
      queueMicrotask(() => handlers.onReady());
    }
  });
  queue.push(codec.start(start));

  const notifyClose = (error?: unknown) => {
    if (closed) return;
    closed = true;
    requestOpen = false;
    queue.finish();
    handlers.onClose(error);
  };

  void (async () => {
    try {
      for await (const response of client.transcribeStream(queue, { signal: abort.signal })) {
        const event = codec.event(response, start);
        if (event) handlers.onEvent(event);
      }
      notifyClose();
    } catch (error) {
      if (!abort.signal.aborted) notifyClose(error);
    }
  })();

  return {
    isOpen: () => requestOpen && !closed,
    sendAudio: (chunk) => {
      if (requestOpen && !closed) queue.push(codec.audio(chunk));
    },
    end: () => {
      if (!requestOpen || closed) return;
      requestOpen = false;
      queue.push(codec.end());
      queue.finish();
    },
    close: () => {
      if (closed) return;
      requestOpen = false;
      queue.finish();
      abort.abort();
      notifyClose();
    },
  };
}
