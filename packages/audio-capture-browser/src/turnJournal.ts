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

export type JournalRecord =
  | { kind: "append"; chunk: JournalChunk }
  | { kind: "ack"; sequence: bigint };

export interface TurnJournalStore {
  load(key: string): Promise<JournalSnapshot | undefined>;
  save(key: string, value: JournalSnapshot): Promise<void>;
  remove(key: string): Promise<void>;
}

interface IncrementalTurnJournalStore extends TurnJournalStore {
  appendRecord(key: string, record: JournalRecord): Promise<void>;
  loadRecords(key: string): Promise<JournalRecord[]>;
  compactRecords(key: string, snapshot: JournalSnapshot): Promise<void>;
}

/**
 * Bounded replay journal. Its invariant is deliberately narrow: captured
 * chunks survive until the server's processed cursor covers them, or the user
 * explicitly discards the turn. A storage adapter may be IndexedDB; a memory
 * adapter is visibly reduced durability rather than an invisible fallback.
 */
export class TurnJournal {
  private snapshot: JournalSnapshot;
  private pendingRecords = 0;
  // Appends run on the capture write chain while processed acknowledgements
  // arrive on an independent network callback. Serialize their IndexedDB
  // transactions here; otherwise a compaction can delete records while an
  // append transaction is still active, producing browser-level AbortError
  // and falsely reducing durability during a healthy stream.
  private persistenceQueue: Promise<void> = Promise.resolve();
  // Keep the un-compacted record tail below the 16 MiB journal quota while
  // avoiding a cursor transaction for every small batch during long-form
  // capture. At 3,200 bytes per 100 ms PCM frame this is about 3.3 MiB.
  private static readonly COMPACTION_RECORDS = 1024;

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
      this.snapshot = {
        ...stored,
        nextSequence: stored.nextSequence ?? (last ? last.sequence + 1n : 0n),
        nextSample: stored.nextSample ?? (last ? last.endSample : 0n),
      };
    }
    const incremental = this.store as Partial<IncrementalTurnJournalStore>;
    if (incremental.loadRecords) {
      const records = await incremental.loadRecords(key);
      if (records.length > 0) {
        for (const record of records) this.applyRecord(record);
        return this.read();
      }
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
    const storedChunk = { ...chunk, audio: chunk.audio.slice(0), sha256: chunk.sha256.slice(0) };
    this.snapshot.chunks.push(storedChunk);
    this.snapshot.retainedBytes += chunk.audio.byteLength;
    this.snapshot.nextSequence = chunk.sequence + 1n;
    this.snapshot.nextSample = chunk.endSample;
    await this.persist({ kind: "append", chunk: storedChunk });
  }

  async acknowledgeProcessed(sequence: bigint): Promise<void> {
    this.compactInMemory(sequence);
    await this.persist({ kind: "ack", sequence });
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
    this.pendingRecords = 0;
  }

  private compactInMemory(sequence: bigint): void {
    const retained = this.snapshot.chunks.filter((chunk) => chunk.sequence > sequence);
    this.snapshot.retainedBytes = retained.reduce((bytes, chunk) => bytes + chunk.audio.byteLength, 0);
    this.snapshot.chunks = retained;
  }

  private applyRecord(record: JournalRecord): void {
    if (record.kind === "append") {
      const chunk = { ...record.chunk, audio: record.chunk.audio.slice(0), sha256: record.chunk.sha256.slice(0) };
      this.snapshot.chunks.push(chunk);
      this.snapshot.retainedBytes += chunk.audio.byteLength;
      this.snapshot.nextSequence = chunk.sequence + 1n;
      this.snapshot.nextSample = chunk.endSample;
    } else {
      this.compactInMemory(record.sequence);
    }
  }

  private async persist(record: JournalRecord): Promise<void> {
    const operation = this.persistenceQueue.then(async () => {
      const key = TurnJournal.key(this.snapshot.sessionId, this.snapshot.generation);
      const incremental = this.store as Partial<IncrementalTurnJournalStore>;
      if (incremental.appendRecord && incremental.compactRecords) {
        await incremental.appendRecord(key, record);
        this.pendingRecords += 1;
        if (this.pendingRecords >= TurnJournal.COMPACTION_RECORDS) {
          await incremental.compactRecords(key, this.read());
          this.pendingRecords = 0;
        }
        return;
      }
      await this.store.save(key, this.read());
    });
    // A failed operation must not poison every later append/ack. The caller
    // still observes this operation's failure, while subsequent persistence
    // attempts get a usable chain for diagnostics and recovery.
    this.persistenceQueue = operation.catch(() => undefined);
    await operation;
  }
}

