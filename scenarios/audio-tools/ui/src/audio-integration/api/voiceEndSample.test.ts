import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { ProviderTier } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import type { VoiceAudioChunk, VoiceStreamHandlers, VoiceStreamStart, VoiceWebSocketCodec } from "@vrooli/audio-capture-browser";
import { StreamSegmentSchema, StreamSessionStatusSchema, TranscribeStreamEventSchema, type TranscribeStreamEvent } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";
import type { TranscribeStreamRequest } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";
import { createVoiceApi } from "./voice";

type TestCodec = VoiceWebSocketCodec<TranscribeStreamRequest, TranscribeStreamEvent>;
type StreamHarness = {
  codec?: TestCodec;
  start?: VoiceStreamStart;
  handlers?: VoiceStreamHandlers;
  requests: TranscribeStreamRequest[];
};

const streamHarness = vi.hoisted<StreamHarness>(() => ({
  requests: [],
}));

// Keep this consumer codec test independent of stale installed package output;
// the adapter under test still runs its real protobuf request/event mappers.
vi.mock("@vrooli/audio-capture-browser", async () => {
  const [{ safeSampleNumber }, { providerTierLabel }] = await Promise.all([
    import("../../../../../../packages/audio-capture-browser/src/streamMessages"),
    import("../../../../../../packages/audio-capture-browser/src/api/protomap"),
  ]);
  return {
    openWebSocketVoiceStream: (_url: string, codec: TestCodec, start: VoiceStreamStart, handlers: VoiceStreamHandlers) => {
      streamHarness.codec = codec;
      streamHarness.start = start;
      streamHarness.handlers = handlers;
      return {
        sendAudio: (chunk: VoiceAudioChunk) => streamHarness.requests.push(codec.audio(chunk)),
        end: () => undefined,
        close: () => undefined,
        isOpen: () => true,
      };
    },
    safeSampleNumber,
    providerTierLabel,
  };
});

function makeEvent(endSample: bigint | undefined): TranscribeStreamEvent {
  const segment = create(StreamSegmentSchema, {
    text: "recognized",
    segmentId: "segment-1",
    startSample: 0n,
    endSample: endSample ?? 0n,
    speakerPolicy: "applied",
    modelId: "model-test",
    providerTier: ProviderTier.LOCAL,
  });
  if (endSample === undefined) Reflect.set(segment, "endSample", undefined);
  return create(TranscribeStreamEventSchema, { event: { case: "segment", value: segment } });
}

function setup() {
  const api = createVoiceApi({
    baseUrl: "http://test",
    stt: {} as never,
    sttAdmin: {} as never,
    tts: {} as never,
    summarize: {} as never,
  });
  const events: unknown[] = [];
  const start = {
    language: "en",
    resumeToken: "",
    protocolVersion: 2,
    inputFormat: "pcm_s16le",
    sampleRateHz: 16_000,
    sessionId: "codec-test",
    generation: 1n,
  } as const;
  const connection = api.openVoiceStream(start, {
    onEvent: (event) => events.push(event),
    onReady: () => undefined,
    onClose: () => undefined,
  });
  return { connection, events, start };
}

describe("Audio Tools voice codec endSample mapping", () => {
  beforeEach(() => {
    streamHarness.codec = undefined;
    streamHarness.start = undefined;
    streamHarness.handlers = undefined;
    streamHarness.requests = [];
  });

  function activeStream(): { codec: TestCodec; start: VoiceStreamStart; handlers: VoiceStreamHandlers } {
    const { codec, start, handlers } = streamHarness;
    if (!codec || !start || !handlers) throw new Error("voice stream harness was not initialized");
    return { codec, start, handlers };
  }

  it("preserves bigint endSample on outgoing audio chunks", () => {
    const { connection } = setup();
    const endSample = 9_007_199_254_741_000n;

    connection.sendAudio({
      sequence: 4n,
      startSample: endSample - 1_600n,
      endSample,
      audio: new Uint8Array([1, 2]),
      sha256: new Uint8Array(32),
    });

    const request = streamHarness.requests[0];
    expect(request?.payload.case).toBe("audioChunk");
    if (request?.payload.case !== "audioChunk") throw new Error("expected audio chunk request");
    expect(request.payload.value.endSample).toBe(endSample);
  });

  it("maps a valid response endSample to the consumer number field", () => {
    const { events, start } = setup();

    const { codec, handlers } = activeStream();
    const event = codec.event(makeEvent(32_000n), start);
    if (event) handlers.onEvent(event);

    expect(events).toContainEqual(expect.objectContaining({ type: "segment-final", endSample: 32_000, providerTier: "local" }));
  });

  it("publishes selected provider and model identity on the ready status before segments", () => {
    const { events, start } = setup();
    const { codec, handlers } = activeStream();
    const status = create(StreamSessionStatusSchema, {
      state: "ready", providerId: "catalog-provider", modelId: "catalog-model", providerTier: ProviderTier.LOCAL,
    });
    const event = codec.event(create(TranscribeStreamEventSchema, { event: { case: "sessionStatus", value: status } }), start);
    if (event) handlers.onEvent(event);
    expect(events).toContainEqual(expect.objectContaining({
      type: "status", providerId: "catalog-provider", modelId: "catalog-model", providerTier: "local",
    }));
  });

  it.each([
    ["missing", undefined],
    ["negative", -1n],
    ["outside the safe integer range", BigInt(Number.MAX_SAFE_INTEGER) + 1n],
  ])("omits %s response endSample", (_label, endSample) => {
    const { events, start } = setup();

    const { codec, handlers } = activeStream();
    const event = codec.event(makeEvent(endSample), start);
    if (event) handlers.onEvent(event);

    expect(events).toContainEqual(expect.objectContaining({ type: "segment-final" }));
    expect((events[0] as { endSample?: number }).endSample).toBeUndefined();
  });
});
