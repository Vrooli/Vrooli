import { createClient } from "@connectrpc/connect";
import { CleanupService } from "@vrooli/proto-types/vrooli-bridge/v1/cleanup/cleanup_pb";

import { transport } from "./client";

export const cleanupClient = createClient(CleanupService, transport);

const encoder = new TextEncoder();
const PRIME = (1n << 255n) - 19n;
const A24 = 121665n;
const MASK = (1n << 255n) - 1n;

function littleToBigInt(value: Uint8Array): bigint {
  let result = 0n;
  for (let index = value.length - 1; index >= 0; index -= 1) {
    result = (result << 8n) | BigInt(value[index] ?? 0);
  }
  return result;
}

function bigIntToLittle(value: bigint): Uint8Array {
  const output = new Uint8Array(32);
  let current = value;
  for (let index = 0; index < output.length; index += 1) {
    output[index] = Number(current & 0xffn);
    current >>= 8n;
  }
  return output;
}

function mod(value: bigint): bigint {
  const result = value % PRIME;
  return result < 0n ? result + PRIME : result;
}

function pow(value: bigint, exponent: bigint): bigint {
  let base = mod(value);
  let result = 1n;
  let remaining = exponent;
  while (remaining > 0n) {
    if ((remaining & 1n) === 1n) result = mod(result * base);
    base = mod(base * base);
    remaining >>= 1n;
  }
  return result;
}

// RFC 7748's Montgomery ladder. This keeps the UI's sealed-envelope format
// identical to the shared Go sealing package without adding a third-party
// cryptography dependency to the Bridge bundle.
function x25519(scalarInput: Uint8Array, uInput: Uint8Array): Uint8Array {
  const scalar = new Uint8Array(scalarInput);
  scalar[0] = (scalar[0] ?? 0) & 248;
  scalar[31] = ((scalar[31] ?? 0) & 127) | 64;
  const u = littleToBigInt(uInput) & MASK;
  let x2 = 1n;
  let z2 = 0n;
  let x3 = u;
  let z3 = 1n;
  let swap = 0n;
  for (let bit = 254; bit >= 0; bit -= 1) {
    const current = BigInt((scalar[Math.floor(bit / 8)] ?? 0) >> (bit & 7) & 1);
    swap ^= current;
    if (swap === 1n) {
      [x2, x3] = [x3, x2];
      [z2, z3] = [z3, z2];
    }
    swap = current;
    const a = mod(x2 + z2);
    const aa = mod(a * a);
    const b = mod(x2 - z2);
    const bb = mod(b * b);
    const e = mod(aa - bb);
    const c = mod(x3 + z3);
    const d = mod(x3 - z3);
    const da = mod(d * a);
    const cb = mod(c * b);
    x3 = mod((da + cb) * (da + cb));
    z3 = mod(u * (da - cb) * (da - cb));
    x2 = mod(aa * bb);
    z2 = mod(e * (aa + A24 * e));
  }
  if (swap === 1n) {
    [x2, x3] = [x3, x2];
    [z2, z3] = [z3, z2];
  }
  return bigIntToLittle(mod(x2 * pow(z2, PRIME - 2n)));
}

async function hmac(key: Uint8Array, value: Uint8Array): Promise<Uint8Array> {
  const cryptoKey = await globalThis.crypto.subtle.importKey("raw", asArrayBuffer(key), { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
  return new Uint8Array(await globalThis.crypto.subtle.sign("HMAC", cryptoKey, asArrayBuffer(value)));
}

async function deriveKey(shared: Uint8Array, ephemeralPublic: Uint8Array, recipientPublic: Uint8Array, aad: Uint8Array): Promise<Uint8Array> {
  const prk = await hmac(encoder.encode("vrooli-cleanup-sealing-v1"), shared);
  const info = new Uint8Array(8 + ephemeralPublic.length + recipientPublic.length + aad.length + 1);
  info.set(encoder.encode("envelope"), 0);
  info.set(ephemeralPublic, 8);
  info.set(recipientPublic, 8 + ephemeralPublic.length);
  info.set(aad, 8 + ephemeralPublic.length + recipientPublic.length);
  info[info.length - 1] = 1;
  return hmac(prk, info);
}

function concat(...parts: Uint8Array[]): Uint8Array {
  const output = new Uint8Array(parts.reduce((total, part) => total + part.length, 0));
  let offset = 0;
  for (const part of parts) {
    output.set(part, offset);
    offset += part.length;
  }
  return output;
}

function asArrayBuffer(value: Uint8Array): ArrayBuffer {
  const output = new ArrayBuffer(value.byteLength);
  new Uint8Array(output).set(value);
  return output;
}

export async function sealCleanupPassphrase(
  recipientPublic: Uint8Array,
  passphrase: string,
  aadParts: string[],
): Promise<Uint8Array> {
  if (recipientPublic.length !== 32 || passphrase.trim() === "") {
    throw new Error("a node sealing key and passphrase are required");
  }
  if (!globalThis.crypto?.subtle || !globalThis.crypto.getRandomValues) {
    throw new Error("this browser cannot seal cleanup authorization locally");
  }
  const ephemeralPrivate = new Uint8Array(32);
  globalThis.crypto.getRandomValues(ephemeralPrivate);
  const basePoint = new Uint8Array(32);
  basePoint[0] = 9;
  const ephemeralPublic = x25519(ephemeralPrivate, basePoint);
  const shared = x25519(ephemeralPrivate, recipientPublic);
  const aad = encoder.encode(aadParts.join("\0"));
  const key = await deriveKey(shared, ephemeralPublic, recipientPublic, aad);
  const nonce = new Uint8Array(12);
  globalThis.crypto.getRandomValues(nonce);
  const cryptoKey = await globalThis.crypto.subtle.importKey("raw", asArrayBuffer(key), { name: "AES-GCM" }, false, ["encrypt"]);
  const ciphertext = new Uint8Array(await globalThis.crypto.subtle.encrypt({ name: "AES-GCM", iv: asArrayBuffer(nonce), additionalData: asArrayBuffer(aad) }, cryptoKey, asArrayBuffer(encoder.encode(passphrase))));
  ephemeralPrivate.fill(0);
  shared.fill(0);
  key.fill(0);
  return concat(encoder.encode("VCS1"), ephemeralPublic, nonce, ciphertext);
}
