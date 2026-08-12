import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { apiBaseMock } from "../../test-utils";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { registerVoiceTransport as registerBrowserVoiceTransport } from "@vrooli/audio-capture-browser";
import { _resetMicOwnershipForTesting, getActiveMicLeases, registerMicStream, } from "../../audio-integration";
vi.mock("@vrooli/api-base", () => apiBaseMock());
const fetchCapabilitiesMock = vi.fn();
const fetchCapabilitiesLivenessCachedMock = vi.fn();
const refreshCapabilitiesLivenessMock = vi.fn();
const getCapabilitiesLivenessSnapshotMock = vi.fn(() => null);
vi.mock("../../api/capabilities", () => ({
    fetchCapabilities: fetchCapabilitiesMock,
    fetchCapabilitiesLiveness: fetchCapabilitiesLivenessCachedMock,
    fetchCapabilitiesLivenessCached: fetchCapabilitiesLivenessCachedMock,
    refreshCapabilitiesLiveness: refreshCapabilitiesLivenessMock,
    getCapabilitiesLivenessSnapshot: getCapabilitiesLivenessSnapshotMock,
}));
// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const mockCapabilities = (whisperAvailable) => {
    const resp = {
        capabilities: [
            {
                id: "audio-tools",
                status: whisperAvailable ? "available" : "unavailable",
                features: whisperAvailable ? ["voice-input", "voice-streaming"] : [],
            },
        ],
        timestamp: new Date().toISOString(),
    };
    fetchCapabilitiesMock.mockResolvedValue(resp);
    fetchCapabilitiesLivenessCachedMock.mockResolvedValue(resp);
    refreshCapabilitiesLivenessMock.mockResolvedValue(resp);
};
const mockMediaDevices = (success) => {
    const mockStream = {
        getTracks: () => [{ readyState: "live", muted: false, kind: "audio", stop: vi.fn() }],
    };
    const getUserMedia = success
        ? vi.fn().mockResolvedValue(mockStream)
        : vi.fn().mockRejectedValue(new Error("Permission denied"));
    Object.defineProperty(navigator, "mediaDevices", {
        value: {
            getUserMedia,
        },
        configurable: true,
    });
    return { getUserMedia, mockStream };
};
/** Minimal SpeechRecognition stub */
function installSpeechRecognition() {
    window.SpeechRecognition = class {
        constructor() {
            this.continuous = false;
            this.interimResults = false;
            this.lang = "";
            this.onresult = null;
            this.onerror = null;
            this.onend = null;
        }
        start() { }
        stop() { }
        abort() { }
        addEventListener() { }
        removeEventListener() { }
        dispatchEvent() {
            return false;
        }
    };
}
function removeSpeechRecognition() {
    delete window.SpeechRecognition;
    delete window.webkitSpeechRecognition;
}
// ---------------------------------------------------------------------------
// Hook integration tests — backend detection and fallback
// ---------------------------------------------------------------------------
describe("useVoiceInput", () => {
    let originalFetch;
    beforeEach(() => {
        originalFetch = globalThis.fetch;
        vi.clearAllMocks();
        registerBrowserVoiceTransport({
            buildStreamUrl: () => "ws://voice.test/stream",
            transcribeRetained: async () => "",
        });
        removeSpeechRecognition();
        useWorkspaceStore.setState({ voiceEnabled: true });
    });
    afterEach(() => {
        globalThis.fetch = originalFetch;
    });
    it("falls back to web-speech when whisper unavailable", async () => {
        mockCapabilities(false);
        installSpeechRecognition();
        mockMediaDevices(true);
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => {
            await vi.dynamicImportSettled?.();
            await new Promise((r) => setTimeout(r, 50));
        });
        expect(result.current.backend).toBe("web-speech");
        expect(result.current.supported).toBe(true);
    });
    it("uses whisper when available", async () => {
        mockCapabilities(true);
        mockMediaDevices(true);
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => {
            await new Promise((r) => setTimeout(r, 50));
        });
        expect(result.current.backend).toBe("whisper");
        expect(result.current.supported).toBe(true);
    });
    it("reports unsupported when no backend available", async () => {
        mockCapabilities(false);
        removeSpeechRecognition();
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => {
            await new Promise((r) => setTimeout(r, 50));
        });
        expect(result.current.supported).toBe(false);
        expect(result.current.backend).toBe("none");
    });
    it("disables when voiceEnabled is false", async () => {
        useWorkspaceStore.setState({ voiceEnabled: false });
        mockCapabilities(true);
        installSpeechRecognition();
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => {
            await new Promise((r) => setTimeout(r, 50));
        });
        expect(result.current.supported).toBe(false);
        expect(result.current.backend).toBe("none");
    });
    it("reports error when capabilities fetch fails", async () => {
        fetchCapabilitiesMock.mockRejectedValue(new Error("Network error"));
        fetchCapabilitiesLivenessCachedMock.mockRejectedValue(new Error("Network error"));
        refreshCapabilitiesLivenessMock.mockRejectedValue(new Error("Network error"));
        removeSpeechRecognition();
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => {
            await new Promise((r) => setTimeout(r, 50));
        });
        expect(result.current.supported).toBe(false);
        expect(result.current.backend).toBe("none");
    });
});
// ---------------------------------------------------------------------------
// WebSpeech deduplication (integration-level via hook)
// ---------------------------------------------------------------------------
/**
 * Controllable SpeechRecognition stub for testing processedResultCount
 * deduplication through the full hook lifecycle.
 */
