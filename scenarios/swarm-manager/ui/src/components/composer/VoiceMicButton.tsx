import { VoiceInputButton } from "../../audio-integration/VoiceInputButton";
import type { StartRecordingOpts, VoiceActivitySnapshot } from "../../audio-integration";

export interface VoiceMicButtonProps {
  supported: boolean;
  isPreparing: boolean;
  isRecording: boolean;
  isListening?: boolean;
  isPassive?: boolean;
  isTranscribing: boolean;
  error: string | null;
  audioLevel?: number;
  voiceActivity?: VoiceActivitySnapshot;
  partialTranscript?: string;
  serverVad?: unknown;
  onStart: (opts?: StartRecordingOpts) => void;
  onStop: () => void;
  onCancel?: () => void;
  onExitPassive?: () => void;
  className?: string;
  buttonClassName?: string;
  testId?: string;
}

/** Host prop adapter; interaction and presentation are owned by RCL. */
export default function VoiceMicButton({
  supported,
  isPreparing,
  isRecording,
  isListening,
  isPassive,
  isTranscribing,
  error,
  audioLevel = 0,
  voiceActivity,
  onStart,
  onStop,
  onPrepare,
  onCancel,
  onExitPassive,
  className,
  buttonClassName,
  testId,
}: VoiceMicButtonProps & { onPrepare?: () => void }) {
  const state = !supported ? "unavailable" : isTranscribing ? "transcribing" : isPreparing ? "preparing" : isPassive ? "recovering" : isRecording || isListening ? "recording" : error ? "error" : "idle";
  return (
    <div className={["shrink-0", className].filter(Boolean).join(" ")}>
      <VoiceInputButton
        state={state}
        mode={isListening ? "always-on" : "timeout"}
        level={audioLevel}
        timeoutProgress={voiceActivity?.autoStopProgress ?? 0}
        onStart={() => onStart?.()}
        onStop={onStop}
        onPrepare={onPrepare}
        onExitPassive={onExitPassive}
        onCancel={onCancel}
        className={buttonClassName}
        data-testid={testId}
      />
    </div>
  );
}
