// audioWav — extract raw signed-16-bit little-endian PCM from a WAV blob.
//
// VoiceStreamProvider.getLastTurnAudio() retains the completed turn as a
// canonical 16 kHz mono WAV (it wraps the captured PCM via
// encodeWavFromPcm16). The corpus stores raw PCM with a `format` hint, so we
// strip the RIFF container back down to its `data` chunk and read the real
// sample rate from the `fmt ` chunk. Parsing the chunk table (rather than
// assuming a fixed 44-byte header) keeps this robust to extra chunks.

export interface ExtractedPcm {
  pcm: Uint8Array;
  sampleRateHz: number;
}

function readTag(view: DataView, offset: number): string {
  return (
    String.fromCharCode(view.getUint8(offset)) +
    String.fromCharCode(view.getUint8(offset + 1)) +
    String.fromCharCode(view.getUint8(offset + 2)) +
    String.fromCharCode(view.getUint8(offset + 3))
  );
}

export function extractPcm16FromWav(buffer: ArrayBuffer): ExtractedPcm {
  if (buffer.byteLength < 12) {
    throw new Error("buffer too small to be a WAV file");
  }
  const view = new DataView(buffer);
  if (readTag(view, 0) !== "RIFF" || readTag(view, 8) !== "WAVE") {
    throw new Error("not a RIFF/WAVE buffer");
  }

  let sampleRateHz = 16_000;
  let dataStart = -1;
  let dataLen = 0;
  let offset = 12;

  while (offset + 8 <= buffer.byteLength) {
    const id = readTag(view, offset);
    const size = view.getUint32(offset + 4, true);
    const body = offset + 8;
    if (id === "fmt " && body + 8 <= buffer.byteLength) {
      // fmt layout: audioFormat(2) numChannels(2) sampleRate(4) ...
      sampleRateHz = view.getUint32(body + 4, true);
    } else if (id === "data") {
      dataStart = body;
      dataLen = size;
    }
    // Chunks are word-aligned: an odd-sized chunk is followed by a pad byte.
    offset = body + size + (size % 2);
  }

  if (dataStart < 0) {
    throw new Error("WAV buffer has no data chunk");
  }
  const end = Math.min(dataStart + dataLen, buffer.byteLength);
  return { pcm: new Uint8Array(buffer.slice(dataStart, end)), sampleRateHz };
}

/** Derive duration in ms from a signed-16-bit mono PCM byte length. */
export function pcm16DurationMs(byteLength: number, sampleRateHz: number): number {
  if (sampleRateHz <= 0) return 0;
  const samples = Math.floor(byteLength / 2);
  return Math.round((samples / sampleRateHz) * 1000);
}
