// Optional presentation analysis. Native <audio> streaming never depends on it.
export const MAX_AUDIO_ANALYSIS_BYTES = 24 * 1024 * 1024;
export const MAX_AUDIO_ANALYSIS_SECONDS = 600;
const PEAK_COUNT = 2048;

export function formatAudioTime(seconds: number, fractional = false): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "–:––";
  const total = Math.floor(seconds);
  const minutes = Math.floor(total / 60);
  const tail = String(total % 60).padStart(2, "0") + (fractional ? `.${String(Math.floor(seconds * 10) % 10)}` : "");
  return minutes < 60 ? `${String(minutes)}:${tail}` : `${String(Math.floor(minutes / 60))}:${String(minutes % 60).padStart(2, "0")}:${tail}`;
}

export function audioWindow(duration: number, zoom: number, start: number) {
  const span = duration > 0 ? duration / Math.max(1, zoom) : 0;
  const from = Math.max(0, Math.min(start, duration - span));
  return { start: from, end: from + span, span };
}

// Include every sample and every channel, preserving transients and quiet audio.
export function waveformPeaks(buffer: Pick<AudioBuffer, "length" | "numberOfChannels" | "getChannelData">): number[] {
  const count = Math.min(PEAK_COUNT, buffer.length);
  if (!count || !buffer.numberOfChannels) return [];
  const peaks = new Float32Array(count);
  for (let channel = 0; channel < buffer.numberOfChannels; channel++) {
    const samples = buffer.getChannelData(channel);
    for (let bin = 0; bin < count; bin++) {
      const end = Math.floor((bin + 1) * samples.length / count);
      let peak = 0;
      for (let i = Math.floor(bin * samples.length / count); i < end; i++) {
        peak = Math.max(peak, Math.abs(samples[i] ?? 0));
      }
      peaks[bin] = Math.max(peaks[bin] ?? 0, peak);
    }
  }
  const maximum = Math.max(...peaks);
  return Array.from(peaks, (peak) => maximum > 0 ? peak / maximum : 0);
}

export async function readAudioBytes(href: string, signal: AbortSignal): Promise<ArrayBuffer> {
  const response = await fetch(href, { signal, credentials: "same-origin" });
  if (!response.ok || !response.body) throw new Error("Audio analysis unavailable");
  if (Number(response.headers.get("Content-Length")) > MAX_AUDIO_ANALYSIS_BYTES) {
    await response.body.cancel();
    throw new Error("Audio analysis size limit");
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let size = 0;
  try {
    let chunk = await reader.read();
    while (!chunk.done) {
      const { value } = chunk;
      size += value.byteLength;
      if (signal.aborted || size > MAX_AUDIO_ANALYSIS_BYTES) throw new Error("Audio analysis cancelled or oversized");
      chunks.push(value);
      chunk = await reader.read();
    }
    signal.throwIfAborted();
  } finally {
    await reader.cancel().catch(() => {});
    reader.releaseLock();
  }
  const bytes = new Uint8Array(size);
  let offset = 0;
  for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength; }
  return bytes.buffer;
}

export async function decodeWaveform(bytes: ArrayBuffer): Promise<number[]> {
  if (typeof OfflineAudioContext === "undefined") return [];
  // Downsampling limits the retained PCM footprint, without opening an output
  // audio context (or changing native volume, routing, or Safari playback).
  const context = new OfflineAudioContext(1, 1, 8000);
  const buffer = await context.decodeAudioData(bytes);
  if (buffer.duration > MAX_AUDIO_ANALYSIS_SECONDS || buffer.numberOfChannels > 8) return [];
  return waveformPeaks(buffer);
}

export async function readAudioTags(bytes: ArrayBuffer, mimeType: string) {
  const { parseBuffer, selectCover } = await import("music-metadata");
  const { common } = await parseBuffer(new Uint8Array(bytes), { mimeType, size: bytes.byteLength }, { duration: false, skipPostHeaders: true });
  const cover = selectCover(common.picture);
  // No SVG/HTML or unbounded image payloads from untrusted metadata.
  const artwork = cover && cover.data.length <= 2 * 1024 * 1024 && /^image\/(jpeg|png|webp|gif)$/i.test(cover.format)
    ? new Blob([Uint8Array.from(cover.data)], { type: cover.format }) : null;
  return { title: common.title?.slice(0, 300), artist: common.artist?.slice(0, 300), artwork };
}
