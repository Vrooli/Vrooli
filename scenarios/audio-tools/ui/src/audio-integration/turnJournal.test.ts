import { describe, expect, it } from "vitest";

import { digestAudio, forgetUnfinishedSession, loadUnfinishedSession, MemoryTurnJournalStore, rememberUnfinishedSession, TurnJournal } from "@vrooli/audio-capture-browser";

describe("TurnJournal", () => {
  it("uses SHA-256 chunk identities even when Web Crypto is unavailable", async () => {
    const digest = new Uint8Array(await digestAudio(new TextEncoder().encode("abc").buffer));
    expect(Array.from(digest, (byte) => byte.toString(16).padStart(2, "0")).join("")).toBe(
      "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
    );
  });

  it("retains only opaque identity needed to resume a journal after reload", () => {
    forgetUnfinishedSession();
    rememberUnfinishedSession({ sessionId: "session-1", resumeToken: "token-1" });
    expect(loadUnfinishedSession()).toEqual({ sessionId: "session-1", resumeToken: "token-1" });
    forgetUnfinishedSession();
    expect(loadUnfinishedSession()).toBeNull();
  });

  it("[REQ:ATD-P0-006] retains only unprocessed replay coverage", async () => {
    const journal = new TurnJournal(new MemoryTurnJournalStore(), "session", 0n, 8, "memory");
    await journal.append({ sequence: 0n, startSample: 0n, endSample: 2n, audio: new Uint8Array([1, 2]).buffer, sha256: new ArrayBuffer(32) });
    await journal.append({ sequence: 1n, startSample: 2n, endSample: 4n, audio: new Uint8Array([3, 4]).buffer, sha256: new ArrayBuffer(32) });
    await journal.acknowledgeProcessed(0n);
    expect(journal.replayAfter(0n).map((chunk) => chunk.sequence)).toEqual([1n]);
    expect(journal.read().retainedBytes).toBe(2);
  });

  it("preserves the next stream cursor after acknowledged audio is compacted and restored", async () => {
    const store = new MemoryTurnJournalStore();
    const journal = new TurnJournal(store, "session", 0n, 8, "memory");
    await journal.append({ sequence: 0n, startSample: 0n, endSample: 2n, audio: new Uint8Array([1, 2]).buffer, sha256: new ArrayBuffer(32) });
    await journal.acknowledgeProcessed(0n);

    const reloaded = new TurnJournal(store, "session", 0n, 8, "memory");
    expect(await reloaded.restore()).toMatchObject({ nextSequence: 1n, nextSample: 2n, chunks: [] });
    await expect(reloaded.append({ sequence: 1n, startSample: 2n, endSample: 4n, audio: new Uint8Array([3, 4]).buffer, sha256: new ArrayBuffer(32) })).resolves.toBeUndefined();
  });

  it("signals quota exhaustion before dropping captured audio", async () => {
    const journal = new TurnJournal(new MemoryTurnJournalStore(), "session", 0n, 2, "memory");
    await journal.append({ sequence: 0n, startSample: 0n, endSample: 2n, audio: new Uint8Array([1, 2]).buffer, sha256: new ArrayBuffer(32) });
    await expect(journal.append({ sequence: 1n, startSample: 2n, endSample: 4n, audio: new Uint8Array([3]).buffer, sha256: new ArrayBuffer(32) })).rejects.toThrow("quota exhausted");
    expect(journal.read().chunks).toHaveLength(1);
  });
});
