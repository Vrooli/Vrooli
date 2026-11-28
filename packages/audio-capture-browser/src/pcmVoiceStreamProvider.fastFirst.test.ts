import { afterEach, describe, expect, it, vi } from "vitest";
import { FakeVoiceTransport, stubCryptoWithoutSubtle } from "./fakeVoiceTransport";
import { MemoryTurnJournalStore, TurnJournal } from "./turnJournal";
import { PcmVoiceStreamProvider } from "./pcmVoiceStreamProvider";
import { forgetUnfinishedSession } from "./sessionIdentity";

describe("PCM first wire batch", () => {
  let provider: PcmVoiceStreamProvider | undefined;

  afterEach(() => {
    provider?.dispose();
    provider = undefined;
    forgetUnfinishedSession();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("sends the 20 ms first batch, then preserves contiguous 50 ms chunks", async () => {
    vi.useFakeTimers();
    stubCryptoWithoutSubtle();
    FakeVoiceTransport.instances = [];
    FakeVoiceTransport.autoOpen = true;
    let onFrame: ((samples: Float32Array, sampleRate: number) => void) | undefined;
    const mediaStream = { getTracks: () => [{ readyState: "live", stop: vi.fn() }] } as unknown as MediaStream;
    provider = new PcmVoiceStreamProvider({
      getUserMedia: async () => mediaStream,
      transport: new FakeVoiceTransport(),
      captureFactory: async (_stream, captureFrame) => {
        onFrame = captureFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => new TurnJournal(new MemoryTurnJournalStore(), "first-batch", 0n, 16 * 1024 * 1024, "memory"),
      wireBatchSamples: 800,
      fastFirstWireBatch: true,
    });

    await provider.start();
    for (let index = 0; index < 12; index += 1) await Promise.resolve();
    onFrame?.(new Float32Array(1_920).fill(0.5), 16_000);
    for (let index = 0; index < 12; index += 1) await Promise.resolve();
    await vi.advanceTimersByTimeAsync(100);
    for (let index = 0; index < 12; index += 1) await Promise.resolve();

    const sent = FakeVoiceTransport.instances[0]!.sendAudio.mock.calls.map(([chunk]) => chunk);
    expect(sent.map(({ sequence, startSample, endSample, audio }) => ({ sequence, startSample, endSample, bytes: audio.byteLength }))).toEqual([
      { sequence: 0n, startSample: 0n, endSample: 320n, bytes: 640 },
      { sequence: 1n, startSample: 320n, endSample: 1_120n, bytes: 1_600 },
      { sequence: 2n, startSample: 1_120n, endSample: 1_920n, bytes: 1_600 },
    ]);
  });

  it("keeps the configured opening batch for engines without the opt-in", async () => {
    vi.useFakeTimers();
    stubCryptoWithoutSubtle();
    FakeVoiceTransport.instances = [];
    FakeVoiceTransport.autoOpen = true;
    let onFrame: ((samples: Float32Array, sampleRate: number) => void) | undefined;
    const mediaStream = { getTracks: () => [{ readyState: "live", stop: vi.fn() }] } as unknown as MediaStream;
    provider = new PcmVoiceStreamProvider({
      getUserMedia: async () => mediaStream,
      transport: new FakeVoiceTransport(),
      captureFactory: async (_stream, captureFrame) => {
        onFrame = captureFrame;
        return { stop: vi.fn() };
      },
      journalFactory: () => new TurnJournal(new MemoryTurnJournalStore(), "normal-batch", 0n, 16 * 1024 * 1024, "memory"),
      wireBatchSamples: 800,
      fastFirstWireBatch: false,
    });

    await provider.start();
    for (let index = 0; index < 12; index += 1) await Promise.resolve();
    onFrame?.(new Float32Array(800).fill(0.5), 16_000);
    for (let index = 0; index < 12; index += 1) await Promise.resolve();

    const sent = FakeVoiceTransport.instances[0]!.sendAudio.mock.calls.map(([chunk]) => chunk);
    expect(sent).toHaveLength(1);
    expect(sent[0]).toMatchObject({ sequence: 0n, startSample: 0n, endSample: 800n });
    expect(sent[0]!.audio.byteLength).toBe(1_600);
  });
});
