import { webcrypto } from "node:crypto";
import { Buffer } from "node:buffer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { sealCleanupPassphrase } from "./cleanup";

describe("api/cleanup.sealCleanupPassphrase", () => {
  beforeEach(() => {
    // jsdom exposes a SubtleCrypto implementation from a different realm than
    // the ArrayBuffer values created by the application. Use Node's Web Crypto
    // implementation here so the test exercises the same standard API without
    // weakening the production conversion boundaries.
    const nodeImportKey = webcrypto.subtle.importKey.bind(webcrypto.subtle) as unknown as (
      format: string,
      keyData: BufferSource,
      algorithm: AlgorithmIdentifier,
      extractable: boolean,
      usages: KeyUsage[],
    ) => Promise<CryptoKey>;
    const nodeSign = webcrypto.subtle.sign.bind(webcrypto.subtle) as unknown as (
      algorithm: AlgorithmIdentifier,
      key: CryptoKey,
      data: BufferSource,
    ) => Promise<ArrayBuffer>;
    const nodeEncrypt = webcrypto.subtle.encrypt.bind(webcrypto.subtle) as unknown as (
      algorithm: AesGcmParams,
      key: CryptoKey,
      data: BufferSource,
    ) => Promise<ArrayBuffer>;
    vi.stubGlobal("crypto", {
      getRandomValues<T extends ArrayBufferView>(value: T): T {
        const random = webcrypto.getRandomValues(new Uint8Array(value.byteLength));
        new Uint8Array(value.buffer, value.byteOffset, value.byteLength).set(random);
        return value;
      },
      subtle: {
        importKey: (format: string, keyData: ArrayBuffer, algorithm: AlgorithmIdentifier, extractable: boolean, usages: KeyUsage[]) =>
          nodeImportKey(format, Buffer.from(new Uint8Array(keyData)), algorithm, extractable, usages),
        sign: (algorithm: AlgorithmIdentifier, key: CryptoKey, data: ArrayBuffer) =>
          nodeSign(algorithm, key, Buffer.from(new Uint8Array(data))),
        encrypt: (algorithm: AesGcmParams, key: CryptoKey, data: ArrayBuffer) =>
          nodeEncrypt(
            {
              ...algorithm,
              iv: Buffer.from(new Uint8Array(algorithm.iv as ArrayBuffer)),
              additionalData: algorithm.additionalData
                ? Buffer.from(new Uint8Array(algorithm.additionalData as ArrayBuffer))
                : undefined,
            },
            key,
            Buffer.from(new Uint8Array(data)),
          ),
      },
    } as Crypto);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("rejects an invalid recipient key or blank passphrase", async () => {
    await expect(sealCleanupPassphrase(new Uint8Array(31), "secret", ["op-1"])).rejects.toThrow(
      "a node sealing key and passphrase are required",
    );
    await expect(sealCleanupPassphrase(new Uint8Array(32), "   ", ["op-1"])).rejects.toThrow(
      "a node sealing key and passphrase are required",
    );
  });

  it("fails closed when browser Web Crypto is unavailable", async () => {
    vi.stubGlobal("crypto", {});

    await expect(sealCleanupPassphrase(new Uint8Array(32), "secret", ["op-1"])).rejects.toThrow(
      "this browser cannot seal cleanup authorization locally",
    );
  });

  it("seals the operator passphrase into a versioned node-bound envelope", async () => {
    const recipientPublic = new Uint8Array(32);
    recipientPublic[0] = 9;

    const envelope = await sealCleanupPassphrase(recipientPublic, "operator-secret", ["op-1", "target"]);
    const prefix = new TextDecoder().decode(envelope.slice(0, 4));

    expect(envelope).toBeInstanceOf(Uint8Array);
    expect(prefix).toBe("VCS1");
    // VCS1 + X25519 ephemeral public key + AES-GCM nonce/tag/ciphertext.
    expect(envelope.length).toBeGreaterThan(4 + 32 + 12 + 16);
  });

  it("uses fresh randomness for each envelope", async () => {
    const recipientPublic = new Uint8Array(32);
    recipientPublic[0] = 9;

    const first = await sealCleanupPassphrase(recipientPublic, "operator-secret", ["op-1"]);
    const second = await sealCleanupPassphrase(recipientPublic, "operator-secret", ["op-1"]);

    expect(Array.from(first)).not.toEqual(Array.from(second));
  });
});