export class MemoryTurnJournalStore implements TurnJournalStore {
  private readonly data = new Map<string, JournalSnapshot>();
  private readonly records = new Map<string, JournalRecord[]>();

  async load(key: string): Promise<JournalSnapshot | undefined> { return this.data.get(key); }
  async save(key: string, value: JournalSnapshot): Promise<void> { this.data.set(key, value); }
  async remove(key: string): Promise<void> { this.data.delete(key); this.records.delete(key); }
  async appendRecord(key: string, record: JournalRecord): Promise<void> {
    const records = this.records.get(key) ?? [];
    records.push(record);
    this.records.set(key, records);
  }
  async loadRecords(key: string): Promise<JournalRecord[]> { return this.records.get(key) ?? []; }
  async compactRecords(key: string, snapshot: JournalSnapshot): Promise<void> {
    this.data.set(key, snapshot);
    this.records.delete(key);
  }
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
    const database = await this.open();
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(["turns", "turnRecords"], "readwrite");
      transaction.objectStore("turns").delete(key);
      const records = transaction.objectStore("turnRecords");
      const request = records.openCursor();
      request.onsuccess = () => {
        const cursor = request.result;
        if (!cursor) return;
        if ((cursor.value as { key?: string }).key === key) cursor.delete();
        cursor.continue();
      };
      request.onerror = () => reject(request.error ?? new Error("remove IndexedDB journal records"));
      transaction.oncomplete = () => resolve();
      transaction.onerror = () => reject(transaction.error ?? new Error("remove IndexedDB journal transaction"));
    });
  }

  private open(): Promise<IDBDatabase> {
    if (this.database) return this.database;
    this.database = new Promise((resolve, reject) => {
      if (typeof indexedDB === "undefined") {
        reject(new Error("IndexedDB is unavailable; use explicit reduced durability"));
        return;
      }
      const request = indexedDB.open(this.databaseName, 2);
      request.onupgradeneeded = () => {
        if (!request.result.objectStoreNames.contains("turns")) request.result.createObjectStore("turns");
        if (!request.result.objectStoreNames.contains("turnRecords")) request.result.createObjectStore("turnRecords", { autoIncrement: true });
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error ?? new Error("open IndexedDB journal"));
    });
    return this.database;
  }

  private async request<T>(mode: IDBTransactionMode, operation: (store: IDBObjectStore) => IDBRequest<T>, storeName = "turns"): Promise<T> {
    const database = await this.open();
    return new Promise<T>((resolve, reject) => {
      const transaction = database.transaction(storeName, mode);
      const request = operation(transaction.objectStore(storeName));
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error ?? new Error("IndexedDB journal operation failed"));
      transaction.onerror = () => reject(transaction.error ?? new Error("IndexedDB journal transaction failed"));
    });
  }

  async appendRecord(key: string, record: JournalRecord): Promise<void> {
    await this.request("readwrite", (store) => store.add({ key, record }), "turnRecords");
  }

  async loadRecords(key: string): Promise<JournalRecord[]> {
    const database = await this.open();
    return new Promise((resolve, reject) => {
      const transaction = database.transaction("turnRecords", "readonly");
      const request = transaction.objectStore("turnRecords").getAll();
      request.onsuccess = () => resolve((request.result as Array<{ key: string; record: JournalRecord }>).filter((entry) => entry.key === key).map((entry) => entry.record));
      request.onerror = () => reject(request.error ?? new Error("load IndexedDB journal records"));
    });
  }

  async compactRecords(key: string, snapshot: JournalSnapshot): Promise<void> {
    const database = await this.open();
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(["turns", "turnRecords"], "readwrite");
      const records = transaction.objectStore("turnRecords");
      const request = records.openCursor();
      request.onsuccess = () => {
        const cursor = request.result;
        if (cursor) {
          if ((cursor.value as { key?: string }).key === key) cursor.delete();
          cursor.continue();
          return;
        }
        // The cursor and snapshot write share one transaction. No second
        // getAll/getAllKeys callback can race this put or reactivate an
        // inactive transaction.
        transaction.objectStore("turns").put(snapshot, key);
      };
      request.onerror = () => reject(request.error ?? new Error("compact IndexedDB journal records"));
      transaction.oncomplete = () => resolve();
      transaction.onerror = () => reject(transaction.error ?? new Error("compact IndexedDB journal transaction"));
    });
  }
}
