import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useScenarioVoiceCore } from "./useVoiceCore";
import { voiceCoreServices } from "../voiceCoreServices";
import { computeFinalTimeout } from "./voice/types";
import { decideAutoStop } from "./voice/autoStopDecision";
import type { ServerVadStateSnapshot } from "@vrooli/audio-capture-browser";

describe("audio-tools shared voice-core adapter", () => {
	it("runs the shared orchestrator with host-owned services", () => {
		const onTranscript = vi.fn();
		const { result } = renderHook(() =>
			useScenarioVoiceCore({
				voiceEnabled: false,
				voiceLanguage: "en-US",
				vadSilenceTimeoutMs: 800,
				persistentMode: false,
				wakeWordEnabled: false,
				segmentSilenceMs: 500,
				onTranscript,
			}),
		);

		expect(result.current.voiceState).toBe("idle");
		expect(result.current.isRecording).toBe(false);
		expect(result.current.isListening).toBe(false);
		expect(result.current.isTranscribing).toBe(false);
		expect(typeof voiceCoreServices.getVoiceStreamConfig).toBe("function");
		expect(typeof voiceCoreServices.transcribeAudio).toBe("function");
		expect(typeof voiceCoreServices.PcmVoiceStreamProvider).toBe("function");
		expect(computeFinalTimeout(2_000)).toBe(10_000);
		expect(computeFinalTimeout(20_000)).toBe(40_000);
		const noServerTick: ServerVadStateSnapshot = {
			voiced: false,
			silenceElapsedMs: 0,
			silenceTimeoutMs: 0,
			receivedAt: 0,
			tickSeq: 0,
			silenceTimedOut: false,
		};
		expect(decideAutoStop({ serverVad: noServerTick, clientVadResult: "stop", nowPerf: 0, staleTickMs: 250 })).toEqual({
			kind: "stop",
			source: "client-fallback",
		});
	});
});
