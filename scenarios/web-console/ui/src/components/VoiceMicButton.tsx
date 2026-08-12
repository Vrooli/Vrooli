import { VoiceInputButton, type ButtonSize } from "../audio-integration/VoiceInputButton";
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
    <div className={className} data-voice-backend={backend}>
      <VoiceInputButton
        state={state}
        mode={persistentMode ? "always-on" : "timeout"}
        level={audioLevel}
        timeoutProgress={voiceActivity?.autoStopProgress ?? 0}
        size={size}
        iconClassName={iconClassName}
        onExitPassive={onExitPassive}
        className={buttonClassName}
        onStart={() => onStart?.()}
        onStop={onStop}
        onPrepare={onPrepare}
        data-testid={testId}
      />
      {error && <div role="alert" className="mt-1 text-center text-xs text-wc-error-text">{error}</div>}
      {(state === "transcribing" && onCancel) || staleLiveMic || (isTtsSpeaking && onTtsStop) || (canExportDiagnostic && onExportDiagnostic) ? (
        <div role="group" aria-label="Voice recovery actions" className="mt-1 flex flex-wrap justify-center gap-1 text-[10px]">
          {state === "transcribing" && onCancel && <button type="button" className="rounded border border-wc-default px-1.5 py-0.5 text-wc-text-muted" onClick={onCancel}>Cancel</button>}
          {staleLiveMic && onReleaseMic && <button type="button" className="rounded border border-wc-accent px-1.5 py-0.5 text-wc-accent" onClick={onReleaseMic}>Release mic</button>}
          {isTtsSpeaking && onTtsStop && <button type="button" className="rounded border border-wc-accent px-1.5 py-0.5 text-wc-accent" onClick={onTtsStop}>Stop speech</button>}
          {canExportDiagnostic && onExportDiagnostic && <button type="button" className="rounded border border-wc-default px-1.5 py-0.5 text-wc-text-muted" onClick={() => { onExportDiagnostic(); }}>Export diagnostic</button>}
        </div>
      ) : null}
    </div>
  );
}
