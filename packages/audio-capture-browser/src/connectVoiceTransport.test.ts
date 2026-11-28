import { describe, expect, it, vi } from "vitest";
import { openConnectVoiceStream, type VoiceConnectCodec } from "./connectVoiceTransport";
import type { VoiceStreamHandlers, VoiceStreamStart } from "./transport";

type Request = { kind: "start" | "audio" | "end" };
type Response = { kind: "accepted" };

const start: VoiceStreamStart = {
  language: "en",
  sessionId: "session-test",
  resumeToken: "resume-test",
  protocolVersion: 2,
  generation: 0n,
  inputFormat: "pcm_s16le",
  sampleRateHz: 16_000,
};

const codec: VoiceConnectCodec<Request, Response> = {
  start: () => ({ kind: "start" }),
  audio: () => ({ kind: "audio" }),
  end: () => ({ kind: "end" }),
  isReady: (response) => response.kind === "accepted",
  event: () => null,
};

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

async function settle(): Promise<void> {
  for (let i = 0; i < 8; i += 1) await Promise.resolve();
}

describe("openConnectVoiceStream readiness", () => {
  it("waits for a server response before reporting ready", async () => {
    const firstResponse = deferred<void>();
    const handlers: VoiceStreamHandlers = {
      onReady: vi.fn(),
      onEvent: vi.fn(),
      onClose: vi.fn(),
    };
    const client = {
      async *transcribeStream(_requests: AsyncIterable<Request>, options: { signal: AbortSignal }) {
        await firstResponse.promise;
        yield { kind: "accepted" };
        await new Promise<void>((resolve) => options.signal.addEventListener("abort", () => resolve(), { once: true }));
      },
    };

    const connection = openConnectVoiceStream(client, codec, start, handlers);
    await settle();

    expect(connection.isOpen()).toBe(false);
    expect(handlers.onReady).not.toHaveBeenCalled();

    firstResponse.resolve();
    await settle();
    expect(connection.isOpen()).toBe(true);
    expect(handlers.onReady).toHaveBeenCalledOnce();
    expect(handlers.onEvent).not.toHaveBeenCalled();
    connection.close();
    await settle();
  });

  it("preserves a transport failure without reporting ready", async () => {
    const failure = new Error("The fetch API does not support streaming request bodies");
    const handlers: VoiceStreamHandlers = {
      onReady: vi.fn(),
      onEvent: vi.fn(),
      onClose: vi.fn(),
    };
    const client = {
      async *transcribeStream() {
        throw failure;
      },
    };

    openConnectVoiceStream(client, codec, start, handlers);
    await settle();

    expect(handlers.onReady).not.toHaveBeenCalled();
    expect(handlers.onClose).toHaveBeenCalledWith(failure);
  });
});