function installControllableSpeechRecognition() {
    let instance = null;
    window.SpeechRecognition = class {
        constructor() {
            this.continuous = false;
            this.interimResults = false;
            this.lang = "";
            this.onresult = null;
            this.onerror = null;
            this.onend = null;
        }
        start() { instance = this; }
        stop() { this.onend?.(); }
        abort() { }
        addEventListener() { }
        removeEventListener() { }
        dispatchEvent() { return false; }
    };
    function fireResult(results) {
        if (!instance?.onresult)
            return;
        const resultList = results.map((r) => {
            const item = { transcript: r.transcript, confidence: 0.95 };
            return Object.assign([item], { isFinal: r.isFinal, length: 1, item: () => item });
        });
        const event = {
            results: Object.assign(resultList, {
                length: resultList.length,
                item: (i) => resultList[i],
            }),
        };
        instance.onresult(event);
    }
    return {
        getInstance: () => instance,
        fireResult,
        triggerEnd: () => instance?.onend?.(),
    };
}
describe("WebSpeechProvider deduplication (via hook)", () => {
    let originalFetch;
    beforeEach(() => {
        originalFetch = globalThis.fetch;
        vi.clearAllMocks();
        registerBrowserVoiceTransport({
            buildStreamUrl: () => "ws://voice.test/stream",
            transcribeRetained: async () => "",
        });
        removeSpeechRecognition();
        useWorkspaceStore.setState({ voiceEnabled: true });
    });
    afterEach(() => {
        globalThis.fetch = originalFetch;
    });
    it("dispatches only new final results, not cumulative duplicates", async () => {
        mockCapabilities(false);
        mockMediaDevices(true);
        const ctrl = installControllableSpeechRecognition();
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
        expect(result.current.backend).toBe("web-speech");
        await act(async () => { await result.current.startRecording(); });
        act(() => {
            ctrl.fireResult([{ transcript: "hello", isFinal: true }]);
        });
        expect(onTranscript).toHaveBeenCalledTimes(1);
        expect(onTranscript).toHaveBeenLastCalledWith("hello");
        act(() => {
            ctrl.fireResult([
                { transcript: "hello", isFinal: true },
                { transcript: " world", isFinal: true },
            ]);
        });
        expect(onTranscript).toHaveBeenCalledTimes(2);
        expect(onTranscript).toHaveBeenLastCalledWith("world");
    });
    it("interim results update partialTranscript but do not dispatch as final", async () => {
        mockCapabilities(false);
        mockMediaDevices(true);
        const ctrl = installControllableSpeechRecognition();
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
        await act(async () => { await result.current.startRecording(); });
        act(() => {
            ctrl.fireResult([{ transcript: "hel", isFinal: false }]);
        });
        expect(onTranscript).not.toHaveBeenCalled();
        expect(result.current.partialTranscript).toBe("hel");
    });
    it("processedResultCount persists across spontaneous recognition restarts", async () => {
        mockCapabilities(false);
        mockMediaDevices(true);
        const ctrl = installControllableSpeechRecognition();
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
        await act(async () => { await result.current.startRecording(); });
        act(() => {
            ctrl.fireResult([{ transcript: "hello", isFinal: true }]);
        });
        expect(onTranscript).toHaveBeenCalledTimes(1);
        act(() => { ctrl.triggerEnd(); });
        act(() => {
            ctrl.fireResult([
                { transcript: "hello", isFinal: true },
                { transcript: " world", isFinal: true },
            ]);
        });
        expect(onTranscript).toHaveBeenCalledTimes(2);
        expect(onTranscript).toHaveBeenLastCalledWith("world");
    });
});
// ---------------------------------------------------------------------------
// Capture lifecycle ownership (mic-lease honesty + self-healing)
// ---------------------------------------------------------------------------
/** SpeechRecognition stub that can fire onerror, for the error-path test. */
function installErrorableSpeechRecognition() {
    let instance = null;
    window.SpeechRecognition = class {
        constructor() {
            this.onresult = null;
            this.onerror = null;
            this.onend = null;
            this.continuous = false;
            this.interimResults = false;
            this.lang = "";
        }
        start() { instance = this; }
        stop() { }
        abort() { }
        addEventListener() { }
        removeEventListener() { }
        dispatchEvent() { return false; }
    };
    return { fireError: (error) => instance?.onerror?.({ error, message: error }) };
}
describe("voice capture lifecycle ownership", () => {
    let originalFetch;
    beforeEach(() => {
        originalFetch = globalThis.fetch;
        vi.clearAllMocks();
        removeSpeechRecognition();
        _resetMicOwnershipForTesting();
        useWorkspaceStore.setState({ voiceEnabled: true });
    });
    afterEach(() => {
        globalThis.fetch = originalFetch;
        _resetMicOwnershipForTesting();
    });
    it("a provider error while recording releases the mic lease and returns the UI to idle", async () => {
        mockCapabilities(false); // web-speech backend (testable without MediaRecorder/WS)
        mockMediaDevices(true);
        const ctrl = installErrorableSpeechRecognition();
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
        expect(result.current.backend).toBe("web-speech");
        await act(async () => { await result.current.startRecording(); });
        // The provider acquired a registry mic lease for capture.
        expect(getActiveMicLeases().length).toBeGreaterThan(0);
        await act(async () => { ctrl.fireError("network"); });
        // Idle UI AND no live mic lease — the error path disposed the provider.
        expect(result.current.voiceState).toBe("idle");
        expect(getActiveMicLeases()).toHaveLength(0);
    });
    it("self-heals an orphaned live mic lease while the UI is idle (registry-vs-UI mismatch)", async () => {
        mockCapabilities(true); // whisper backend; idle, not recording
        mockMediaDevices(true);
        const warn = vi.spyOn(console, "warn").mockImplementation(() => { });
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
        expect(result.current.voiceState).toBe("idle");
        // Inject the bug: a live active-recording lease appears while the UI is idle.
        await act(async () => {
            registerMicStream("voice-stream", {
                getTracks: () => [{ readyState: "live", muted: false, kind: "audio", stop: vi.fn(), addEventListener() { }, removeEventListener() { } }],
            });
        });
        // The registry subscription detected and self-healed the orphan.
        expect(getActiveMicLeases()).toHaveLength(0);
        expect(warn).toHaveBeenCalledWith(expect.stringContaining("INVARIANT VIOLATION"), expect.anything(), expect.anything());
        warn.mockRestore();
    });
    it("does not acquire the mic on mount or visibility with persisted wake-word flag", async () => {
        mockCapabilities(true);
        const media = mockMediaDevices(true);
        useWorkspaceStore.setState({
            voiceEnabled: true,
            wakeWordEnabled: true,
            persistentMode: false,
        });
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
        expect(result.current.backend).toBe("whisper");
        expect(media.getUserMedia).not.toHaveBeenCalled();
        expect(getActiveMicLeases()).toHaveLength(0);
        await act(async () => {
            Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
            document.dispatchEvent(new Event("visibilitychange"));
            await Promise.resolve();
        });
        expect(media.getUserMedia).not.toHaveBeenCalled();
        expect(getActiveMicLeases()).toHaveLength(0);
    });
    it("prepareRecording never acquires the mic (no pre-warm / no idle mic hold)", async () => {
        // Regression: low-latency pre-warm was removed. Signalling intent must NOT
        // open the microphone — holding the mic idle ducks other apps' audio and
        // churns the iOS audio session. The mic is acquired only on actual record.
        mockCapabilities(true);
        const media = mockMediaDevices(true);
        useWorkspaceStore.setState({
            voiceEnabled: true,
            wakeWordEnabled: false,
            persistentMode: false,
        });
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
        expect(media.getUserMedia).not.toHaveBeenCalled();
        await act(async () => {
            result.current.prepareRecording();
            await new Promise((r) => setTimeout(r, 50));
        });
        // Still no mic — prepare only arms intent / passive reconcile.
        expect(media.getUserMedia).not.toHaveBeenCalled();
        expect(getActiveMicLeases()).toHaveLength(0);
    });
});
class FinalFrameWebSocket {
    constructor(url) {
        this.url = url;
        this.readyState = FinalFrameWebSocket.CONNECTING;
        this.onopen = null;
        this.onclose = null;
        this.onerror = null;
        this.onmessage = null;
        this.send = vi.fn();
        this.close = vi.fn(() => {
            this.readyState = FinalFrameWebSocket.CLOSED;
            this.onclose?.();
        });
        FinalFrameWebSocket.instances.push(this);
        queueMicrotask(() => {
            this.readyState = FinalFrameWebSocket.OPEN;
            this.onopen?.();
        });
    }
    emitFinal(text) {
        this.onmessage?.({ data: JSON.stringify({ type: "final", text }) });
    }
}
FinalFrameWebSocket.OPEN = 1;
FinalFrameWebSocket.CLOSED = 3;
FinalFrameWebSocket.CONNECTING = 0;
FinalFrameWebSocket.instances = [];
class FinalFrameMediaRecorder {
    constructor(_stream, _opts) {
        this.state = "inactive";
        this.ondataavailable = null;
        this.onstop = null;
        this.start = vi.fn(() => {
            this.state = "recording";
        });
        this.stop = vi.fn(() => {
            this.state = "inactive";
            this.onstop?.();
        });
        FinalFrameMediaRecorder.instances.push(this);
    }
}
FinalFrameMediaRecorder.instances = [];
FinalFrameMediaRecorder.isTypeSupported = vi.fn(() => true);
function installFinalFrameBrowserFakes() {
    FinalFrameWebSocket.instances = [];
    FinalFrameMediaRecorder.instances = [];
    globalThis.WebSocket = FinalFrameWebSocket;
    globalThis.MediaRecorder = FinalFrameMediaRecorder;
    const tracks = [];
    const getUserMedia = vi.fn(async () => {
        const track = {
            kind: "audio",
            muted: false,
            readyState: "live",
            stop: vi.fn(function stop() {
                this.readyState = "ended";
            }),
            addEventListener() { },
            removeEventListener() { },
        };
        tracks.push(track);
        return {
            active: true,
            getTracks: () => [track],
            getAudioTracks: () => [track],
        };
    });
    Object.defineProperty(navigator, "mediaDevices", {
        configurable: true,
        value: { getUserMedia },
    });
    return { getUserMedia, tracks };
}
describe("voice capture lifecycle server-final wedge", () => {
    let originalFetch;
    beforeEach(() => {
        originalFetch = globalThis.fetch;
        vi.clearAllMocks();
        removeSpeechRecognition();
        _resetMicOwnershipForTesting();
        useWorkspaceStore.setState({
            voiceEnabled: true,
            wakeWordEnabled: false,
            persistentMode: false,
        });
    });
    afterEach(() => {
        globalThis.fetch = originalFetch;
        _resetMicOwnershipForTesting();
    });
    it("does not accept a premature server final as a successful turn", async () => {
        mockCapabilities(true);
        const browser = installFinalFrameBrowserFakes();
        const onTranscript = vi.fn();
        const { useScenarioVoiceInput: useVoiceInput } = await import("../../audio-integration/hooks/useScenarioVoiceInput");
        const { result } = renderHook(() => useVoiceInput(onTranscript));
        await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
        expect(result.current.backend).toBe("whisper");
        await act(async () => { await result.current.startRecording(); });
        await waitFor(() => expect(result.current.voiceState).toBe("recording"));
        expect(getActiveMicLeases()).toHaveLength(1);
        const firstSocket = FinalFrameWebSocket.instances.at(-1);
        if (!firstSocket)
            throw new Error("expected streaming socket fake to exist");
        expect(browser.tracks[0]?.readyState).toBe("live");
        await act(async () => {
            firstSocket.emitFinal("server finished");
            await Promise.resolve();
        });
        // A server final while the browser is still recording is a degraded
        // transport signal, not a successful turn. The provider reconnects and
        // replays retained PCM so the words are not silently discarded.
        await waitFor(() => expect(result.current.streamingDegradationNotice).toContain("recovering retained audio"));
        expect(result.current.voiceState).toBe("recording");
        expect(onTranscript).not.toHaveBeenCalled();
        expect(getActiveMicLeases()).toHaveLength(1);
        expect(browser.tracks[0]?.readyState).toBe("live");
        await act(async () => { result.current.releaseMicrophone(); });
        await waitFor(() => expect(result.current.voiceState).toBe("idle"));
        expect(getActiveMicLeases()).toHaveLength(0);
        expect(browser.tracks[0]?.stop).toHaveBeenCalled();
        await act(async () => { await result.current.startRecording(); });
        await waitFor(() => expect(result.current.voiceState).toBe("recording"));
        expect(browser.getUserMedia).toHaveBeenCalledTimes(2);
        expect(getActiveMicLeases()).toHaveLength(1);
    });
});
