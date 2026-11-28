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

interface ExpiringTurnJournalStore extends TurnJournalStore {
  prepareRestore(key: string): Promise<boolean>;
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
  private expiredOnRestore = false;
  private restoredDurableState = false;
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

  /** True only when this restore deleted expired or unageable durable audio. */
  wasExpiredOnRestore(): boolean { return this.expiredOnRestore; }

  /** True when restore found a durable snapshot or replay record for this identity. */
  hasRestoredDurableState(): boolean { return this.restoredDurableState; }

  /** Read quota accounting without cloning the retained audio chunks. */
  get retainedBytes(): number { return this.snapshot.retainedBytes; }

  /** Read the next cursor without cloning the retained audio chunks. */
  get nextSequence(): bigint { return this.snapshot.nextSequence; }

  /** Start a fresh identity after expiry; never move retained audio to a new owner. */
  rebindAfterExpiry(sessionId: string, generation: bigint): void {
    if (!this.expiredOnRestore || this.snapshot.retainedBytes !== 0 || this.snapshot.chunks.length !== 0) {
      throw new Error("journal identity can only be rebound after an expired restore with no retained audio");
    }
    this.snapshot = { ...this.snapshot, sessionId, generation, chunks: [], retainedBytes: 0, nextSequence: 0n, nextSample: 0n };
    this.expiredOnRestore = false;
  }

