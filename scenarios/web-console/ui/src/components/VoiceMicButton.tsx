import { VoiceInputButton, type ButtonSize } from "../audio-integration/SharedVoiceInputButton";
import type { StartRecordingOpts, VoiceActivitySnapshot } from "../audio-integration";

export interface VoiceMicButtonProps {
  supported: boolean;
  isPreparing: boolean;
  isRecording: boolean;
  persistentMode?: boolean;
  isListening?: boolean;
  isPassive?: boolean;
  isTranscribing: boolean;
  staleLiveMic?: boolean;
  error: string | null;
  audioLevel?: number;
  voiceActivity?: VoiceActivitySnapshot;
  partialTranscript?: string;
  backend?: string;
  isTtsSpeaking?: boolean;
  size?: ButtonSize;
  onStart: (opts?: StartRecordingOpts) => void;
  onStop: () => void;
  onCancel?: () => void;
  onExitPassive?: () => void;
  onReleaseMic?: () => void;
  onTtsStop?: () => void;
  onPrepare?: () => void;
  canExportDiagnostic?: boolean;
  onExportDiagnostic?: () => string | null;
  className?: string;
  buttonClassName?: string;
  iconClassName?: string;
  testId?: string;
}

/** Host prop adapter; interaction and presentation are owned by RCL. */
export default function VoiceMicButton({
  supported,
  isPreparing,
  isRecording,
  persistentMode = false,
  isListening,
  isPassive,
  isTranscribing,
  error,
  audioLevel = 0,
  voiceActivity,
  partialTranscript,
  staleLiveMic,
  backend,
  isTtsSpeaking,
  size = "sm",
  onStart,
  onStop,
  onCancel,
  onExitPassive,
  onReleaseMic,
  onTtsStop,
  onPrepare,
  className,
  canExportDiagnostic,
  onExportDiagnostic,
  buttonClassName,
  iconClassName,
  testId,
}: VoiceMicButtonProps) {
  const state = !supported ? "unavailable" : isTranscribing ? "transcribing" : isPreparing ? "preparing" : isPassive ? "recovering" : isRecording || isListening ? "recording" : error ? "error" : "idle";
  return (
    <VoiceInputButton
      state={state}
      mode={persistentMode ? "always-on" : "timeout"}
      level={audioLevel}
      timeoutProgress={voiceActivity?.autoStopProgress ?? 0}
      error={error ?? undefined}
      partialTranscript={partialTranscript}
      staleLiveMic={staleLiveMic}
      backend={backend}
      isTtsSpeaking={isTtsSpeaking}
      size={size}
      canExportDiagnostic={canExportDiagnostic}
      onCancel={onCancel}
      onExitPassive={onExitPassive}
      onReleaseMic={onReleaseMic}
      onTtsStop={onTtsStop}
      onExportDiagnostic={onExportDiagnostic}
      wrapperClassName={className}
      iconClassName={iconClassName}
      className={buttonClassName}
      onStart={() => onStart?.()}
      onStop={onStop}
      onPrepare={onPrepare}
      data-testid={testId}
    />
  );
}
