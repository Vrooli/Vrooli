import { beforeEach, describe, expect, it, vi } from "vitest";
import type { TranscribeStreamEvent } from "@vrooli/proto-types/web-console/v1/audio_runtime/audio_runtime_pb";

const streamHarness = vi.hoisted(() => ({
  codec: undefined as any,
  requests: [] as any[],
}));

vi.mock("@connectrpc/connect", () => ({ createClient: () => ({}) }));
vi.mock("../../api/client", () => ({ transport: {}, API_BASE: "http://test" }));
vi.mock("@vrooli/audio-capture-browser", async () => {
  const { safeSampleNumber } = await import("../../../../../../packages/audio-capture-browser/src/streamMessages");
  return {
    openConnectVoiceStream: (_client: unknown, codec: any, _start: unknown, handlers: any) => {
      streamHarness.codec = codec;
      return {
        sendAudio: (chunk: unknown) => streamHarness.requests.push(codec.audio(chunk)),
        end: () => undefined,
        close: () => undefined,
        isOpen: () => true,
        emit: (event: unknown, start: unknown) => handlers.onEvent(codec.event(event, start)),
      };
    },
    safeSampleNumber,
  };
});

function makeEvent(endSample: bigint | undefined): TranscribeStreamEvent {
  return {
    event: {
      case: "segment",
      value: {
        text: "recognized",
        segmentId: "segment-1",
        startSample: 0n,
        endSample,
        speakerPolicy: "applied",
      },
    },
  } as unknown as TranscribeStreamEvent;
}

describe("Web Console voice codec endSample mapping", () => {
  beforeEach(() => {
    streamHarness.codec = undefined;
    streamHarness.requests = [];
  });

  it("preserves bigint endSample on outgoing audio chunks", async () => {
    const { openVoiceStream } = await import("./voice");
    const connection = openVoiceStream({ language: "en", sessionId: "codec-test", resumeToken: "resume", protocolVersion: 2, inputFormat: "pcm_s16le", sampleRateHz: 16_000, generation: 1n }, {
      onEvent: () => undefined,
      onReady: () => undefined,
      onClose: () => undefined,
    });
    const endSample = 9_007_199_254_741_000n;

    connection.sendAudio({ sequence: 4n, startSample: endSample - 1_600n, endSample, audio: new Uint8Array([1, 2]), sha256: new Uint8Array(32) });

    expect(streamHarness.requests[0]?.payload.case).toBe("audioChunk");
    expect(streamHarness.requests[0]?.payload.value.endSample).toBe(endSample);
  });

  it("maps valid response endSample to a consumer number and rejects absent or unsafe values", async () => {
    const { openVoiceStream } = await import("./voice");
    const events: unknown[] = [];
    const start = { language: "en", sessionId: "codec-response-test", resumeToken: "resume", protocolVersion: 2 as const, inputFormat: "pcm_s16le" as const, sampleRateHz: 16_000, generation: 1n } as const;
    const connection = openVoiceStream(start, {
      onEvent: (event) => events.push(event),
      onReady: () => undefined,
      onClose: () => undefined,
    });
    const emit = (connection as typeof connection & { emit: (event: unknown, start: unknown) => void }).emit;

    emit(makeEvent(32_000n), start);
    emit(makeEvent(undefined), start);
    emit(makeEvent(-1n), start);
    emit(makeEvent(BigInt(Number.MAX_SAFE_INTEGER) + 1n), start);

    expect((events[0] as { endSample?: number }).endSample).toBe(32_000);
    expect((events[1] as { endSample?: number }).endSample).toBeUndefined();
    expect((events[2] as { endSample?: number }).endSample).toBeUndefined();
    expect((events[3] as { endSample?: number }).endSample).toBeUndefined();
  });
});
