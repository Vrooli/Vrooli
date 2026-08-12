import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
const captureMocks = vi.hoisted(() => {
    const journals = [];
    return {
        captureFrame: null,
        encodedFrames: vi.fn(),
        journals,
    };
});
vi.mock("../../api/voice", () => ({
    buildVoiceStreamWsUrl: (language, sessionId, resumeToken) => `ws://voice.test/stream?language=${language ?? ""}&session_id=${sessionId ?? ""}&resume_token=${resumeToken ?? ""}&protocol_version=2`,
    transcribeAudioWithRetry: vi.fn(),
}));
vi.mock("./sharedAudioContext", () => ({ getSharedAudioContext: vi.fn(() => ({})) }));
vi.mock("@vrooli/audio-capture-browser", async (importOriginal) => {
    const actual = await importOriginal();
    class TurnJournal {
        constructor() {
            this.append = vi.fn(async () => { });
            this.acknowledgeProcessed = vi.fn(async () => { });
            this.replayAfter = vi.fn(() => []);
            this.discard = vi.fn(async () => { });
            this.read = vi.fn(() => ({ chunks: [], nextSequence: 0n, nextSample: 0n }));
            this.restore = vi.fn(async () => ({ chunks: [], nextSequence: 0n, nextSample: 0n }));
            captureMocks.journals.push(this);
        }
    }
    class StreamDiagnosticRecorder {
        constructor(sessionId = "", generation = 0, durability = "reduced") {
            this.snapshot = { sessionId: "", generation: 0, durability: "reduced", state: "preparing", capturedSequence: -1, sentSequence: -1, processedSequence: -1, statusCodes: [], errorCodes: [], events: [] };
            this.reset(sessionId, generation, durability);
        }
        reset(sessionId, generation, durability) { this.snapshot = { sessionId, generation, durability, state: "preparing", capturedSequence: -1, sentSequence: -1, processedSequence: -1, statusCodes: [], errorCodes: [], events: [] }; }
        state(state, code = state) { this.snapshot.state = state; this.snapshot.events.push({ kind: "state", code }); }
        captured(sequence) { this.snapshot.capturedSequence = Number(sequence); }
        sent(sequence) { this.snapshot.sentSequence = Number(sequence); }
        processed(sequence) { this.snapshot.processedSequence = Number(sequence); }
        status(code) { this.snapshot.statusCodes.push(code); }
        error(code) { this.snapshot.errorCodes.push(code); }
        terminal(state, code) { this.snapshot.state = state; this.snapshot.events.push({ kind: "terminal", code }); }
        read() { return { ...this.snapshot, statusCodes: [...this.snapshot.statusCodes], errorCodes: [...this.snapshot.errorCodes], events: [...this.snapshot.events] }; }
        exportJSON() { return JSON.stringify(this.read()); }
    }
    return {
        ...actual,
        TurnJournal,
        StreamDiagnosticRecorder,
        IndexedDBTurnJournalStore: class {
        },
        MemoryTurnJournalStore: class {
        },
        TARGET_SAMPLE_RATE: 16000,
        concatInt16: (parts) => parts[0] ?? new Int16Array(),
        createCanonicalPcmCapture: async (_context, _stream, onFrame) => {
            captureMocks.captureFrame = onFrame;
            return { stop: vi.fn() };
        },
        digestAudio: async () => new Uint8Array(32).buffer,
        encodeAudioFrame: (frame) => {
            captureMocks.encodedFrames(frame);
            return new ArrayBuffer(1);
        },
        encodeWavFromPcm16: () => new Blob(),
        frameToCanonicalPcm16: () => new Int16Array([1, 2, 3]),
        forgetUnfinishedSession: vi.fn(),
        loadUnfinishedSession: () => null,
        newSessionIdentity: (() => {
            let next = 0;
            return () => `identity-${++next}`;
        })(),
        rememberUnfinishedSession: vi.fn(),
        dispatchStreamMessage: (raw, handlers, delivered) => {
            const message = JSON.parse(raw);
            if (message.type === "status")
                handlers.onStatus?.(message.code ?? "stream_status", message.text ?? "Streaming transcription status updated.", message.processedSequence === undefined ? undefined : BigInt(message.processedSequence));
            else if (message.type === "final")
                handlers.onFinal?.(message.text?.trim() ?? "");
            else if (message.type === "error")
                handlers.onError?.(message.code ?? "provider_failure", message.text ?? "Streaming provider failed.");
            else if (message.type === "segment-final" && message.text !== undefined && (!message.segmentId || !delivered.has(message.segmentId))) {
                if (message.segmentId)
                    delivered.add(message.segmentId);
                handlers.onSegmentFinal?.(message.text, message.segmentIndex ?? 0);
            }
        },
    };
});
import { buildVoiceStreamWsUrl } from "../../api/voice";
import { TurnJournal } from "@vrooli/audio-capture-browser";
import { PcmVoiceStreamProvider } from "./PcmVoiceStreamProvider";
class FakeWebSocket {
    constructor(url) {
        this.url = url;
        this.readyState = FakeWebSocket.OPEN;
        this.onopen = null;
        this.onclose = null;
        this.onmessage = null;
        this.send = vi.fn();
        this.close = vi.fn(() => { this.readyState = FakeWebSocket.CLOSED; });
        FakeWebSocket.instances.push(this);
        queueMicrotask(() => this.onopen?.());
    }
}
FakeWebSocket.OPEN = 1;
FakeWebSocket.CLOSED = 3;
FakeWebSocket.CONNECTING = 0;
FakeWebSocket.instances = [];
function fakeStream() {
    return { getTracks: () => [{ readyState: "live", stop: vi.fn() }] };
}
async function settle() {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
}
describe("PcmVoiceStreamProvider", () => {
    let provider;
    beforeEach(() => {
        FakeWebSocket.instances = [];
        captureMocks.captureFrame = null;
        captureMocks.encodedFrames.mockClear();
        captureMocks.journals.splice(0);
        globalThis.WebSocket = FakeWebSocket;
        Object.defineProperty(navigator, "mediaDevices", {
            configurable: true,
            value: { getUserMedia: vi.fn(async () => fakeStream()) },
        });
        provider = new PcmVoiceStreamProvider({
            transport: {
                buildStreamUrl: (language, sessionId, resumeToken) => buildVoiceStreamWsUrl(language, sessionId, resumeToken),
                transcribeRetained: async () => "",
            },
            captureFactory: async (_stream, onFrame) => {
                captureMocks.captureFrame = onFrame;
                return { stop: vi.fn() };
            },
            journalFactory: () => new TurnJournal(),
        });
    });
    afterEach(() => provider.dispose());
    it("uses a durable v2 session and journals PCM before sending its first frame", async () => {
        await provider.start();
        await settle();
        const ws = FakeWebSocket.instances.at(-1);
        if (!ws || !captureMocks.captureFrame)
            throw new Error("expected an active PCM stream");
        expect(ws.url).toContain("protocol_version=2");
        expect(ws.url).toMatch(/session_id=[^&]+/);
        captureMocks.captureFrame(new Float32Array([0.1, 0.2]), 48000);
        await settle();
        expect(captureMocks.journals[0]?.append).toHaveBeenCalledOnce();
        expect(ws.send).toHaveBeenCalledWith(expect.any(ArrayBuffer));
        expect(ws.send).toHaveBeenCalledWith(expect.any(ArrayBuffer));
    });
    it("acknowledges processed durable chunks and drops post-verdict PCM", async () => {
        await provider.start();
        await settle();
        const ws = FakeWebSocket.instances.at(-1);
        if (!ws || !captureMocks.captureFrame)
            throw new Error("expected an active PCM stream");
        captureMocks.captureFrame(new Float32Array([0.1]), 16000);
        await settle();
        ws.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 0 }) });
        await settle();
        expect(captureMocks.journals[0]?.acknowledgeProcessed).toHaveBeenCalledWith(0n);
        const beforeDrop = captureMocks.encodedFrames.mock.calls.length;
        provider.dropTail();
        captureMocks.captureFrame(new Float32Array([0.2]), 16000);
        await settle();
        expect(captureMocks.encodedFrames).toHaveBeenCalledTimes(beforeDrop);
    });
    it("[REQ:ATD-P1-002] exposes a metadata-only terminal diagnostic with coverage", async () => {
        await provider.start();
        await settle();
        const ws = FakeWebSocket.instances.at(-1);
        if (!ws || !captureMocks.captureFrame)
            throw new Error("expected an active PCM stream");
        captureMocks.captureFrame(new Float32Array([0.1]), 16000);
        await settle();
        ws.onmessage?.({ data: JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 0 }) });
        provider.stop();
        ws.onmessage?.({ data: JSON.stringify({ type: "final", text: "private transcript" }) });
        expect(provider.getDiagnostic()).toMatchObject({ capturedSequence: 0, processedSequence: 0, state: "completed" });
        expect(provider.exportDiagnostic()).not.toContain("private transcript");
    });
    it("[REQ:ATD-P0-001] delivers a replayed durable segment identity once", async () => {
        const onSegmentFinal = vi.fn();
        provider.onSegmentFinal = onSegmentFinal;
        await provider.start();
        await settle();
        const ws = FakeWebSocket.instances.at(-1);
        if (!ws)
            throw new Error("expected an active PCM stream");
        const durableSegment = { type: "segment-final", text: "replayed once", segmentIndex: 0, segmentId: "turn-1:0:0:1" };
        ws.onmessage?.({ data: JSON.stringify(durableSegment) });
        ws.onmessage?.({ data: JSON.stringify(durableSegment) });
        expect(onSegmentFinal).toHaveBeenCalledTimes(1);
        expect(onSegmentFinal).toHaveBeenCalledWith("replayed once", 0);
    });
    it("[REQ:ATD-P0-004] retains the journal when a final follows incomplete coverage", async () => {
        const onResult = vi.fn();
        const onError = vi.fn();
        provider.onResult = onResult;
        provider.onError = onError;
        await provider.start();
        await settle();
        const ws = FakeWebSocket.instances.at(-1);
        if (!ws || !captureMocks.captureFrame)
            throw new Error("expected an active PCM stream");
        captureMocks.captureFrame(new Float32Array([0.1]), 16000);
        await settle();
        ws.onmessage?.({ data: JSON.stringify({ type: "error", code: "incomplete_coverage", text: "coverage incomplete" }) });
        ws.onmessage?.({ data: JSON.stringify({ type: "final", text: "must not be delivered" }) });
        await settle();
        expect(onError).toHaveBeenCalledWith("coverage incomplete");
        expect(onResult).not.toHaveBeenCalled();
        expect(captureMocks.journals[0]?.discard).not.toHaveBeenCalled();
        expect(provider.getDiagnostic()).toMatchObject({ state: "failed" });
    });
});
