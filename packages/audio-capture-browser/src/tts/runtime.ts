import type { TTSVoiceInfo } from "./types";

export interface TTSCacheControl {
  eventId?: string;
  version?: "active" | "original";
  chunkIndex?: number;
}

export interface TTSSynthesisMetrics {
  requestId: string;
  synthStartMs: number;
  totalChars: number;
}

export type TTSRuntime = {
  synthesizeTTS(input: string, voice?: string, speed?: number, signal?: AbortSignal, cache?: TTSCacheControl): Promise<Blob>;
  synthesizeTTSWithMetrics?(input: string, voice?: string, speed?: number, signal?: AbortSignal, cache?: TTSCacheControl): Promise<{ blob: Blob; metrics: TTSSynthesisMetrics }>;
  fetchCachedTTS?(eventId: string, voice: string, speed: number, version?: "active" | "original", signal?: AbortSignal, chunkIndex?: number): Promise<Blob | null>;
  getTTSVoices?(): Promise<TTSVoiceInfo[]>;
  reportTTSPlayStart?(metrics: TTSSynthesisMetrics): void;
};
