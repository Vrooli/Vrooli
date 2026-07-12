export type DurabilityLevel = "persistent" | "memory" | "reduced";

export interface JournalChunk {
  sequence: bigint;
  startSample: bigint;
  endSample: bigint;
  audio: ArrayBuffer;
  sha256: ArrayBuffer;
}

export interface JournalSnapshot {
  sessionId: string;
  generation: bigint;
  durability: DurabilityLevel;
  chunks: JournalChunk[];
  retainedBytes: number;
  /** The identity assigned to the next capture frame, even after compaction. */
  nextSequence: bigint;
  /** The canonical sample position assigned to the next capture frame. */
  nextSample: bigint;
}

export interface TurnJournalStore {
  load(key: string): Promise<JournalSnapshot | undefined>;
  save(key: string, value: JournalSnapshot): Promise<void>;
  remove(key: string): Promise<void>;
}

/**
 * Bounded replay journal. Its invariant is deliberately narrow: captured
 * chunks survive until the server's processed cursor covers them, or the user
 * explicitly discards the turn. A storage adapter may be IndexedDB; a memory
 * adapter is visibly reduced durability rather than an invisible fallback.
 */
export class TurnJournal {
  private snapshot: JournalSnapshot;

  constructor(
    private readonly store: TurnJournalStore,
    sessionId: string,
    generation: bigint,
    private readonly maxBytes: number,
    durability: DurabilityLevel,
  ) {
    this.snapshot = { sessionId, generation, durability, chunks: [], retainedBytes: 0, nextSequence: 0n, nextSample: 0n };
  }

  static key(sessionId: string, generation: bigint): string {
    return `${sessionId}:${generation.toString()}`;
  }

  async restore(): Promise<JournalSnapshot> {
    const key = TurnJournal.key(this.snapshot.sessionId, this.snapshot.generation);
    const stored = await this.store.load(key);
    if (stored) {
      const last = stored.chunks.at(-1);
      // Snapshots written before the cursor fields existed retain enough
      // information only while a replayable tail remains. New snapshots never
      // rely on that lossy inference: compaction must not reset stream identity.
      this.snapshot = {
        ...stored,
        nextSequence: stored.nextSequence ?? (last ? last.sequence + 1n : 0n),
        nextSample: stored.nextSample ?? (last ? last.endSample : 0n),
      };
    }
    return this.read();
  }

  async append(chunk: JournalChunk): Promise<void> {
    if (chunk.endSample < chunk.startSample || chunk.audio.byteLength === 0) {
      throw new Error("journal chunk must contain a non-empty non-negative sample range");
    }
    if (chunk.sequence !== this.snapshot.nextSequence || chunk.startSample !== this.snapshot.nextSample) {
      throw new Error("journal chunks must have contiguous sequence identities");
    }
    if (this.snapshot.retainedBytes + chunk.audio.byteLength > this.maxBytes) {
      throw new Error("journal quota exhausted before capture; audio was not discarded");
    }
    this.snapshot.chunks.push({ ...chunk, audio: chunk.audio.slice(0), sha256: chunk.sha256.slice(0) });
    this.snapshot.retainedBytes += chunk.audio.byteLength;
    this.snapshot.nextSequence = chunk.sequence + 1n;
    this.snapshot.nextSample = chunk.endSample;
    await this.persist();
  }

  async acknowledgeProcessed(sequence: bigint): Promise<void> {
    const retained = this.snapshot.chunks.filter((chunk) => chunk.sequence > sequence);
    this.snapshot.retainedBytes = retained.reduce((bytes, chunk) => bytes + chunk.audio.byteLength, 0);
    this.snapshot.chunks = retained;
    await this.persist();
  }

  replayAfter(sequence: bigint): JournalChunk[] {
    return this.snapshot.chunks
      .filter((chunk) => chunk.sequence > sequence)
      .map((chunk) => ({ ...chunk, audio: chunk.audio.slice(0), sha256: chunk.sha256.slice(0) }));
  }

  read(): JournalSnapshot {
    return {
      ...this.snapshot,
      chunks: this.snapshot.chunks.map((chunk) => ({ ...chunk, audio: chunk.audio.slice(0), sha256: chunk.sha256.slice(0) })),
    };
  }

  async discard(): Promise<void> {
    await this.store.remove(TurnJournal.key(this.snapshot.sessionId, this.snapshot.generation));
    this.snapshot.chunks = [];
    this.snapshot.retainedBytes = 0;
  }

  private async persist(): Promise<void> {
    await this.store.save(TurnJournal.key(this.snapshot.sessionId, this.snapshot.generation), this.read());
  }
}

export class MemoryTurnJournalStore implements TurnJournalStore {
  private readonly data = new Map<string, JournalSnapshot>();

  async load(key: string): Promise<JournalSnapshot | undefined> { return this.data.get(key); }
  async save(key: string, value: JournalSnapshot): Promise<void> { this.data.set(key, value); }
  async remove(key: string): Promise<void> { this.data.delete(key); }
}

/** IndexedDB-backed journal storage for supported browser contexts. */
export class IndexedDBTurnJournalStore implements TurnJournalStore {
  private database: Promise<IDBDatabase> | null = null;

  constructor(private readonly databaseName = "vrooli-audio-turn-journal") {}

  async load(key: string): Promise<JournalSnapshot | undefined> {
    return this.request<JournalSnapshot | undefined>("readonly", (store) => store.get(key));
  }

  async save(key: string, value: JournalSnapshot): Promise<void> {
    await this.request("readwrite", (store) => store.put(value, key));
  }

  async remove(key: string): Promise<void> {
    await this.request("readwrite", (store) => store.delete(key));
  }

  private open(): Promise<IDBDatabase> {
    if (this.database) return this.database;
    this.database = new Promise((resolve, reject) => {
      if (typeof indexedDB === "undefined") {
        reject(new Error("IndexedDB is unavailable; use explicit reduced durability"));
        return;
      }
      const request = indexedDB.open(this.databaseName, 1);
      request.onupgradeneeded = () => {
        if (!request.result.objectStoreNames.contains("turns")) request.result.createObjectStore("turns");
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error ?? new Error("open IndexedDB journal"));
    });
    return this.database;
  }

  private async request<T>(mode: IDBTransactionMode, operation: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> {
    const database = await this.open();
    return new Promise<T>((resolve, reject) => {
      const transaction = database.transaction("turns", mode);
      const request = operation(transaction.objectStore("turns"));
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error ?? new Error("IndexedDB journal operation failed"));
      transaction.onerror = () => reject(transaction.error ?? new Error("IndexedDB journal transaction failed"));
    });
  }
}