  async restore(): Promise<JournalSnapshot> {
    const key = TurnJournal.key(this.snapshot.sessionId, this.snapshot.generation);
    const expiring = this.store as Partial<ExpiringTurnJournalStore>;
    if (expiring.prepareRestore && !(await expiring.prepareRestore(key))) {
      this.expiredOnRestore = true;
      return this.read();
    }
    const stored = await this.store.load(key);
    if (stored) {
      this.restoredDurableState = true;
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
        this.restoredDurableState = true;
        const beforeReplay = { ...this.snapshot, chunks: [...this.snapshot.chunks] };
        try {
          for (const record of records) this.applyRecord(record);
        } catch (error) {
          this.snapshot = beforeReplay;
          throw error;
        }
        return this.read();
      }
    }
    return this.read();
  }

  async append(chunk: JournalChunk): Promise<void> {
    const storedChunk = { ...chunk, audio: chunk.audio.slice(0), sha256: chunk.sha256.slice(0) };
    await this.serialize(async () => {
      if (storedChunk.endSample < storedChunk.startSample || storedChunk.audio.byteLength === 0) {
        throw new Error("journal chunk must contain a non-empty non-negative sample range");
      }
      if (storedChunk.sequence !== this.snapshot.nextSequence || storedChunk.startSample !== this.snapshot.nextSample) {
        throw new Error(`journal chunks must have contiguous sequence identities (expected sequence ${this.snapshot.nextSequence.toString()} at sample ${this.snapshot.nextSample.toString()}, got sequence ${storedChunk.sequence.toString()} at sample ${storedChunk.startSample.toString()})`);
      }
      if (this.snapshot.retainedBytes + storedChunk.audio.byteLength > this.maxBytes) {
        throw new Error("journal quota exhausted before capture; audio was not discarded");
      }
      const next = { ...this.snapshot, chunks: [...this.snapshot.chunks, storedChunk],
        retainedBytes: this.snapshot.retainedBytes + storedChunk.audio.byteLength,
        nextSequence: storedChunk.sequence + 1n, nextSample: storedChunk.endSample };
      await this.persist({ kind: "append", chunk: storedChunk }, next);
      this.snapshot = next;
    });
  }

  async acknowledgeProcessed(sequence: bigint): Promise<void> {
    await this.serialize(async () => {
      // A server cursor is authoritative only for audio the client has
      // actually assigned a sequence to. A malformed future acknowledgement
      // must never compact captured-but-unprocessed audio from the replay
      // journal. Validate at the durability owner even when transports also
      // validate the wire shape.
      if (sequence < 0n || sequence >= this.snapshot.nextSequence) {
        throw new Error(`processed cursor ${sequence.toString()} is outside captured sequence range`);
      }
      const chunks = this.snapshot.chunks.filter((chunk) => chunk.sequence > sequence);
      const next = { ...this.snapshot, chunks, retainedBytes: chunks.reduce((n, chunk) => n + chunk.audio.byteLength, 0) };
      await this.persist({ kind: "ack", sequence }, next);
      this.snapshot = next;
    });
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

  /** Seed a replacement journal after a storage adapter loses availability. */
  seed(snapshot: JournalSnapshot): void {
    this.snapshot = {
      ...snapshot,
      chunks: snapshot.chunks.map((chunk) => ({
        ...chunk,
        audio: chunk.audio.slice(0),
        sha256: chunk.sha256.slice(0),
      })),
    };
    this.pendingRecords = 0;
    this.persistenceQueue = Promise.resolve();
  }

  /**
   * Repair a stale persisted sample cursor at the next sequence boundary.
   * This is only valid after storage recovery has already reduced durability:
   * no retained chunk may be at or beyond the cursor, so the next frame is
   * the authoritative sample boundary for the active capture.
   */
  rebaseNextSample(nextSample: bigint): void {
    if (this.snapshot.chunks.some((chunk) => chunk.sequence >= this.snapshot.nextSequence)) {
      throw new Error("journal cursor cannot be rebased with a future retained chunk");
    }
    this.snapshot.nextSample = nextSample;
  }

  async discard(): Promise<void> {
    await this.serialize(async () => {
      await this.store.remove(TurnJournal.key(this.snapshot.sessionId, this.snapshot.generation));
      this.snapshot.chunks = [];
      this.snapshot.retainedBytes = 0;
      this.pendingRecords = 0;
    });
  }

  private compactInMemory(sequence: bigint): void {
    const retained = this.snapshot.chunks.filter((chunk) => chunk.sequence > sequence);
    this.snapshot.retainedBytes = retained.reduce((bytes, chunk) => bytes + chunk.audio.byteLength, 0);
    this.snapshot.chunks = retained;
  }

  private applyRecord(record: JournalRecord): void {
    if (record.kind === "append") {
      const chunk = { ...record.chunk, audio: record.chunk.audio.slice(0), sha256: record.chunk.sha256.slice(0) };
      if (chunk.sequence !== this.snapshot.nextSequence || chunk.startSample !== this.snapshot.nextSample) {
        throw new Error(`journal recovery encountered a non-contiguous append (expected sequence ${this.snapshot.nextSequence.toString()} at sample ${this.snapshot.nextSample.toString()}, got sequence ${chunk.sequence.toString()} at sample ${chunk.startSample.toString()})`);
      }
      if (chunk.endSample < chunk.startSample || chunk.audio.byteLength === 0) {
        throw new Error("journal recovery encountered an invalid audio chunk");
      }
      this.snapshot.chunks.push(chunk);
      this.snapshot.retainedBytes += chunk.audio.byteLength;
      this.snapshot.nextSequence = chunk.sequence + 1n;
      this.snapshot.nextSample = chunk.endSample;
    } else {
      if (record.sequence < 0n || record.sequence >= this.snapshot.nextSequence) {
        throw new Error(`journal recovery encountered a processed cursor outside captured range (${record.sequence.toString()})`);
      }
      this.compactInMemory(record.sequence);
    }
  }

  private async serialize(operation: () => Promise<void>): Promise<void> {
    const pending = this.persistenceQueue.then(operation);
    this.persistenceQueue = pending.catch(() => undefined);
    await pending;
  }

  private async persist(record: JournalRecord, next: JournalSnapshot): Promise<void> {
    const key = TurnJournal.key(next.sessionId, next.generation);
    const incremental = this.store as Partial<IncrementalTurnJournalStore>;
    if (incremental.appendRecord && incremental.compactRecords) {
      await incremental.appendRecord(key, record);
      this.pendingRecords += 1;
      if (this.pendingRecords >= TurnJournal.COMPACTION_RECORDS) {
        await incremental.compactRecords(key, next);
        this.pendingRecords = 0;
      }
      return;
    }
    await this.store.save(key, next);
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

  /** Abandoned audio is recoverable for one day, then deleted before it can be replayed. */
  static readonly RETENTION_MS = 24 * 60 * 60 * 1000;

  constructor(private readonly databaseName = "vrooli-audio-turn-journal") {}

  async load(key: string): Promise<JournalSnapshot | undefined> {
    return this.request<JournalSnapshot | undefined>("readonly", (store) => store.get(key));
  }

  async prepareRestore(key: string): Promise<boolean> {
    const database = await this.open();
    return new Promise<boolean>((resolve, reject) => {
      const transaction = database.transaction(["turns", "turnRecords", "turnMeta"], "readwrite");
      const turns = transaction.objectStore("turns");
      const records = transaction.objectStore("turnRecords");
      const meta = transaction.objectStore("turnMeta");
      const snapshotRequest = turns.get(key);
      const metaRequest = meta.get(key);
      const countRequest = records.index("byKey").count(key);
      let snapshot: (JournalSnapshot & { createdAt?: number; updatedAt?: number }) | undefined;
      let activity: { createdAt?: number; updatedAt?: number } | undefined;
      let recordCount: number | undefined;
      let inspected = 0;
      let retain = true;
      let decided = false;
      const inspect = () => {
        inspected += 1;
        if (inspected !== 3 || decided) return;
        decided = true;
        snapshot = snapshotRequest.result as (JournalSnapshot & { createdAt?: number; updatedAt?: number }) | undefined;
        activity = metaRequest.result as { createdAt?: number; updatedAt?: number } | undefined;
        recordCount = countRequest.result;
        if (!snapshot && recordCount === 0) {
          // A never-used journal is not an expired recovery attempt.
          if (activity) meta.delete(key);
          return;
        }
        // Records written before the activity store existed have no trustworthy
        // per-session age. Expire them conservatively instead of scanning PCM.
        const timestamp = activity?.createdAt ?? snapshot?.createdAt ?? (recordCount === 0 ? (activity?.updatedAt ?? snapshot?.updatedAt) : undefined);
        const expired = typeof timestamp !== "number" || Date.now() - timestamp >= IndexedDBTurnJournalStore.RETENTION_MS;
        if (!expired) return;
        retain = false;
        turns.delete(key);
        meta.delete(key);
        this.deleteRecordsForKey(records, key);
      };
      snapshotRequest.onsuccess = inspect;
      metaRequest.onsuccess = inspect;
      countRequest.onsuccess = inspect;
      for (const request of [snapshotRequest, metaRequest, countRequest]) {
        request.onerror = () => reject(request.error ?? new Error("inspect IndexedDB journal retention"));
      }
      transaction.oncomplete = () => resolve(retain);
      transaction.onabort = () => reject(transaction.error ?? new Error("inspect IndexedDB journal retention"));
      transaction.onerror = () => reject(transaction.error ?? new Error("inspect IndexedDB journal retention"));
    });
  }

  async save(key: string, value: JournalSnapshot): Promise<void> {
    const database = await this.open();
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(["turns", "turnMeta"], "readwrite");
      const updatedAt = Date.now();
      this.recordActivity(transaction, key, updatedAt, (createdAt) => {
        transaction.objectStore("turns").put({ ...value, createdAt, updatedAt }, key);
      });
      transaction.oncomplete = () => resolve();
      transaction.onabort = () => reject(transaction.error ?? new Error("save IndexedDB journal transaction aborted"));
      transaction.onerror = () => reject(transaction.error ?? new Error("save IndexedDB journal transaction failed"));
    });
  }

  async remove(key: string): Promise<void> {
    const database = await this.open();
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(["turns", "turnRecords", "turnMeta"], "readwrite");
      transaction.objectStore("turns").delete(key);
      transaction.objectStore("turnMeta").delete(key);
      const records = transaction.objectStore("turnRecords");
      this.deleteRecordsForKey(records, key);
      transaction.oncomplete = () => resolve();
      transaction.onabort = () => reject(transaction.error ?? new Error("IndexedDB journal transaction aborted"));
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
      const request = indexedDB.open(this.databaseName, 3);
      request.onupgradeneeded = () => {
        if (!request.result.objectStoreNames.contains("turns")) request.result.createObjectStore("turns");
        if (!request.result.objectStoreNames.contains("turnRecords")) request.result.createObjectStore("turnRecords", { autoIncrement: true });
        if (!request.result.objectStoreNames.contains("turnMeta")) request.result.createObjectStore("turnMeta");
        const records = request.transaction?.objectStore("turnRecords");
        if (records && !records.indexNames.contains("byKey")) records.createIndex("byKey", "key", { unique: false });
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
      // A successful request is provisional until its transaction commits.
      transaction.oncomplete = () => resolve(request.result);
      transaction.onabort = () => reject(transaction.error ?? new Error("IndexedDB journal transaction aborted"));
      request.onerror = () => reject(request.error ?? new Error("IndexedDB journal operation failed"));
      transaction.onerror = () => reject(transaction.error ?? new Error("IndexedDB journal transaction failed"));
    });
  }

  async appendRecord(key: string, record: JournalRecord): Promise<void> {
    const database = await this.open();
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(["turnRecords", "turnMeta"], "readwrite");
      const updatedAt = Date.now();
      transaction.objectStore("turnRecords").add({ key, record });
      this.recordActivity(transaction, key, updatedAt);
      transaction.oncomplete = () => resolve();
      transaction.onabort = () => reject(transaction.error ?? new Error("append IndexedDB journal record aborted"));
      transaction.onerror = () => reject(transaction.error ?? new Error("append IndexedDB journal record failed"));
    });
  }

  async loadRecords(key: string): Promise<JournalRecord[]> {
    const database = await this.open();
    return new Promise((resolve, reject) => {
      const transaction = database.transaction("turnRecords", "readonly");
      const request = transaction.objectStore("turnRecords").index("byKey").getAll(IDBKeyRange.only(key));
      transaction.oncomplete = () => resolve((request.result as Array<{ key: string; record: JournalRecord }>).map((entry) => entry.record));
      transaction.onabort = () => reject(transaction.error ?? new Error("load IndexedDB journal transaction aborted"));
      transaction.onerror = () => reject(transaction.error ?? new Error("load IndexedDB journal transaction failed"));
      request.onerror = () => reject(request.error ?? new Error("load IndexedDB journal records"));
    });
  }

  async compactRecords(key: string, snapshot: JournalSnapshot): Promise<void> {
    const database = await this.open();
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(["turns", "turnRecords", "turnMeta"], "readwrite");
      const records = transaction.objectStore("turnRecords");
      const request = records.index("byKey").openCursor(IDBKeyRange.only(key));
      request.onsuccess = () => {
        const cursor = request.result;
        if (cursor) {
          cursor.delete();
          cursor.continue();
          return;
        }
        // The cursor and snapshot write share one transaction. No second
        // getAll/getAllKeys callback can race this put or reactivate an
        // inactive transaction.
        const updatedAt = Date.now();
        this.recordActivity(transaction, key, updatedAt, (createdAt) => {
          transaction.objectStore("turns").put({ ...snapshot, createdAt, updatedAt }, key);
        });
      };
      request.onerror = () => reject(request.error ?? new Error("compact IndexedDB journal records"));
      transaction.oncomplete = () => resolve();
      transaction.onabort = () => reject(transaction.error ?? new Error("IndexedDB journal transaction aborted"));
      transaction.onerror = () => reject(transaction.error ?? new Error("compact IndexedDB journal transaction"));
    });
  }

  private deleteRecordsForKey(records: IDBObjectStore, key: string): void {
    const request = records.index("byKey").openCursor(IDBKeyRange.only(key));
    request.onsuccess = () => {
      const cursor = request.result;
      if (!cursor) return;
      cursor.delete();
      cursor.continue();
    };
  }

  private recordActivity(transaction: IDBTransaction, key: string, updatedAt: number, onCreated?: (createdAt: number) => void): void {
    const store = transaction.objectStore("turnMeta");
    const request = store.get(key);
    request.onsuccess = () => {
      const previous = request.result as { createdAt?: number; updatedAt?: number } | undefined;
      const createdAt = previous?.createdAt ?? previous?.updatedAt ?? updatedAt;
      store.put({ createdAt, updatedAt }, key);
      onCreated?.(createdAt);
    };
  }
}
