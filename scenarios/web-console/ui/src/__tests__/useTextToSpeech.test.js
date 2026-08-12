import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
// Must set up speechSynthesis BEFORE the hook module is imported,
// because the hook evaluates `browserSupported` at module load time.
const mockSynthSpeak = vi.fn();
const mockSynthCancel = vi.fn();
const mockSynthPause = vi.fn();
const mockSynthResume = vi.fn();
const mockSynthGetVoices = vi.fn().mockReturnValue([]);
Object.defineProperty(window, "speechSynthesis", {
    value: {
        speak: mockSynthSpeak,
        cancel: mockSynthCancel,
        pause: mockSynthPause,
        resume: mockSynthResume,
        getVoices: mockSynthGetVoices,
        speaking: false,
        paused: false,
        onvoiceschanged: null,
    },
    writable: true,
    configurable: true,
});
class FakeUtterance {
    constructor(text) {
        this.rate = 1;
        this.pitch = 1;
        this.voice = null;
        this.onend = null;
        this.onerror = null;
        this.text = text;
    }
}
Object.defineProperty(globalThis, "SpeechSynthesisUtterance", {
    value: FakeUtterance,
    writable: true,
    configurable: true,
});
// Mock api module — include the cached wrapper the hook actually imports
const { _mockFetchCaps } = vi.hoisted(() => ({
    _mockFetchCaps: vi.fn(),
}));
vi.mock("../api/capabilities", () => ({
    fetchCapabilitiesLiveness: _mockFetchCaps,
    fetchCapabilitiesLivenessCached: (...args) => _mockFetchCaps(...args),
    _resetCapabilitiesCache: vi.fn(),
}));
// The core hooks inside audio-integration import the TTS API helpers from
// the barrel via "../index". Vitest's mock of "../audio-integration"
// intercepts the test's own imports of that barrel, but does NOT intercept
// the barrel-relative imports from sibling files inside the same package.
// Share a single set of vi.fn() instances across both mocks so a single
// mockResolvedValue() call satisfies both code paths.
const sharedTTSMocks = vi.hoisted(() => ({
    fetchCachedTTS: vi.fn(),
    getTTSVoices: vi.fn(),
    synthesizeTTS: vi.fn(),
    synthesizeTTSWithMetrics: vi.fn(),
}));
vi.mock("../audio-integration", async (importOriginal) => {
    const actual = await importOriginal();
    return {
        ...actual,
        fetchCachedTTS: sharedTTSMocks.fetchCachedTTS,
        getTTSVoices: sharedTTSMocks.getTTSVoices,
        synthesizeTTS: sharedTTSMocks.synthesizeTTS,
        synthesizeTTSWithMetrics: sharedTTSMocks.synthesizeTTSWithMetrics,
    };
});
vi.mock("../audio-integration/api/tts", async (importOriginal) => {
    const actual = await importOriginal();
    return {
        ...actual,
        fetchCachedTTS: sharedTTSMocks.fetchCachedTTS,
        getTTSVoices: sharedTTSMocks.getTTSVoices,
        synthesizeTTS: sharedTTSMocks.synthesizeTTS,
        synthesizeTTSWithMetrics: sharedTTSMocks.synthesizeTTSWithMetrics,
    };
});
vi.mock("../api/ttsHook", () => ({
    recordTTSPlaybackEvent: vi.fn().mockResolvedValue(undefined),
}));
import { fetchCapabilitiesLiveness } from "../api/capabilities";
import { fetchCachedTTS, getTTSVoices, synthesizeTTS } from "../audio-integration";
import { useTextToSpeech } from "../hooks/useTextToSpeech";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
const mockFetchCaps = fetchCapabilitiesLiveness;
const mockFetchCachedTTS = fetchCachedTTS;
const mockGetVoices = getTTSVoices;
const mockSynthesizeTTS = synthesizeTTS;
const defaultSettings = {
    voice: "",
    rate: 1.0,
    pitch: 1.0,
    kokoroVoice: "af_heart",
    kokoroSpeed: 1.0,
    backendPreference: "auto",
};
function audioToolsCapability(status) {
    return {
        id: "audio-tools",
        status,
        features: ["voice-output", "tts-summarization", "tts-cache"],
    };
}
beforeEach(() => {
    vi.clearAllMocks();
    sharedTTSMocks.synthesizeTTSWithMetrics.mockImplementation(async (...args) => ({
        blob: await sharedTTSMocks.synthesizeTTS(...args),
        metrics: { requestId: "test-request", synthStartMs: 0, totalChars: String(args[0] ?? "").length },
    }));
    useWorkspaceStore.setState({ startMutedOnLoad: true });
});
afterEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
    // Re-setup speechSynthesis mock after clearing (vi.clearAllMocks resets fn implementations)
    mockSynthGetVoices.mockReturnValue([]);
});
describe("useTextToSpeech", () => {
    function unlockBrowserAudio() {
        act(() => {
            window.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
        });
    }
    it("selects kokoro backend when capability is available", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [audioToolsCapability("available")],
            timestamp: new Date().toISOString(),
        });
        mockGetVoices.mockResolvedValue([
            { id: "af_heart", name: "af_heart" },
            { id: "bf_emma", name: "bf_emma" },
        ]);
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("kokoro");
        });
        expect(result.current.supported).toBe(true);
        expect(result.current.voices).toHaveLength(2);
    });
    it("falls back to browser when kokoro is unavailable", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [audioToolsCapability("unavailable")],
            timestamp: new Date().toISOString(),
        });
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("browser");
        });
        expect(result.current.supported).toBe(true);
    });
    it("falls back to browser when capabilities fetch fails", async () => {
        mockFetchCaps.mockRejectedValue(new Error("Network error"));
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("browser");
        });
        expect(result.current.supported).toBe(true);
    });
    it("uses browser directly when backendPreference is browser", async () => {
        const settings = { ...defaultSettings, backendPreference: "browser" };
        const { result } = renderHook(() => useTextToSpeech(settings));
        await waitFor(() => {
            expect(result.current.backend).toBe("browser");
        });
        // Should not have called capabilities check
        expect(mockFetchCaps).not.toHaveBeenCalled();
    });
    it("stop resets isSpeaking state", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [],
            timestamp: new Date().toISOString(),
        });
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("browser");
        });
        act(() => result.current.stop());
        expect(result.current.isSpeaking).toBe(false);
    });
    it("speak triggers provider and sets isSpeaking", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [],
            timestamp: new Date().toISOString(),
        });
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("browser");
        });
        const browserUtterances = [];
        mockSynthSpeak.mockImplementation((u) => {
            browserUtterances.push(u);
        });
        unlockBrowserAudio();
        act(() => result.current.speak("hello world"));
        expect(result.current.isSpeaking).toBe(true);
        await waitFor(() => {
            expect(browserUtterances).toHaveLength(1);
        });
        await act(async () => {
            browserUtterances.shift()?.onend?.();
            await Promise.resolve();
        });
        await waitFor(() => {
            expect(result.current.isSpeaking).toBe(false);
        });
    });
    it("speakParagraphs speaks multiple paragraphs sequentially", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [],
            timestamp: new Date().toISOString(),
        });
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("browser");
        });
        const browserUtterances = [];
        mockSynthSpeak.mockImplementation((u) => {
            browserUtterances.push(u);
        });
        unlockBrowserAudio();
        let playback;
        await act(async () => {
            playback = result.current.speakParagraphs(["para 1", "para 2"]);
            await Promise.resolve();
        });
        await waitFor(() => {
            expect(browserUtterances).toHaveLength(1);
        });
        await act(async () => {
            browserUtterances.shift()?.onend?.();
            await Promise.resolve();
        });
        await waitFor(() => {
            expect(browserUtterances).toHaveLength(1);
        });
        await act(async () => {
            browserUtterances.shift()?.onend?.();
            await Promise.resolve();
        });
        await act(async () => {
            await playback;
        });
        await waitFor(() => {
            expect(result.current.isSpeaking).toBe(false);
        });
        // Both paragraphs should have been spoken (cancel + speak for each)
        expect(mockSynthSpeak).toHaveBeenCalledTimes(2);
    });
    it("speak() falls back to browser when kokoro synthesis fails at runtime", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [{ id: "kokoro-tts", status: "available" }],
            timestamp: new Date().toISOString(),
        });
        mockGetVoices.mockResolvedValue([
            { id: "af_heart", name: "af_heart" },
        ]);
        mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));
        // Browser fallback should complete via onend
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("kokoro");
        });
        const browserUtterances = [];
        mockSynthSpeak.mockImplementation((u) => {
            browserUtterances.push(u);
        });
        unlockBrowserAudio();
        act(() => result.current.speak("hello"));
        // The kokoro provider rejects, triggering browser fallback
        await waitFor(() => {
            expect(mockSynthSpeak).toHaveBeenCalled();
        });
        await waitFor(() => {
            expect(browserUtterances).toHaveLength(1);
        });
        await act(async () => {
            browserUtterances.shift()?.onend?.();
            await Promise.resolve();
        });
        // isSpeaking resets after the error handler runs (kokoro path sets it false on rejection)
        await waitFor(() => {
            expect(result.current.isSpeaking).toBe(false);
        });
    });
    it("runtime fallback does not crash when both backends fail", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [{ id: "kokoro-tts", status: "available" }],
            timestamp: new Date().toISOString(),
        });
        mockGetVoices.mockResolvedValue([
            { id: "af_heart", name: "af_heart" },
        ]);
        mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));
        // Browser fallback also fails via onerror
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("kokoro");
        });
        const browserUtterances = [];
        mockSynthSpeak.mockImplementation((u) => {
            browserUtterances.push(u);
        });
        unlockBrowserAudio();
        // Should not throw an unhandled rejection
        act(() => result.current.speak("hello"));
        await waitFor(() => {
            expect(mockSynthSpeak).toHaveBeenCalled();
        });
        await waitFor(() => {
            expect(browserUtterances).toHaveLength(1);
        });
        await act(async () => {
            browserUtterances.shift()?.onerror?.(new Error("Browser speech failed"));
            await Promise.resolve();
        });
        await waitFor(() => {
            expect(result.current.isSpeaking).toBe(false);
        });
        // Verify both backends were attempted
        expect(mockSynthesizeTTS).toHaveBeenCalled();
        expect(mockSynthSpeak).toHaveBeenCalled();
    });
    it("defaults kokoro voices to af_heart when voice fetch fails", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [{ id: "kokoro-tts", status: "available" }],
            timestamp: new Date().toISOString(),
        });
        mockGetVoices.mockRejectedValue(new Error("Failed"));
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("kokoro");
        });
        expect(result.current.voices).toEqual([{ id: "af_heart", name: "af_heart" }]);
    });
    it("uses strict kokoro mode without silently falling back to browser", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [{ id: "kokoro-tts", status: "unavailable" }],
            timestamp: new Date().toISOString(),
        });
        const { result } = renderHook(() => useTextToSpeech({
            ...defaultSettings,
            backendPreference: "kokoro",
        }));
        await waitFor(() => {
            expect(result.current.backend).toBe("none");
            expect(result.current.backendReason).toContain("Kokoro backend was selected explicitly");
        });
    });
    it("speakParagraphs falls back to browser in auto mode when kokoro fails", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [{ id: "kokoro-tts", status: "available" }],
            timestamp: new Date().toISOString(),
        });
        mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
        mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("kokoro");
        });
        const browserUtterances = [];
        mockSynthSpeak.mockImplementation((u) => {
            browserUtterances.push(u);
        });
        unlockBrowserAudio();
        let playback;
        await act(async () => {
            playback = result.current.speakParagraphs(["para 1", "para 2"]);
            await Promise.resolve();
        });
        // Resilient speakSequence (Phase 5) degrades PER PARAGRAPH: each paragraph
        // whose Kokoro synth fails (after a retry) falls back to the browser voice
        // independently and the sequence continues. So both paragraphs surface as
        // their own browser utterance; drain them one at a time.
        for (let i = 0; i < 2; i++) {
            await waitFor(() => {
                expect(browserUtterances.length).toBeGreaterThanOrEqual(i + 1);
            });
            await act(async () => {
                browserUtterances[i]?.onend?.();
                await Promise.resolve();
            });
        }
        await act(async () => {
            await playback;
        });
        // Every paragraph still played — via the browser voice — and playback ended.
        expect(mockSynthSpeak).toHaveBeenCalledTimes(2);
        await waitFor(() => {
            expect(result.current.isSpeaking).toBe(false);
        });
    });
    it("pause and resume control the active fallback browser provider after runtime kokoro failure", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [{ id: "kokoro-tts", status: "available" }],
            timestamp: new Date().toISOString(),
        });
        mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
        mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("kokoro");
        });
        const browserUtterances = [];
        mockSynthSpeak.mockImplementation((u) => {
            browserUtterances.push(u);
        });
        unlockBrowserAudio();
        act(() => result.current.speak("fallback speech"));
        await waitFor(() => {
            expect(browserUtterances).toHaveLength(1);
        });
        act(() => result.current.pause());
        expect(mockSynthPause).toHaveBeenCalledTimes(1);
        expect(result.current.isPaused).toBe(true);
        act(() => result.current.resume());
        expect(mockSynthResume).toHaveBeenCalledTimes(1);
        expect(result.current.isPaused).toBe(false);
        await act(async () => {
            browserUtterances.shift()?.onend?.();
            await Promise.resolve();
        });
        await waitFor(() => {
            expect(result.current.isSpeaking).toBe(false);
        });
    });
    it("playback state snapshots come from the active fallback browser provider", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [{ id: "kokoro-tts", status: "available" }],
            timestamp: new Date().toISOString(),
        });
        mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
        mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("kokoro");
        });
        const browserUtterances = [];
        mockSynthSpeak.mockImplementation((u) => {
            browserUtterances.push(u);
        });
        unlockBrowserAudio();
        act(() => result.current.speak("fallback snapshot"));
        await waitFor(() => {
            expect(browserUtterances).toHaveLength(1);
        });
        let playback = result.current.getPlaybackState();
        expect(playback?.capabilities.canPause).toBe(true);
        expect(playback?.capabilities.canSeek).toBe(false);
        act(() => result.current.pause());
        playback = result.current.getPlaybackState();
        expect(playback?.isPaused).toBe(true);
        await act(async () => {
            browserUtterances.shift()?.onend?.();
            await Promise.resolve();
        });
    });
    it("reports browser audio readiness after user interaction", async () => {
        mockFetchCaps.mockResolvedValue({
            capabilities: [],
            timestamp: new Date().toISOString(),
        });
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("browser");
        });
        expect(result.current.browserAudioReady).toBe(false);
        unlockBrowserAudio();
        await waitFor(() => {
            expect(result.current.browserAudioReady).toBe(true);
        });
    });
    it("tracks the start-muted preference before the user explicitly changes mute", async () => {
        useWorkspaceStore.setState({ startMutedOnLoad: false });
        mockFetchCaps.mockResolvedValue({
            capabilities: [],
            timestamp: new Date().toISOString(),
        });
        const { result } = renderHook(() => useTextToSpeech(defaultSettings));
        await waitFor(() => {
            expect(result.current.backend).toBe("browser");
        });
        expect(result.current.isMuted).toBe(false);
        act(() => {
            useWorkspaceStore.setState({ startMutedOnLoad: true });
        });
        await waitFor(() => {
            expect(result.current.isMuted).toBe(true);
        });
    });
    describe("autoplay unlock", () => {
        // Minimal HTMLAudioElement stand-in used when we need to drive the Kokoro
        // path through jsdom. Only the handful of properties KokoroProvider calls
        // are implemented; the rest are no-ops that mirror real element behavior.
        class FakeAudio extends EventTarget {
            constructor() {
                super(...arguments);
                this.src = "";
                this.currentTime = 0;
                this.duration = NaN;
                this.paused = true;
                this.playbackRate = 1;
                this.volume = 1;
                this.muted = false;
                this.play = vi.fn(async () => {
                    this.paused = false;
                    Object.defineProperty(this, "duration", { value: 1, writable: true, configurable: true });
                    setTimeout(() => this.dispatchEvent(new Event("ended")), 0);
                });
                this.pause = vi.fn(() => { this.paused = true; });
                this.load = vi.fn(() => { this.currentTime = 0; });
                this.removeAttribute = vi.fn();
            }
        }
        let fakeAudio;
        let createObjectURLSpy;
        let revokeObjectURLSpy;
        beforeEach(() => {
            fakeAudio = new FakeAudio();
            vi.stubGlobal("Audio", vi.fn(() => fakeAudio));
            createObjectURLSpy = vi.fn(() => "blob:fake-url");
            revokeObjectURLSpy = vi.fn();
            globalThis.URL.createObjectURL = createObjectURLSpy;
            globalThis.URL.revokeObjectURL = revokeObjectURLSpy;
        });
        afterEach(() => {
            vi.unstubAllGlobals();
        });
        it("needsUnlock starts false", async () => {
            mockFetchCaps.mockResolvedValue({
                capabilities: [{ id: "kokoro-tts", status: "available" }],
                timestamp: new Date().toISOString(),
            });
            mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
            const { result } = renderHook(() => useTextToSpeech(defaultSettings));
            await waitFor(() => expect(result.current.backend).toBe("kokoro"));
            expect(result.current.needsUnlock).toBe(false);
        });
        it("speakParagraphs sets needsUnlock=true (not error) on NotAllowedError", async () => {
            mockFetchCaps.mockResolvedValue({
                capabilities: [{ id: "kokoro-tts", status: "available" }],
                timestamp: new Date().toISOString(),
            });
            mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
            const blob = new Blob(["audio"], { type: "audio/mp3" });
            mockSynthesizeTTS.mockResolvedValue(blob);
            fakeAudio.play = vi.fn().mockRejectedValue(Object.assign(new Error("not allowed by the user agent"), { name: "NotAllowedError" }));
            const { result } = renderHook(() => useTextToSpeech({
                ...defaultSettings,
                backendPreference: "kokoro",
            }));
            await waitFor(() => expect(result.current.backend).toBe("kokoro"));
            await act(async () => {
                await result.current.speakParagraphs(["paragraph one"]);
            });
            await waitFor(() => expect(result.current.needsUnlock).toBe(true));
            expect(result.current.error).toBeNull();
        });
        it("generic non-autoplay failure sets error, not needsUnlock", async () => {
            mockFetchCaps.mockResolvedValue({
                capabilities: [{ id: "kokoro-tts", status: "available" }],
                timestamp: new Date().toISOString(),
            });
            mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
            mockSynthesizeTTS.mockRejectedValue(new Error("Kokoro synthesis failed"));
            const { result } = renderHook(() => useTextToSpeech({
                ...defaultSettings,
                backendPreference: "kokoro",
            }));
            await waitFor(() => expect(result.current.backend).toBe("kokoro"));
            await act(async () => {
                try {
                    await result.current.speakParagraphs(["paragraph one"]);
                }
                catch {
                    // expected
                }
            });
            await waitFor(() => expect(result.current.error).toBe("Kokoro synthesis failed"));
            expect(result.current.needsUnlock).toBe(false);
        });
        it("unlockAudio() calls provider.unlock() and clears needsUnlock on success", async () => {
            mockFetchCaps.mockResolvedValue({
                capabilities: [{ id: "kokoro-tts", status: "available" }],
                timestamp: new Date().toISOString(),
            });
            mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
            const blob = new Blob(["audio"], { type: "audio/mp3" });
            mockSynthesizeTTS.mockResolvedValue(blob);
            fakeAudio.play = vi.fn().mockRejectedValue(Object.assign(new Error("not allowed"), { name: "NotAllowedError" }));
            const { result } = renderHook(() => useTextToSpeech({
                ...defaultSettings,
                backendPreference: "kokoro",
            }));
            await waitFor(() => expect(result.current.backend).toBe("kokoro"));
            await act(async () => { await result.current.speakParagraphs(["x"]); });
            await waitFor(() => expect(result.current.needsUnlock).toBe(true));
            // Reconfigure play to succeed so unlock() resolves true.
            fakeAudio.play = vi.fn(async () => { fakeAudio.paused = false; });
            let unlockResult;
            await act(async () => { unlockResult = await result.current.unlockAudio(); });
            expect(unlockResult).toBe(true);
            expect(result.current.needsUnlock).toBe(false);
            expect(result.current.browserAudioReady).toBe(true);
        });
        it("dismissNeedsUnlock clears the flag without touching the provider", async () => {
            mockFetchCaps.mockResolvedValue({
                capabilities: [{ id: "kokoro-tts", status: "available" }],
                timestamp: new Date().toISOString(),
            });
            mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
            const blob = new Blob(["audio"], { type: "audio/mp3" });
            mockSynthesizeTTS.mockResolvedValue(blob);
            fakeAudio.play = vi.fn().mockRejectedValue(Object.assign(new Error("not allowed"), { name: "NotAllowedError" }));
            const { result } = renderHook(() => useTextToSpeech({
                ...defaultSettings,
                backendPreference: "kokoro",
            }));
            await waitFor(() => expect(result.current.backend).toBe("kokoro"));
            await act(async () => { await result.current.speakParagraphs(["x"]); });
            await waitFor(() => expect(result.current.needsUnlock).toBe(true));
            act(() => result.current.dismissNeedsUnlock());
            expect(result.current.needsUnlock).toBe(false);
        });
        it("a user gesture does NOT silently play the audio element (no iOS session activation)", async () => {
            // Regression: a global pointerdown/keydown listener used to preemptively
            // "unlock" the Kokoro media element with a silent play() on ANY gesture.
            // On iOS that activates AVAudioSession and ducks other apps' audio even
            // when TTS is never used (the "web-console pauses my music on any tap"
            // bug). A gesture must flip only the no-audio readiness flag.
            mockFetchCaps.mockResolvedValue({
                capabilities: [{ id: "kokoro-tts", status: "available" }],
                timestamp: new Date().toISOString(),
            });
            mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
            fakeAudio.play = vi.fn(async () => { fakeAudio.paused = false; });
            const { result } = renderHook(() => useTextToSpeech({
                ...defaultSettings,
                backendPreference: "kokoro",
            }));
            await waitFor(() => expect(result.current.backend).toBe("kokoro"));
            act(() => { window.dispatchEvent(new Event("pointerdown")); });
            act(() => { window.dispatchEvent(new KeyboardEvent("keydown", { key: "a" })); });
            // No media-element playback was triggered by the gesture …
            expect(fakeAudio.play).not.toHaveBeenCalled();
            // … but the browser-TTS readiness flag still flips (no audio involved).
            expect(result.current.browserAudioReady).toBe(true);
        });
        it("pause controls cached Kokoro blob playback from the active provider", async () => {
            mockFetchCaps.mockResolvedValue({
                capabilities: [{ id: "kokoro-tts", status: "available" }],
                timestamp: new Date().toISOString(),
            });
            mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
            mockFetchCachedTTS.mockResolvedValue(new Blob(["audio"], { type: "audio/mp3" }));
            const { result } = renderHook(() => useTextToSpeech({
                ...defaultSettings,
                backendPreference: "kokoro",
            }));
            await waitFor(() => expect(result.current.backend).toBe("kokoro"));
            let playback;
            await act(async () => {
                playback = result.current.speakParagraphs(["cached paragraph"], {
                    eventId: "evt-cached",
                    version: "active",
                });
                await Promise.resolve();
            });
            act(() => result.current.pause());
            expect(fakeAudio.pause).toHaveBeenCalled();
            expect(result.current.isPaused).toBe(true);
            await act(async () => {
                fakeAudio.dispatchEvent(new Event("ended"));
                await playback;
            });
        });
        it("replays a fully-cached multi-paragraph message per chunk without synthesizing", async () => {
            mockFetchCaps.mockResolvedValue({
                capabilities: [{ id: "kokoro-tts", status: "available" }],
                timestamp: new Date().toISOString(),
            });
            mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
            // Distinct cached blob per chunk index — a full hit.
            mockFetchCachedTTS.mockImplementation((_e, _v, _s, _ver, _sig, chunkIndex = 0) => Promise.resolve(new Blob([`chunk-${chunkIndex}`], { type: "audio/mp3" })));
            const { result } = renderHook(() => useTextToSpeech({ ...defaultSettings, backendPreference: "kokoro" }));
            await waitFor(() => expect(result.current.backend).toBe("kokoro"));
            let playback;
            await act(async () => {
                playback = result.current.speakParagraphs(["para one", "para two"], { eventId: "evt-multi", version: "active" });
                await Promise.resolve();
            });
            // Two tracks: dispatch 'ended' for each.
            await act(async () => { fakeAudio.dispatchEvent(new Event("ended")); await Promise.resolve(); });
            await act(async () => { fakeAudio.dispatchEvent(new Event("ended")); await playback; });
            // Both chunks were fetched by index; nothing was synthesized.
            const indices = mockFetchCachedTTS.mock.calls.map((c) => c[5]);
            expect(indices).toEqual([0, 1]);
            expect(mockSynthesizeTTS).not.toHaveBeenCalled();
        });
        it("falls through to synthesis when any chunk is a cache miss", async () => {
            mockFetchCaps.mockResolvedValue({
                capabilities: [{ id: "kokoro-tts", status: "available" }],
                timestamp: new Date().toISOString(),
            });
            mockGetVoices.mockResolvedValue([{ id: "af_heart", name: "af_heart" }]);
            // Chunk 0 hits, chunk 1 misses → not fully cached → synthesize path.
            mockFetchCachedTTS.mockImplementation((_e, _v, _s, _ver, _sig, chunkIndex = 0) => Promise.resolve(chunkIndex === 0 ? new Blob(["c0"], { type: "audio/mp3" }) : null));
            mockSynthesizeTTS.mockResolvedValue(new Blob(["synth"], { type: "audio/mp3" }));
            const { result } = renderHook(() => useTextToSpeech({ ...defaultSettings, backendPreference: "kokoro" }));
            await waitFor(() => expect(result.current.backend).toBe("kokoro"));
            let playback;
            await act(async () => {
                playback = result.current.speakParagraphs(["para one", "para two"], { eventId: "evt-partial", version: "active" });
                await Promise.resolve();
            });
            await act(async () => { fakeAudio.dispatchEvent(new Event("ended")); await Promise.resolve(); });
            await act(async () => { fakeAudio.dispatchEvent(new Event("ended")); await playback; });
            // Fell through to synthesis (which repopulates the cache per chunk).
            expect(mockSynthesizeTTS).toHaveBeenCalled();
        });
    });
});
