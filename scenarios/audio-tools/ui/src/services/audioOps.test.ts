import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Mock } from "vitest";
import { AudioFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import { NormalizationMethod } from "@vrooli/proto-types/audio-tools/v1/audio/audio_pb";

const { calls } = vi.hoisted(() => ({ calls: {} as Record<string, Mock> }));

vi.mock("../api/client", async (importOriginal) => ({ ...(await importOriginal<typeof import("../api/client")>()), transport: {} }));
vi.mock("@connectrpc/connect", () => ({
  createClient: () => new Proxy({}, {
    get: (_target, method: string) => (...args: unknown[]) => calls[method]!(...args),
  }),
}));
vi.mock("./settings", () => ({
  normalizeConnectError: (error: unknown) => error instanceof Error ? error : new Error(String(error)),
}));

import { runAudioOp } from "./audioOps";

const bytes = new Uint8Array([4, 5]);
const file = (name = "voice.wav") => {
  const value = new File([bytes], name, { type: "audio/wav" });
  Object.defineProperty(value, "arrayBuffer", { value: () => Promise.resolve(bytes.buffer.slice(0)) });
  return value;
};

beforeEach(() => {
  Object.assign(calls, Object.fromEntries(
    ["transcode", "trim", "merge", "split", "fade", "volume", "normalize", "extractMetadata"]
      .map((method) => [method, vi.fn()]),
  ));
});

describe("runAudioOp", () => {
  it("maps transcode fields, truncates integer options, and names output by format", async () => {
    calls.transcode!.mockResolvedValue({ audio: bytes, contentType: "audio/mpeg" });
    const result = await runAudioOp({ id: "transcode", method: "transcode", values: {
      audio: file(), input_format: " WAV ", output_format: "mp3", sample_rate: "44100.9", channels: "2", bitrate: "128000",
    } });
    expect(calls.transcode).toHaveBeenCalledWith({ audio: bytes, inputFormat: AudioFormat.WAV,
      outputFormat: AudioFormat.MP3, sampleRate: 44100, channels: 2, bitrate: 128000 });
    expect(result).toEqual({ ok: true, data: { kind: "audio", bytes, contentType: "audio/mpeg", filename: "transcode.mp3" } });
  });

  it("encodes merge sources and rejects an empty source list before RPC", async () => {
    calls.merge!.mockResolvedValue({ audio: bytes, contentType: "audio/flac" });
    const result = await runAudioOp({ id: "merge", method: "merge", values: { sources: [file("a.wav"), file("b.wav")], format: "wav", output_format: "flac", crossfade_seconds: "0.5" } });
    expect(calls.merge).toHaveBeenCalledWith({ sources: [{ audio: bytes, format: AudioFormat.WAV }, { audio: bytes, format: AudioFormat.WAV }], outputFormat: AudioFormat.FLAC, crossfadeSeconds: 0.5 });
    expect(result).toMatchObject({ ok: true, data: { kind: "audio", filename: "merge.flac" } });
    calls.merge!.mockClear();
    await runAudioOp({ id: "merge", method: "merge", values: { sources: file("single.wav") } });
    expect(calls.merge).toHaveBeenCalledWith({ sources: [{ audio: bytes, format: AudioFormat.UNSPECIFIED }], outputFormat: AudioFormat.UNSPECIFIED, crossfadeSeconds: 0 });
    const empty = await runAudioOp({ id: "merge", method: "merge", values: { sources: [] } });
    expect(empty).toMatchObject({ ok: false, error: { message: "internal: merge requires at least one source file", code: "internal" } });
  });

  it("parses finite split boundaries and assigns stable chunk filenames", async () => {
    calls.split!.mockResolvedValue({ chunks: [
      { audio: bytes, contentType: "audio/ogg", startSeconds: 0, durationSeconds: 1.25 },
      { audio: new Uint8Array([8]), contentType: "audio/ogg", startSeconds: 1.25, durationSeconds: 0.5 },
    ] });
    const result = await runAudioOp({ id: "split", method: "split", values: { audio: file(), format: "wav", chunk_seconds: "bad", boundaries_seconds: " 1, nope, 2.5,,Infinity ", output_format: "ogg" } });
    expect(calls.split).toHaveBeenCalledWith({ audio: bytes, format: AudioFormat.WAV, chunkSeconds: 0, boundariesSeconds: [1, 2.5], outputFormat: AudioFormat.OGG });
    expect(result).toMatchObject({ ok: true, data: { kind: "chunks", contentType: "audio/*", chunks: [
      { contentType: "audio/ogg", startSeconds: 0, durationSeconds: 1.25, filename: "chunk-000.ogg" },
      { contentType: "audio/ogg", startSeconds: 1.25, durationSeconds: 0.5, filename: "chunk-001.ogg" },
    ] } });
  });

  it("normalizes missing metadata and returns a typed error for missing input files", async () => {
    calls.extractMetadata!.mockResolvedValue({});
    const metadata = await runAudioOp({ id: "extract_metadata", method: "extractMetadata", values: { audio: file(), format: "unknown" } });
    expect(calls.extractMetadata).toHaveBeenCalledWith({ audio: bytes, format: AudioFormat.UNSPECIFIED });
    expect(metadata).toEqual({ ok: true, data: { kind: "metadata", metadata: { durationSeconds: 0, sampleRate: 0, channels: 0, bitrate: 0, codec: "", format: 0, tags: {} } } });
    const missing = await runAudioOp({ id: "trim", method: "trim", values: {} });
    expect(missing).toMatchObject({ ok: false, error: { message: "internal: audio file is required", code: "internal" } });
    expect(calls.trim).not.toHaveBeenCalled();
  });

  it("maps normalization methods and converts RPC failures into Results", async () => {
    calls.normalize!.mockRejectedValue(new Error("service unavailable"));
    const result = await runAudioOp({ id: "normalize", method: "normalize", values: { audio: file(), format: "opus", method: "peak", target_lufs: "-14.2", output_format: "bad" } });
    expect(calls.normalize).toHaveBeenCalledWith({ audio: bytes, format: AudioFormat.OPUS, method: NormalizationMethod.PEAK, targetLufs: -14.2, outputFormat: AudioFormat.UNSPECIFIED });
    expect(result).toMatchObject({ ok: false, error: { message: "internal: service unavailable", code: "internal" } });
  });
});
