import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { IndexedDBTurnJournalStore, TurnJournal, type JournalSnapshot, type TurnJournalStore } from "./turnJournal";
import { MAX_RETAINED_TURN_AUDIO_BYTES, PcmVoiceStreamProvider } from "./pcmVoiceStreamProvider";

function deferred() {
  let resolve!: () => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<void>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
const chunk = (sequence = 0n) => ({ sequence, startSample: sequence * 2n, endSample: sequence * 2n + 2n, audio: new ArrayBuffer(4), sha256: new ArrayBuffer(32) });
class DeferredStore implements TurnJournalStore {
  pending = deferred();
  save = vi.fn(async (_key: string, _value: JournalSnapshot) => this.pending.promise);
  load = async () => undefined;
  remove = async () => {};
}
class Socket {
  static OPEN = 1;
  static CONNECTING = 0;
  static CLOSED = 3;
  static instances: Socket[] = [];
  readyState = Socket.CONNECTING;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  send = vi.fn();
  close = vi.fn(() => { this.readyState = Socket.CLOSED; });
  constructor(_url: string) { Socket.instances.push(this); }
  open() { this.readyState = Socket.OPEN; this.onopen?.(); }
}
async function settle() { for (let i = 0; i < 128; i++) await Promise.resolve(); }

describe("selected journal acceptance", () => {
  let provider: PcmVoiceStreamProvider | undefined;
  let frame!: (samples: Float32Array, rate: number) => void;
  beforeEach(() => { vi.useFakeTimers(); Socket.instances = []; vi.stubGlobal("WebSocket", Socket); });
  afterEach(() => { provider?.dispose(); provider = undefined; vi.unstubAllGlobals(); vi.useRealTimers(); });
  async function start(store: DeferredStore, durability: "persistent" | "memory" = "persistent") {
    const journal = new TurnJournal(store, "acceptance", 0n, MAX_RETAINED_TURN_AUDIO_BYTES, durability);
    provider = new PcmVoiceStreamProvider({
      getUserMedia: async () => ({ getTracks: () => [{ readyState: "live", stop: vi.fn() }] }) as unknown as MediaStream,
      transport: { buildStreamUrl: () => "ws://fixture/voice", transcribeRetained: async () => "" },
      captureFactory: (_stream, onFrame) => { frame = onFrame; return { stop: vi.fn() }; },
      journalFactory: () => journal, wireBatchSamples: 2,
    });
    await provider.start();
    return { journal, socket: Socket.instances.at(-1)! };
  }
  it("keeps provisional appends out of committed snapshots and preserves cursors on rejection", async () => {
    const store = new DeferredStore();
    const journal = new TurnJournal(store, "test", 0n, 16, "persistent");
    const pending = journal.append(chunk());
    const rejected = expect(pending).rejects.toThrow("aborted");
    await settle();
    expect(journal.read()).toMatchObject({ nextSequence: 0n, nextSample: 0n, retainedBytes: 0, chunks: [] });
    store.pending.reject(new Error("transaction aborted"));
    await rejected;
    expect(journal.read().nextSequence).toBe(0n);
    store.pending = deferred();
    const retry = journal.append(chunk());
    await settle(); store.pending.resolve(); await retry;
    expect(journal.read()).toMatchObject({ nextSequence: 1n, nextSample: 2n, retainedBytes: 4 });
  });
  it("fresh socket open and stop cannot overtake an unresolved persistent append", async () => {
    const store = new DeferredStore(); const { socket } = await start(store);
    frame(new Float32Array([0.5, 0.5]), 16_000); await settle();
    socket.open(); provider!.stop(); await settle();
    expect(socket.send).not.toHaveBeenCalled();
    store.pending.resolve(); await settle();
    expect(socket.send.mock.calls).toHaveLength(2);
    expect(socket.send.mock.calls[0]![0]).toBeInstanceOf(ArrayBuffer);
    expect(socket.send.mock.calls[1]![0]).toBe(JSON.stringify({ type: "done" }));
  });
  it("explicit memory mode also waits for acceptance and sends exactly once", async () => {
    const store = new DeferredStore(); const { socket } = await start(store, "memory"); socket.open();
    frame(new Float32Array([0.5, 0.5]), 16_000); await settle(); expect(socket.send).not.toHaveBeenCalled();
    store.pending.resolve(); await settle(); expect(socket.send).toHaveBeenCalledOnce();
  });
  it("persistent failure receives bounded memory acceptance before one visibly reduced send", async () => {
    const store = new DeferredStore(); const { socket, journal } = await start(store); socket.open();
    const status = vi.fn(); provider!.onStatus = status;
    frame(new Float32Array([0.5, 0.5]), 16_000); await settle(); expect(socket.send).not.toHaveBeenCalled();
    store.pending.reject(new Error("transaction aborted")); await settle();
    expect(journal.read().chunks).toEqual([]);
    expect(status).toHaveBeenCalledWith(expect.objectContaining({ code: "durability_reduced" }));
    expect(provider!.getDiagnostic().retainedBytes).toBe(4);
    expect(socket.send).toHaveBeenCalledOnce();
  });
  it("failed reduced writes never send frames or a successful done", async () => {
    const store = new DeferredStore(); const { socket } = await start(store, "memory"); socket.open();
    frame(new Float32Array([0.5, 0.5]), 16_000); await settle();
    store.pending.reject(new Error("quota exhausted")); await settle(); provider!.stop(); await settle();
    expect(socket.send).not.toHaveBeenCalled(); expect(provider!.getDiagnostic().state).toBe("failed");
  });
  it("an early terminal response cannot complete or discard an unaccepted stopped frame", async () => {
    const store = new DeferredStore(); const { socket, journal } = await start(store); socket.open();
    const result = vi.fn(); provider!.onResult = result;
    frame(new Float32Array([0.5, 0.5]), 16_000); await settle(); provider!.stop();
    socket.onmessage!({ data: JSON.stringify({ type: "final", text: "committed words" }) }); await settle();
    expect(result).not.toHaveBeenCalled(); expect(socket.send).not.toHaveBeenCalled();
    expect(journal.read().nextSequence).toBe(0n);
    store.pending.resolve(); await settle();
    expect(result).toHaveBeenCalledWith("committed words");
    expect(socket.send.mock.calls).toHaveLength(2);
  });
  it("dispose fences late accepted frames and done", async () => {
    const store = new DeferredStore(); const { socket } = await start(store); socket.open();
    frame(new Float32Array([0.5, 0.5]), 16_000); await settle(); provider!.dispose();
    store.pending.resolve(); await settle(); expect(socket.send).not.toHaveBeenCalled();
  });
  it.each(["connecting", "open"])("reuses a %s permission-warmed socket while honoring acceptance and done", async (state) => {
    const store = new DeferredStore(); await start(store);
    provider!.dispose(); Socket.instances = [];
    provider!.preConnect("en");
    const socket = Socket.instances[0]!;
    if (state === "open") socket.open();
    await provider!.start();
    expect(Socket.instances).toHaveLength(1);
    frame(new Float32Array([0.5, 0.5]), 16_000); await settle();
    provider!.stop(); if (state === "connecting") socket.open(); await settle();
    expect(socket.send).not.toHaveBeenCalled();
    store.pending.resolve(); await settle();
    expect(socket.send.mock.calls).toHaveLength(2);
    expect(socket.send.mock.calls[1]![0]).toBe(JSON.stringify({ type: "done" }));
  });
  it("a queued acknowledgement and old socket callback cannot compact the next turn", async () => {
    const store = new DeferredStore(); const { socket, journal } = await start(store); socket.open();
    const oldMessage = socket.onmessage!;
    const ack = deferred(); const acknowledged = vi.spyOn(journal, "acknowledgeProcessed").mockImplementationOnce(async () => ack.promise);
    socket.onmessage!({ data: JSON.stringify({ type: "status", code: "processed", processedSequence: 0 }) });
    await settle();
    socket.onmessage!({ data: JSON.stringify({ type: "status", code: "processed", processedSequence: 1 }) });
    await provider!.start();
    oldMessage({ data: JSON.stringify({ type: "status", code: "processed", processedSequence: 99 }) });
    ack.resolve(); await settle();
    expect(acknowledged).toHaveBeenCalledTimes(1);
    expect(provider!.getDiagnostic().processedSequence).not.toBe(99);
  });
  it("a stalled synchronous capture burst terminates at the unchanged 16 MiB budget", async () => {
    const store = new DeferredStore(); const { socket } = await start(store); socket.open();
    frame(new Float32Array(MAX_RETAINED_TURN_AUDIO_BYTES / 2 + 1), 16_000); await settle();
    expect(store.save).not.toHaveBeenCalled(); expect(socket.send).not.toHaveBeenCalled();
    expect(provider!.getDiagnostic().state).toBe("failed");
  });
});

describe("IndexedDB transaction acceptance", () => {
  afterEach(() => vi.unstubAllGlobals());
  it.each(["complete", "abort", "error"])("request success remains provisional until transaction %s", async (outcome) => {
    const request = { result: 7, error: null, onsuccess: null, onerror: null } as unknown as IDBRequest;
    const transaction = { error: new Error("transaction failed"), objectStore: () => ({ put: () => request }) } as unknown as IDBTransaction;
    const database = { transaction: () => transaction } as unknown as IDBDatabase;
    const open = { result: database, onsuccess: null } as unknown as IDBOpenDBRequest;
    vi.stubGlobal("indexedDB", { open: () => { queueMicrotask(() => open.onsuccess?.({} as Event)); return open; } });
    const store = new IndexedDBTurnJournalStore();
    const completed = vi.fn(); const failed = vi.fn();
    const pending = store.save("test", {} as JournalSnapshot).then(completed, failed);
    await settle(); request.onsuccess?.({} as Event); await settle();
    expect(completed).not.toHaveBeenCalled(); expect(failed).not.toHaveBeenCalled();
    if (outcome === "complete") transaction.oncomplete?.({} as Event);
    else if (outcome === "abort") transaction.onabort?.({} as Event);
    else transaction.onerror?.({} as Event);
    await pending;
    expect(completed).toHaveBeenCalledTimes(outcome === "complete" ? 1 : 0);
    expect(failed).toHaveBeenCalledTimes(outcome === "complete" ? 0 : 1);
  });
});
