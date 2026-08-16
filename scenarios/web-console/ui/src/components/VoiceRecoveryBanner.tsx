import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";

export interface VoiceRecoveryBannerProps {
  /** Latest voice-input error, or null when there is nothing wrong. */
  readonly error?: string | null;
  /** A transcription is in flight and can still be abandoned. */
  readonly isTranscribing?: boolean;
  readonly onCancel?: () => void;
  /** The microphone lease outlived its turn and is still held. */
  readonly staleLiveMic?: boolean;
  readonly onReleaseMic?: () => void;
  /** Text-to-speech is talking and can be silenced. */
  readonly isTtsSpeaking?: boolean;
  readonly onTtsStop?: () => void;
  readonly canExportDiagnostic?: boolean;
  readonly onExportDiagnostic?: () => string | null;
}

/**
 * App-level surface for everything that can go wrong with voice input.
 *
 * These affordances used to render inside the microphone button's own wrapper,
 * in flow, so the control's footprint changed with its state and the layout
 * around it moved while the operator was speaking. The button is a button; a
 * recovery action is app chrome. They live in different places for that reason,
 * and `VoiceInputButton`'s experience contract now declares a fixed footprint
 * so the split cannot quietly reverse.
 */
export default function VoiceRecoveryBanner({
  error = null,
  isTranscribing = false,
  onCancel,
  staleLiveMic = false,
  onReleaseMic,
  isTtsSpeaking = false,
  onTtsStop,
  canExportDiagnostic = false,
  onExportDiagnostic,
}: VoiceRecoveryBannerProps) {
  const { t } = useTranslation();

  const actions = [
    isTranscribing && onCancel
      ? { key: "cancel", label: t(strings.voiceRecovery.cancel), run: onCancel }
      : null,
    staleLiveMic && onReleaseMic
      ? { key: "release-mic", label: t(strings.voiceRecovery.releaseMic), run: onReleaseMic }
      : null,
    isTtsSpeaking && onTtsStop
      ? { key: "stop-speech", label: t(strings.voiceRecovery.stopSpeech), run: onTtsStop }
      : null,
    canExportDiagnostic && onExportDiagnostic
      ? {
        key: "export-diagnostic",
        label: t(strings.voiceRecovery.exportDiagnostic),
        run: () => { onExportDiagnostic(); },
      }
      : null,
  ].filter((action): action is { key: string; label: string; run: () => void } => action !== null);

  if (!error && actions.length === 0) return null;

  return (
    <div
      data-testid="voice-recovery-banner"
      role={error ? "alert" : "status"}
      aria-label={t(strings.voiceRecovery.regionLabel)}
      className="wc-stable-theme flex flex-wrap items-center gap-2 border-b border-wc-default bg-wc-surface-raised py-1.5 ps-[max(0.75rem,var(--wc-safe-left,0px))] pe-[max(0.75rem,var(--wc-safe-right,0px))] text-xs"
    >
      {error && (
        <span data-testid="voice-recovery-error" className="min-w-0 flex-1 break-words text-wc-error-text">
          {error}
        </span>
      )}
      {actions.map((action) => (
        <button
          key={action.key}
          type="button"
          data-testid={`voice-recovery-${action.key}`}
          onClick={action.run}
          className="shrink-0 rounded border border-wc-default px-2 py-0.5 text-wc-text-muted transition active:bg-wc-surface-base"
        >
          {action.label}
        </button>
      ))}
    </div>
  );
}
