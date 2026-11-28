import { describe, expect, it } from "vitest";
import { MemoryTurnJournalStore, TurnJournal } from "./turnJournal";

const chunk = (sequence: bigint) => ({
  sequence,
  startSample: sequence * 2n,
  endSample: sequence * 2n + 2n,
  audio: new ArrayBuffer(4),
  sha256: new ArrayBuffer(32),
});

describe("TurnJournal acknowledgement bounds", () => {
  it("rejects a future processed cursor without deleting retained audio", async () => {
    const journal = new TurnJournal(new MemoryTurnJournalStore(), "bounded-ack", 0n, 32, "memory");
    await journal.append(chunk(0n));

    await expect(journal.acknowledgeProcessed(1_000n)).rejects.toThrow("outside captured sequence range");
    expect(journal.read()).toMatchObject({
      nextSequence: 1n,
      retainedBytes: 4,
      chunks: [{ sequence: 0n }],
    });
  });

  it("compacts the inclusive acknowledged sequence when it was captured", async () => {
    const journal = new TurnJournal(new MemoryTurnJournalStore(), "valid-ack", 0n, 32, "memory");
    await journal.append(chunk(0n));
    await journal.append(chunk(1n));

    expect(journal.retainedBytes).toBe(8);
    expect(journal.nextSequence).toBe(2n);

    await journal.acknowledgeProcessed(0n);
    expect(journal.read()).toMatchObject({
      nextSequence: 2n,
      retainedBytes: 4,
      chunks: [{ sequence: 1n }],
    });
    expect(journal.retainedBytes).toBe(4);
    expect(journal.nextSequence).toBe(2n);
  });

  it("rejects negative cursors", async () => {
    const journal = new TurnJournal(new MemoryTurnJournalStore(), "negative-ack", 0n, 32, "memory");
    await journal.append(chunk(0n));

    await expect(journal.acknowledgeProcessed(-1n)).rejects.toThrow("outside captured sequence range");
    expect(journal.read().chunks.map(({ sequence }) => sequence)).toEqual([0n]);
  });
});

describe("retention-aware restore", () => {
  it("does not treat a never-used journal as expired", async () => {
    class EmptyStore extends MemoryTurnJournalStore {
      async prepareRestore(_key: string): Promise<boolean> { return true; }
    }
    const journal = new TurnJournal(new EmptyStore(), "fresh-session", 0n, 32, "persistent");
    await journal.restore();
    expect(journal.wasExpiredOnRestore()).toBe(false);
  });

  it("recovers audio before expiry and refuses it after the store expires it", async () => {
    class RetentionStore extends MemoryTurnJournalStore {
      expired = false;
      async prepareRestore(key: string): Promise<boolean> {
        if (!this.expired) return true;
        await this.remove(key);
        return false;
      }
    }
    const store = new RetentionStore();
    const original = new TurnJournal(store, "retention", 0n, 32, "persistent");
    await original.append(chunk(0n));

    const recoverable = new TurnJournal(store, "retention", 0n, 32, "persistent");
    expect((await recoverable.restore()).retainedBytes).toBe(4);

    store.expired = true;
    const expired = new TurnJournal(store, "retention", 0n, 32, "persistent");
    expect((await expired.restore()).retainedBytes).toBe(0);
    expect(await store.load(TurnJournal.key("retention", 0n))).toBeUndefined();
  });
});

describe("incremental recovery integrity", () => {
  it.each([
    ["duplicate sequence", { ...chunk(0n), startSample: 0n, endSample: 2n }],
    ["sample gap", { ...chunk(1n), startSample: 3n, endSample: 5n }],
  ])("rejects a %s instead of replaying ambiguous audio", async (_label, second) => {
    const store = new MemoryTurnJournalStore();
    const key = TurnJournal.key("corrupt-replay", 0n);
    await store.appendRecord(key, { kind: "append", chunk: chunk(0n) });
    await store.appendRecord(key, { kind: "append", chunk: second });
    const journal = new TurnJournal(store, "corrupt-replay", 0n, 32, "persistent");

    await expect(journal.restore()).rejects.toThrow("non-contiguous append");
    expect(journal.read()).toMatchObject({ nextSequence: 0n, nextSample: 0n, retainedBytes: 0, chunks: [] });
  });

  it("rejects a processed record that claims uncaptured audio", async () => {
    const store = new MemoryTurnJournalStore();
    const key = TurnJournal.key("future-replay-ack", 0n);
    await store.appendRecord(key, { kind: "ack", sequence: 0n });
    const journal = new TurnJournal(store, "future-replay-ack", 0n, 32, "persistent");

    await expect(journal.restore()).rejects.toThrow("processed cursor outside captured range");
    expect(journal.read()).toMatchObject({ nextSequence: 0n, retainedBytes: 0, chunks: [] });
  });
});
