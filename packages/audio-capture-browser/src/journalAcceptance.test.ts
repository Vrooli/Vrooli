import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FakeVoiceTransport as Socket, stubCryptoWithoutSubtle } from "./fakeVoiceTransport";
import { IndexedDBTurnJournalStore, TurnJournal, type JournalSnapshot, type TurnJournalStore } from "./turnJournal";
import { MAX_RETAINED_TURN_AUDIO_BYTES, PcmVoiceStreamProvider } from "./pcmVoiceStreamProvider";
import { forgetUnfinishedSession, loadUnfinishedSession, rememberUnfinishedSession } from "./sessionIdentity";

function deferred<T = void>() {
  let resolve!: (value: T) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
const chunk = (sequence = 0n) => ({ sequence, startSample: sequence * 2n, endSample: sequence * 2n + 2n, audio: new ArrayBuffer(4), sha256: new ArrayBuffer(32) });
class DeferredStore implements TurnJournalStore {
  pending = deferred();
  save = vi.fn(async (_key: string, _value: JournalSnapshot) => this.pending.promise);
  load = async () => undefined;
  remove = async () => {};
}
async function settle() { for (let i = 0; i < 128; i++) await Promise.resolve(); }

describe("selected journal acceptance", () => {
  let provider: PcmVoiceStreamProvider | undefined;
  let frame!: (samples: Float32Array, rate: number) => void;
  beforeEach(() => { vi.useFakeTimers(); stubCryptoWithoutSubtle(); Socket.instances = []; Socket.autoOpen = false; });
  afterEach(() => { provider?.dispose(); provider = undefined; forgetUnfinishedSession(); vi.unstubAllGlobals(); vi.useRealTimers(); });
  async function start(store: DeferredStore, durability: "persistent" | "memory" = "persistent") {
    const journal = new TurnJournal(store, "acceptance", 0n, MAX_RETAINED_TURN_AUDIO_BYTES, durability);
    provider = new PcmVoiceStreamProvider({
      getUserMedia: async () => ({ getTracks: () => [{ readyState: "live", stop: vi.fn() }] }) as unknown as MediaStream,
      transport: new Socket(),
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
  it("overlaps admission with journal restore but sends replay only after restore", async () => {
    const restore = deferred<JournalSnapshot | undefined>();
    const store: TurnJournalStore = {
      load: vi.fn(() => restore.promise),
      save: vi.fn(async () => {}),
      remove: vi.fn(async () => {}),
    };
    const saved = chunk();
    const snapshot: JournalSnapshot = {
      sessionId: "restore-gate",
      generation: 0n,
      durability: "persistent",
      chunks: [saved],
      retainedBytes: saved.audio.byteLength,
      nextSequence: 1n,
      nextSample: saved.endSample,
    };
    const journal = new TurnJournal(store, "restore-gate", 0n, 32, "persistent");
    let captureCreated = false;
    provider = new PcmVoiceStreamProvider({
      getUserMedia: async () => ({ getTracks: () => [{ readyState: "live", stop: vi.fn() }] }) as unknown as MediaStream,
      transport: new Socket(),
      captureFactory: () => { captureCreated = true; return { stop: vi.fn() }; },
      journalFactory: () => journal,
    });
    rememberUnfinishedSession({ sessionId: "restore-gate", resumeToken: "resume-gate" });

    const starting = provider.start();
    await settle();
    expect(Socket.instances).toHaveLength(1);
    const socket = Socket.instances[0]!;
    socket.open();
    await settle();
    expect(captureCreated).toBe(false);
    expect(socket.sendAudio).not.toHaveBeenCalled();

    restore.resolve(snapshot);
    await starting;
    await settle();
    expect(captureCreated).toBe(true);
    expect(socket.sendAudio).toHaveBeenCalledOnce();
    expect(socket.sendAudio).toHaveBeenCalledWith(expect.objectContaining({ sequence: 0n, startSample: 0n, endSample: 2n }));
  });
  it.each(["restore rejection", "missing durable snapshot"])("fails closed for an unfinished identity when restore has %s", async (failure) => {
    const restore = deferred<JournalSnapshot | undefined>();
    const store: TurnJournalStore = {
      load: vi.fn(() => restore.promise),
      save: vi.fn(async () => {}),
      remove: vi.fn(async () => {}),
    };
    const identity = { sessionId: "unfinished-session", resumeToken: "unfinished-resume" };
    const trackStop = vi.fn();
    const captureFactory = vi.fn(() => ({ stop: vi.fn() }));
    const onError = vi.fn();
    provider = new PcmVoiceStreamProvider({
      getUserMedia: async () => ({ getTracks: () => [{ readyState: "live", stop: trackStop }] }) as unknown as MediaStream,
      transport: new Socket(),
      captureFactory,
      journalFactory: () => new TurnJournal(store, identity.sessionId, 0n, 32, "persistent"),
    });
    provider.onError = onError;
    rememberUnfinishedSession(identity);

    const starting = provider.start();
    await settle();
    const socket = Socket.instances[0]!;
    socket.open();
    await settle();
    expect(captureFactory).not.toHaveBeenCalled();
    expect(socket.sendAudio).not.toHaveBeenCalled();

    if (failure === "restore rejection") restore.reject(new Error("IndexedDB unavailable"));
    else restore.resolve(undefined);
    await expect(starting).rejects.toThrow("The previous voice session's recovery journal is unavailable");

    expect(socket.sendAudio).not.toHaveBeenCalled();
    expect(socket.send).not.toHaveBeenCalled();
    expect(socket.close).toHaveBeenCalledOnce();
    expect(trackStop).toHaveBeenCalledOnce();
    expect(captureFactory).not.toHaveBeenCalled();
    expect(loadUnfinishedSession()).toEqual(identity);
    expect(provider.getDiagnostic()).toMatchObject({ state: "failed", terminalReason: "journal_restore_failed" });
    expect(provider.getDiagnostic().errorCodes).toContain("journal_restore_failed");
    expect(onError).not.toHaveBeenCalled();
  });
  it("preConnect reuses an unfinished identity and keeps it when its socket expires", async () => {
    const identity = { sessionId: "unfinished-preconnect", resumeToken: "unfinished-preconnect-token" };
    rememberUnfinishedSession(identity);
    provider = new PcmVoiceStreamProvider({ transport: new Socket() });

    provider.preConnect("en");
    expect(Socket.instances[0]!.start.sessionId).toBe(identity.sessionId);
    expect(loadUnfinishedSession()).toEqual(identity);

    await vi.advanceTimersByTimeAsync(30_000);
    expect(loadUnfinishedSession()).toEqual(identity);
  });
  it("fails closed when an expired preconnect cannot restore the unfinished journal", async () => {
    const identity = { sessionId: "unfinished-expired-preconnect", resumeToken: "unfinished-expired-preconnect-token" };
    rememberUnfinishedSession(identity);
    const store: TurnJournalStore = {
      load: vi.fn(async () => { throw new Error("IndexedDB unavailable"); }),
      save: vi.fn(async () => {}),
      remove: vi.fn(async () => {}),
    };
    const captureFactory = vi.fn(() => ({ stop: vi.fn() }));
    const trackStop = vi.fn();
    provider = new PcmVoiceStreamProvider({
      transport: new Socket(),
      getUserMedia: async () => ({ getTracks: () => [{ readyState: "live", stop: trackStop }] }) as unknown as MediaStream,
      captureFactory,
      journalFactory: () => new TurnJournal(store, identity.sessionId, 0n, 32, "persistent"),
    });

    provider.preConnect("en");
    expect(Socket.instances[0]!.start.sessionId).toBe(identity.sessionId);
    await vi.advanceTimersByTimeAsync(30_000);
    expect(loadUnfinishedSession()).toEqual(identity);

    const starting = provider.start();
    await settle();
    const resumedSocket = Socket.instances.at(-1)!;
    resumedSocket.open();
    await expect(starting).rejects.toThrow("The previous voice session's recovery journal is unavailable");

    expect(resumedSocket.sendAudio).not.toHaveBeenCalled();
    expect(resumedSocket.send).not.toHaveBeenCalled();
    expect(resumedSocket.close).toHaveBeenCalled();
    expect(trackStop).toHaveBeenCalledOnce();
    expect(captureFactory).not.toHaveBeenCalled();
    expect(loadUnfinishedSession()).toEqual(identity);
    expect(provider.getDiagnostic()).toMatchObject({ state: "failed", terminalReason: "journal_restore_failed" });
  });
  it("starts microphone acquisition while restore is pending and holds capture behind the restore gate", async () => {
    const restore = deferred<JournalSnapshot | undefined>();
    const microphone = deferred<MediaStream>();
    const load = vi.fn(() => restore.promise);
    const getUserMedia = vi.fn(() => microphone.promise);
    const store: TurnJournalStore = {
      load,
      save: vi.fn(async () => {}),
      remove: vi.fn(async () => {}),
    };
    const journal = new TurnJournal(store, "overlap", 0n, 32, "persistent");
    const captureFactory = vi.fn(() => ({ stop: vi.fn() }));
    provider = new PcmVoiceStreamProvider({
      getUserMedia,
      transport: new Socket(),
      captureFactory,
      journalFactory: () => journal,
    });

    const starting = provider.start();
    await settle();
    expect(load).toHaveBeenCalledOnce();
    expect(getUserMedia).toHaveBeenCalledOnce();
    expect(captureFactory).not.toHaveBeenCalled();

    microphone.resolve({ getTracks: () => [{ readyState: "live", stop: vi.fn() }] } as unknown as MediaStream);
    await settle();
    expect(captureFactory).not.toHaveBeenCalled();
    expect(provider.getDiagnostic().state).toBe("recording");

    restore.resolve(undefined);
    await starting;
    expect(captureFactory).toHaveBeenCalledOnce();
  });
  it("fresh socket open and stop cannot overtake an unresolved persistent append", async () => {
    const store = new DeferredStore(); const { socket } = await start(store);
    frame(new Float32Array([0.5, 0.5]), 16_000); await settle();
    socket.open(); provider!.stop(); await settle();
    expect(socket.send).not.toHaveBeenCalled();
    store.pending.resolve(); await settle();
    expect(socket.send.mock.calls).toHaveLength(2);
    expect(socket.send.mock.calls[0]![0]).toMatchObject({ sequence: 0n, startSample: 0n, endSample: 2n });
    expect(socket.send.mock.calls[1]![0]).toEqual({ type: "end" });
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
    const store = new DeferredStore(); const { journal } = await start(store);
    provider!.dispose(); Socket.instances = [];
    store.load = async () => journal.read();
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
    expect(socket.send.mock.calls[1]![0]).toEqual({ type: "end" });
  });
  it("a queued acknowledgement and old socket callback cannot compact the next turn", async () => {
    const store = new DeferredStore(); const { socket, journal } = await start(store); socket.open();
    const oldMessage = socket.onmessage!;
    const ack = deferred(); const acknowledged = vi.spyOn(journal, "acknowledgeProcessed").mockImplementationOnce(async () => ack.promise);
    socket.onmessage!({ data: JSON.stringify({ type: "status", code: "processed", processedSequence: 0 }) });
    await settle();
    socket.onmessage!({ data: JSON.stringify({ type: "status", code: "processed", processedSequence: 1 }) });
    store.load = async () => journal.read();
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
    const transaction = { error: new Error("transaction failed"), objectStore: () => ({ put: () => request, get: () => request }) } as unknown as IDBTransaction;
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
