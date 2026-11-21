import { createClient } from "@connectrpc/connect";
import { AudioFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import {
  AudioProcessingService,
  NormalizationMethod,
  type AudioMetadata,
} from "@vrooli/proto-types/audio-tools/v1/audio/audio_pb";
import { transport } from "../api/client";
import { tryCall, type Result } from "./result";
import { normalizeConnectError } from "./settings";
import type { AudioOpId, AudioOpMethod } from "../features/diagnostics/audioOps.generated";

const client = createClient(AudioProcessingService, transport);

export type AudioOpValue = string | File | File[] | boolean | undefined;
export type AudioOpValues = Record<string, AudioOpValue>;

export interface AudioOpChunk {
  bytes: Uint8Array;
  contentType: string;
  startSeconds: number;
  durationSeconds: number;
  filename: string;
}

export interface AudioOpMetadata {
  durationSeconds: number;
  sampleRate: number;
  channels: number;
  bitrate: number;
  codec: string;
  format: number;
  tags: Record<string, string>;
}

export type AudioOpOutput =
  | { kind: "audio"; bytes: Uint8Array; contentType: string; filename: string }
  | { kind: "chunks"; contentType: string; chunks: AudioOpChunk[] }
  | { kind: "metadata"; metadata: AudioOpMetadata };

export interface RunAudioOpInput {
  id: AudioOpId;
  method: AudioOpMethod;
  values: AudioOpValues;
}

/**
 * runAudioOp maps the generic panel's field values onto the matching
 * AudioProcessingService Connect method and normalizes every response into one
 * of the three result shapes the panel knows how to render.
 */
export async function runAudioOp(input: RunAudioOpInput): Promise<Result<AudioOpOutput>> {
  return tryCall(async () => {
    try {
      return await invoke(input);
    } catch (e) {
      throw normalizeConnectError(e);
    }
  });
}

async function invoke({ id, method, values }: RunAudioOpInput): Promise<AudioOpOutput> {
  switch (method) {
    case "transcode": {
      const res = await client.transcode({
        audio: await fileBytes(values, "audio"),
        inputFormat: audioFormat(values.input_format),
        outputFormat: audioFormat(values.output_format),
        sampleRate: integer(values.sample_rate),
        channels: integer(values.channels),
        bitrate: integer(values.bitrate),
      });
      return audioOutput(id, res.audio, res.contentType, values.output_format);
    }
    case "trim": {
      const res = await client.trim({
        audio: await fileBytes(values, "audio"),
        format: audioFormat(values.format),
        startSeconds: numberValue(values.start_seconds),
        endSeconds: numberValue(values.end_seconds),
      });
      return audioOutput(id, res.audio, res.contentType, values.format);
    }
    case "merge": {
      const sources = sourceFiles(values, "sources");
      if (sources.length === 0) throw new Error("merge requires at least one source file");
      const format = audioFormat(values.format);
      const res = await client.merge({
        sources: await Promise.all(
          sources.map(async (file) => ({ audio: await bytesOf(file), format })),
        ),
        outputFormat: audioFormat(values.output_format),
        crossfadeSeconds: numberValue(values.crossfade_seconds),
      });
      return audioOutput(id, res.audio, res.contentType, values.output_format);
    }
    case "split": {
      const res = await client.split({
        audio: await fileBytes(values, "audio"),
        format: audioFormat(values.format),
        chunkSeconds: numberValue(values.chunk_seconds),
        boundariesSeconds: numberList(values.boundaries_seconds),
        outputFormat: audioFormat(values.output_format),
      });
      const ext = formatExtension(values.output_format);
      return {
        kind: "chunks",
        contentType: "audio/*",
        chunks: res.chunks.map((chunk, index) => ({
          bytes: chunk.audio,
          contentType: chunk.contentType,
          startSeconds: chunk.startSeconds,
          durationSeconds: chunk.durationSeconds,
          filename: `chunk-${String(index).padStart(3, "0")}.${ext}`,
        })),
      };
    }
    case "fade": {
      const res = await client.fade({
        audio: await fileBytes(values, "audio"),
        format: audioFormat(values.format),
        fadeInSeconds: numberValue(values.fade_in_seconds),
        fadeOutSeconds: numberValue(values.fade_out_seconds),
        outputFormat: audioFormat(values.output_format),
      });
      return audioOutput(id, res.audio, res.contentType, values.output_format);
    }
    case "volume": {
      const res = await client.volume({
        audio: await fileBytes(values, "audio"),
        format: audioFormat(values.format),
        gainDb: numberValue(values.gain_db),
        outputFormat: audioFormat(values.output_format),
      });
      return audioOutput(id, res.audio, res.contentType, values.output_format);
    }
    case "normalize": {
      const res = await client.normalize({
        audio: await fileBytes(values, "audio"),
        format: audioFormat(values.format),
        method: normalizationMethod(values.method),
        targetLufs: numberValue(values.target_lufs),
        outputFormat: audioFormat(values.output_format),
      });
      return audioOutput(id, res.audio, res.contentType, values.output_format);
    }
    case "extractMetadata": {
      const res = await client.extractMetadata({
        audio: await fileBytes(values, "audio"),
        format: audioFormat(values.format),
      });
      return { kind: "metadata", metadata: metadataView(res.metadata) };
    }
  }
}

function audioOutput(
  id: AudioOpId,
  bytes: Uint8Array,
  contentType: string,
  outputFormat: AudioOpValue,
): AudioOpOutput {
  return {
    kind: "audio",
    bytes,
    contentType,
    filename: `${id}.${formatExtension(outputFormat)}`,
  };
}

function metadataView(metadata: AudioMetadata | undefined): AudioOpMetadata {
  return {
    durationSeconds: metadata?.durationSeconds ?? 0,
    sampleRate: metadata?.sampleRate ?? 0,
    channels: metadata?.channels ?? 0,
    bitrate: metadata?.bitrate ?? 0,
    codec: metadata?.codec ?? "",
    format: metadata?.format ?? 0,
    tags: metadata?.tags ?? {},
  };
}

function audioFormat(value: AudioOpValue): AudioFormat {
  switch (textOf(value).trim().toLowerCase()) {
    case "wav":
      return AudioFormat.WAV;
    case "mp3":
      return AudioFormat.MP3;
    case "flac":
      return AudioFormat.FLAC;
    case "ogg":
      return AudioFormat.OGG;
    case "aac":
      return AudioFormat.AAC;
    case "webm":
      return AudioFormat.WEBM;
    case "opus":
      return AudioFormat.OPUS;
    case "pcm_s16le":
      return AudioFormat.PCM_S16LE;
    default:
      return AudioFormat.UNSPECIFIED;
  }
}

function normalizationMethod(value: AudioOpValue): NormalizationMethod {
  switch (textOf(value).trim().toLowerCase()) {
    case "ebu_r128":
      return NormalizationMethod.EBU_R128;
    case "rms":
      return NormalizationMethod.RMS;
    case "peak":
      return NormalizationMethod.PEAK;
    default:
      return NormalizationMethod.UNSPECIFIED;
  }
}

function formatExtension(value: AudioOpValue): string {
  const raw = textOf(value).trim().toLowerCase();
  return ["wav", "mp3", "flac", "ogg", "aac", "webm", "opus"].includes(raw) ? raw : "wav";
}

function textOf(value: AudioOpValue): string {
  if (typeof value === "string") return value;
  if (typeof value === "boolean") return String(value);
  return "";
}

function numberValue(value: AudioOpValue): number {
  const raw = textOf(value).trim();
  if (raw === "") return 0;
  const parsed = Number(raw);
  return Number.isFinite(parsed) ? parsed : 0;
}

function integer(value: AudioOpValue): number {
  return Math.trunc(numberValue(value));
}

function numberList(value: AudioOpValue): number[] {
  return textOf(value)
    .split(",")
    .map((part) => part.trim())
    .filter((part) => part !== "")
    .map((part) => Number(part))
    .filter((part) => Number.isFinite(part));
}

function fileValue(values: AudioOpValues, name: string): File {
  const value = values[name];
  if (value instanceof File) return value;
  if (Array.isArray(value) && value[0] instanceof File) return value[0];
  throw new Error(`${name} file is required`);
}

async function fileBytes(values: AudioOpValues, name: string): Promise<Uint8Array> {
  return bytesOf(fileValue(values, name));
}

function sourceFiles(values: AudioOpValues, name: string): File[] {
  const value = values[name];
  if (Array.isArray(value)) return value.filter((item): item is File => item instanceof File);
  if (value instanceof File) return [value];
  return [];
}

async function bytesOf(file: File): Promise<Uint8Array> {
  return new Uint8Array(await file.arrayBuffer());
}
