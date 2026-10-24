import { useEffect, useRef } from "react";
import { useScenarioVoiceInput } from "../audio-integration";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useAgentVoice, selectAgentVoice } from "./useAgentVoice";

/** One microphone surface; native transcripts never enter the text-input path. */
export function useRoutedVoiceInput(onTranscript: (text: string) => void) {
  const sessionId = useWorkspaceStore((s) => s.activePane);
  const mode = useWorkspaceStore((s) => s.agentVoiceMode);
  const persistentMode = useWorkspaceStore((s) => s.persistentMode);
  const voiceEnabled = useWorkspaceStore((s) => s.voiceEnabled);
  // The provider may already be listening through persistent/wake-word mode
  // on mount. Pane changes invalidate this binding until a fresh mic gesture.
  const dictationTarget = useRef<string | null>(sessionId);
  const dictation = useScenarioVoiceInput((text) => {
    if (dictationTarget.current === useWorkspaceStore.getState().activePane) onTranscript(text);
  });
  const native = useAgentVoice(sessionId);
  const dictationStop = useRef(dictation.stopRecording); dictationStop.current = dictation.stopRecording;
  const dictationCancel = useRef(dictation.cancelTranscription); dictationCancel.current = dictation.cancelTranscription;
  const exitPassive = useRef(dictation.exitPassiveMode); exitPassive.current = dictation.exitPassiveMode;
  useEffect(() => { if (native.selected) { exitPassive.current(); dictationStop.current(); } }, [native.selected]);
  useEffect(() => () => { dictationTarget.current = null; exitPassive.current(); dictationCancel.current(); }, [sessionId]);
  const useNative = mode !== "dictation" && (native.selected || native.voiceState !== "idle" || native.error !== null);
  const startRecordingWithMode = async (chosen: "auto" | "agent" | "dictation", options?: Parameters<typeof dictation.startRecording>[0]) => {
    if (selectAgentVoice(chosen, native.capability)) {
      dictation.exitPassiveMode();
      dictation.stopRecording();
      await native.start();
    } else {
      native.stop();
      dictationTarget.current = sessionId;
      dictation.prepareRecording();
      await dictation.startRecording(options);
    }
  };
  const routed = useNative ? {
    supported: voiceEnabled && native.capability.available,
    capabilityReason: native.capability.reason,
    backend: "agent" as const,
    voiceState: native.voiceState,
    isPreparing: native.voiceState === "preparing",
    isRecording: false,
    isListening: native.voiceState === "listening",
    isActive: native.voiceState !== "idle",
    isTranscribing: false,
    isPassive: false,
    audioLevel: native.audioLevel,
    error: native.error,
    partialTranscript: native.partialTranscript,
    prepareRecording: async () => {},
    startRecording: async () => { dictation.exitPassiveMode(); dictation.stopRecording(); await native.start(); },
    stopRecording: native.stop,
    cancelTranscription: native.stop,
    releaseMicrophone: native.stop,
  } : {
    prepareRecording: () => {
      dictationTarget.current = sessionId;
      dictation.prepareRecording();
    },
    startRecording: async (options?: Parameters<typeof dictation.startRecording>[0]) => {
      dictationTarget.current = sessionId;
      await dictation.startRecording(options);
    },
  };
  return {
    ...dictation,
    ...routed,
    partialTranscriptRole: useNative ? native.partialTranscriptRole : null,
    needsRouteSelection: mode !== "dictation" && (useNative ? !native.capability.available : !dictation.supported),
    native,
    startRecordingWithMode,
    persistentMode: useNative || persistentMode,
  };
}
