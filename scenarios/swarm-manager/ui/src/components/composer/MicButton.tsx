// MicButton — composer-side host that wires audio-tools' voice pipeline
// (useVoiceCore) to swarm-manager's themed VoiceMicButton. VAD timing is
// pulled from useVoiceConfigStore so the ring countdown matches audio-
// tools' actual server-side cut (`stt_stream_config.vad_silence_ms`).
//
// Used in:
//   - SessionConversation.tsx (Session Details composer)
//   - quick-capture-input.tsx (Quick Capture dialog)
//   - MessageComposer.tsx (generic composer)

import { useCallback, useEffect } from "react";

import { useVoiceCore, useAudioToolsUnavailableReason, useVoiceConfigStore, useServerVadStateStore } from "../../audio-integration";
import VoiceMicButton from "./VoiceMicButton";

interface MicButtonProps {
  onTranscript: (text: string) => void;
  onPartialTranscript?: (text: string) => void;
  disabled?: boolean;
  testId?: string;
}

export function MicButton({ onTranscript, onPartialTranscript, disabled, testId }: MicButtonProps) {
  const unavailableReason = useAudioToolsUnavailableReason();
  const vadSilenceTimeoutMs = useVoiceConfigStore((s) => s.vadSilenceTimeoutMs);
  const segmentSilenceMs = useVoiceConfigStore((s) => s.segmentSilenceMs);
  const persistentMode = useVoiceConfigStore((s) => s.persistentMode);
  const wakeWordEnabled = useVoiceConfigStore((s) => s.wakeWordEnabled);
  const wakeWordThreshold = useVoiceConfigStore((s) => s.wakeWordThreshold);
  const serverVad = useServerVadStateStore((s) => s);

  const voice = useVoiceCore({
    voiceEnabled: !unavailableReason && !disabled,
    voiceLanguage: "en",
    vadSilenceTimeoutMs,
    persistentMode,
    wakeWordEnabled,
    wakeWordThreshold,
    segmentSilenceMs,
    lowLatencyVoice: false,
    onTranscript: (text) => {
      const trimmed = text.trim();
      if (trimmed) onTranscript(trimmed);
    },
  });

  useEffect(() => {
    onPartialTranscript?.(voice.partialTranscript);
  }, [onPartialTranscript, voice.partialTranscript]);

  const handleStart = useCallback(() => {
    void voice.startRecording();
  }, [voice]);

  const handleStop = useCallback(() => {
    voice.stopRecording();
  }, [voice]);

  const supported = !unavailableReason;
  const errorMessage = unavailableReason
    ? `Voice input unavailable (${unavailableReason})`
    : voice.error;

  return (
    <VoiceMicButton
      supported={supported}
      isPreparing={voice.isPreparing}
      isRecording={voice.isRecording}
      isListening={voice.isListening}
      isPassive={voice.isPassive}
      isTranscribing={voice.isTranscribing}
      error={errorMessage}
      audioLevel={voice.audioLevel}
      voiceActivity={voice.voiceActivity}
      serverVad={serverVad}
      partialTranscript={voice.partialTranscript}
      onStart={handleStart}
      onStop={handleStop}
      onCancel={voice.cancelTranscription}
      onExitPassive={voice.exitPassiveMode}
      buttonClassName="mb-0.5"
      testId={testId}
    />
  );
}
