import { VoiceInputButton, type ButtonSize } from "@vrooli/react-component-library/VoiceInputButton/4.3.0";
import type { StartRecordingOpts, VoiceActivitySnapshot } from "../audio-integration";

export interface VoiceMicButtonProps {
  supported: boolean;
  isPreparing: boolean;
  isRecording: boolean;
  persistentMode?: boolean;
  isListening?: boolean;
  isPassive?: boolean;
  isTranscribing: boolean;
  error: string | null;
  audioLevel?: number;
  voiceActivity?: VoiceActivitySnapshot;
  backend?: string;
  size?: ButtonSize;
  onStart: (opts?: StartRecordingOpts) => void;
  onStop: () => void;
  onExitPassive?: () => void;
  onPrepare?: () => void;
  className?: string;
  buttonClassName?: string;
  iconClassName?: string;
  testId?: string;
}

/**
 * Host prop adapter: maps web-console's voice state onto the library button's
 * lifecycle states. Interaction and presentation are owned by RCL.
 *
 * This renders the button and nothing else, on purpose. Error text and recovery
 * actions (cancel, release microphone, stop speech, export diagnostic) are app
 * chrome and live in `VoiceRecoveryBanner`; rendering them here changed the
 * control's footprint with its state and shifted the layout mid-sentence.
 * `voice-mic-button-footprint.test.tsx` holds that line.
 */
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
  backend,
  size = "sm",
  onStart,
  onStop,
  onExitPassive,
  onPrepare,
  className,
  buttonClassName,
  iconClassName,
  testId,
}: VoiceMicButtonProps) {
  const state = !supported ? "unavailable"
    : isTranscribing ? "transcribing"
      : isPreparing ? "preparing"
        : isPassive ? "recovering"
          : isRecording || isListening ? "recording"
            : error ? "error"
              : "idle";
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
    </div>
  );
}
