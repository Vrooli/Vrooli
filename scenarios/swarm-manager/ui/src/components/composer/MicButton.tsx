// MicButton — swarm-manager-themed mic toggle for the message composer.
// Wraps useVoiceCore from audio-integration; appends transcripts via the
// onTranscript prop so the caller appends to its own draft state.
//
// Used in:
//   - SessionConversation.tsx (Session Details composer)
//   - quick-capture-input.tsx (Quick Capture dialog)

import { useCallback } from "react";
import { Mic, MicOff } from "lucide-react";
import { cn } from "../../lib/utils";
import { useVoiceCore, useAudioToolsUnavailableReason } from "../../audio-integration";

interface MicButtonProps {
  onTranscript: (text: string) => void;
  disabled?: boolean;
  testId?: string;
}

export function MicButton({ onTranscript, disabled, testId }: MicButtonProps) {
  const unavailableReason = useAudioToolsUnavailableReason();

  const voice = useVoiceCore({
    voiceEnabled: !unavailableReason,
    voiceLanguage: "en",
    vadSilenceTimeoutMs: 1500,
    persistentMode: false,
    wakeWordEnabled: false,
    segmentSilenceMs: 800,
    lowLatencyVoice: false,
    onTranscript: (text) => {
      const trimmed = text.trim();
      if (trimmed) onTranscript(trimmed);
    },
  });

  const handleClick = useCallback(() => {
    if (voice.isActive) {
      voice.stopRecording();
    } else {
      void voice.startRecording();
    }
  }, [voice]);

  const isUnavailable = Boolean(unavailableReason);
  const buttonDisabled = disabled || isUnavailable;

  const Icon = isUnavailable ? MicOff : Mic;
  const title = isUnavailable
    ? `Voice input unavailable (${unavailableReason})`
    : voice.isActive
      ? "Stop recording"
      : "Start voice input";

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={buttonDisabled}
      className={cn(
        "mb-0.5 shrink-0 rounded p-1 transition-colors disabled:opacity-50",
        voice.isActive
          ? "animate-pulse text-cyan-400 hover:bg-cyan-500/10"
          : "text-slate-500 hover:bg-slate-700 hover:text-slate-300",
        isUnavailable && "text-red-400",
      )}
      title={title}
      aria-pressed={voice.isActive}
      data-testid={testId}
    >
      <Icon className="h-4 w-4" />
      <span className="sr-only">{title}</span>
    </button>
  );
}
