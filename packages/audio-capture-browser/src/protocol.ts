export const STREAM_PROTOCOL_VERSION = 2;
export const ATV2_MAGIC = "ATV2";
/** 4-byte magic + sequence + start/end sample cursors + SHA-256 digest. */
export const V2_AUDIO_HEADER_BYTES = 60;

export interface AudioFrame {
  sequence: bigint;
  startSample: bigint;
  endSample: bigint;
  audio: ArrayBufferView;
  sha256: ArrayBufferView;
}

/** Encodes the single browser-to-server v2 binary frame shape: ATV2 + cursor + PCM. */
export function encodeAudioFrame(frame: AudioFrame): ArrayBuffer {
  if (frame.endSample < frame.startSample || frame.audio.byteLength === 0) {
    throw new Error("audio frame must contain a non-empty non-negative sample range");
  }
  const bytes = new Uint8Array(frame.audio.buffer, frame.audio.byteOffset, frame.audio.byteLength);
  const digest = new Uint8Array(frame.sha256.buffer, frame.sha256.byteOffset, frame.sha256.byteLength);
  if (digest.byteLength !== 32) throw new Error("audio frame requires a SHA-256 digest");
  const output = new ArrayBuffer(V2_AUDIO_HEADER_BYTES + bytes.byteLength);
  const view = new DataView(output);
  view.setUint8(0, 0x41);
  view.setUint8(1, 0x54);
  view.setUint8(2, 0x56);
  view.setUint8(3, 0x32);
  view.setBigUint64(4, frame.sequence, false);
  view.setBigInt64(12, frame.startSample, false);
  view.setBigInt64(20, frame.endSample, false);
  new Uint8Array(output, 28, 32).set(digest);
  new Uint8Array(output, V2_AUDIO_HEADER_BYTES).set(bytes);
  return output;
}

/** SHA-256 for chunk identity. The protocol fails visibly if unavailable. */
export async function digestAudio(audio: ArrayBuffer): Promise<ArrayBuffer> {
  if (globalThis.crypto?.subtle) {
    try {
      return await globalThis.crypto.subtle.digest("SHA-256", audio);
    } catch {
      // Cross-realm ArrayBuffers (notably jsdom) are rejected by some Web
      // Crypto implementations despite being valid binary input. Fall through
      // to the exact same SHA-256 algorithm below.
    }
  }
  // jsdom and some constrained browser contexts do not expose SubtleCrypto.
  // Keep the protocol identity intact with a small, dependency-free SHA-256
  // implementation rather than silently omitting the digest.
  return sha256Fallback(audio);
}

const SHA256_K = new Uint32Array([
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
  0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
  0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
  0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
  0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
]);

function rotateRight(value: number, bits: number): number {
  return (value >>> bits) | (value << (32 - bits));
}

function sha256Fallback(input: ArrayBuffer): ArrayBuffer {
  const source = new Uint8Array(input);
  const padding = (64 - ((source.length + 9) % 64)) % 64;
  const bytes = new Uint8Array(source.length + 9 + padding);
  bytes.set(source);
  bytes[source.length] = 0x80;
  new DataView(bytes.buffer).setBigUint64(bytes.length - 8, BigInt(source.length) * 8n, false);
  const state = new Uint32Array([0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19]);
  const words = new Uint32Array(64);
  const view = new DataView(bytes.buffer);
  for (let offset = 0; offset < bytes.length; offset += 64) {
    for (let i = 0; i < 16; i++) words[i] = view.getUint32(offset + i * 4, false);
    for (let i = 16; i < 64; i++) {
      const a = words[i - 15] ?? 0;
      const b = words[i - 2] ?? 0;
      words[i] = (rotateRight(a, 7) ^ rotateRight(a, 18) ^ (a >>> 3)) + (words[i - 16] ?? 0) + (rotateRight(b, 17) ^ rotateRight(b, 19) ^ (b >>> 10)) + (words[i - 7] ?? 0);
    }
    let a = state[0] ?? 0;
    let b = state[1] ?? 0;
    let c = state[2] ?? 0;
    let d = state[3] ?? 0;
    let e = state[4] ?? 0;
    let f = state[5] ?? 0;
    let g = state[6] ?? 0;
    let h = state[7] ?? 0;
    for (let i = 0; i < 64; i++) {
      const s1 = rotateRight(e, 6) ^ rotateRight(e, 11) ^ rotateRight(e, 25);
      const choose = (e & f) ^ (~e & g);
      const t1 = (h + s1 + choose + (SHA256_K[i] ?? 0) + (words[i] ?? 0)) >>> 0;
      const s0 = rotateRight(a, 2) ^ rotateRight(a, 13) ^ rotateRight(a, 22);
      const majority = (a & b) ^ (a & c) ^ (b & c);
      const t2 = (s0 + majority) >>> 0;
      h = g; g = f; f = e; e = (d + t1) >>> 0; d = c; c = b; b = a; a = (t1 + t2) >>> 0;
    }
    state[0] = ((state[0] ?? 0) + a) >>> 0;
    state[1] = ((state[1] ?? 0) + b) >>> 0;
    state[2] = ((state[2] ?? 0) + c) >>> 0;
    state[3] = ((state[3] ?? 0) + d) >>> 0;
    state[4] = ((state[4] ?? 0) + e) >>> 0;
    state[5] = ((state[5] ?? 0) + f) >>> 0;
    state[6] = ((state[6] ?? 0) + g) >>> 0;
    state[7] = ((state[7] ?? 0) + h) >>> 0;
  }
  const digest = new ArrayBuffer(32);
  const digestView = new DataView(digest);
  for (let i = 0; i < state.length; i++) digestView.setUint32(i * 4, state[i] ?? 0, false);
  return digest;
}

export function newSessionIdentity(): string {
  const cryptoAPI = globalThis.crypto;
  if (typeof cryptoAPI?.randomUUID === "function") return cryptoAPI.randomUUID();
  if (typeof cryptoAPI?.getRandomValues !== "function") {
    throw new Error("Secure random identity generation is unavailable");
  }
  const bytes = new Uint8Array(16);
  cryptoAPI.getRandomValues(bytes);
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}
