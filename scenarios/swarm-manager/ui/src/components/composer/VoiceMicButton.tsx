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
    <div className={className}>
      <VoiceInputButton
        state={state}
        mode={isListening ? "always-on" : "timeout"}
        level={audioLevel}
        timeoutProgress={voiceActivity?.autoStopProgress ?? 0}
        onStart={() => onStart?.()}
        onStop={onStop}
        onPrepare={onPrepare}
        onExitPassive={onExitPassive}
        onPointerCancel={() => onCancel?.()}
        className={buttonClassName}
        data-testid={testId}
      />
      {error && <div role="alert" className="mt-1 text-center text-xs text-app-danger">{error}</div>}
      {state === "transcribing" && onCancel && (
        <button type="button" className="mt-1 rounded border border-app-border px-1.5 py-0.5 text-xs text-app-muted-foreground" onClick={onCancel}>
          Cancel
        </button>
      )}
    </div>
  );
}
