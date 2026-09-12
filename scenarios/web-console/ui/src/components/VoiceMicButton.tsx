import { useEffect, useRef, type CSSProperties } from "react";
import { VoiceInputButton, type ButtonSize } from "@vrooli/react-component-library/VoiceInputButton/4";
import type { StartRecordingOpts, VoiceActivitySnapshot } from "../audio-integration";

const WAVEFORM_POINTS = 48;

function waveformPath(samples: number[]) {
  const points = samples.map((sample, index) => ({
    x: (index / (WAVEFORM_POINTS - 1)) * 100,
    // Treat the bottom of the control as the quiet baseline. Louder speech
    // lifts the smooth trace toward the top instead of oscillating around the
    // button's vertical center.
    y: 94 - sample * 86,
  }));
  const first = points[0];
  if (!first) return "none";
  let path = `M ${first.x.toFixed(2)} ${first.y.toFixed(2)}`;
  for (let index = 1; index < points.length; index += 1) {
    const previous = points[index - 1]!;
    const current = points[index]!;
    const midpointX = (previous.x + current.x) / 2;
    const midpointY = (previous.y + current.y) / 2;
    path += ` Q ${previous.x.toFixed(2)} ${previous.y.toFixed(2)} ${midpointX.toFixed(2)} ${midpointY.toFixed(2)}`;
  }
  const last = points[points.length - 1]!;
  path += ` Q ${last.x.toFixed(2)} ${last.y.toFixed(2)} ${last.x.toFixed(2)} ${last.y.toFixed(2)}`;
  return path;
}

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
  capabilityReason?: string;
  operatorCommand?: string;
  size?: ButtonSize;
  onStart: (opts?: StartRecordingOpts) => void;
  onStop: () => void;
  onExitPassive?: () => void;
  onPrepare?: () => void;
  className?: string;
  buttonClassName?: string;
  iconClassName?: string;
  style?: CSSProperties;
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
  capabilityReason,
  operatorCommand,
  size = "sm",
  onStart,
  onStop,
  onExitPassive,
  onPrepare,
  className,
  buttonClassName,
  iconClassName,
  style,
  testId,
}: VoiceMicButtonProps) {
  const state = !supported ? "unavailable"
    : isTranscribing ? "transcribing"
      : isPreparing ? "preparing"
        : isPassive ? "recovering"
          : isRecording || isListening ? "recording"
            : error ? "error"
              : "idle";
  const active = state === "recording" || state === "recovering";
  const hostRef = useRef<HTMLDivElement>(null);
  const waveformRef = useRef<number[]>([]);
  const latestLevelRef = useRef(0);
  latestLevelRef.current = Math.max(0, Math.min(1, audioLevel));

  useEffect(() => {
    const host = hostRef.current;
    const path = host?.querySelector<SVGPathElement>("[data-voice-waveform-path]");
    if (!path) return;

    if (!active) {
      waveformRef.current = [];
      path.setAttribute("d", "M 0 94 Q 50 94 100 94");
      return;
    }

    const samples = waveformRef.current;
    while (samples.length < WAVEFORM_POINTS) samples.push(0.04);
    let phase = 0;
    let frame = 0;

    const draw = () => {
      const level = latestLevelRef.current;
      phase += 0.13;
      const envelope = 0.04 + level * 0.86;
      const previous = samples[samples.length - 1] ?? 0.04;
      const signal = envelope * (0.72 + 0.28 * Math.sin(phase));
      samples.push(previous * 0.76 + signal * 0.24);
      if (samples.length > WAVEFORM_POINTS) samples.shift();
      path.setAttribute("d", waveformPath(samples));
      frame = window.requestAnimationFrame(draw);
    };

    draw();
    return () => window.cancelAnimationFrame(frame);
  }, [active]);

  return (
    <div
      ref={hostRef}
      className={["voice-waveform-host", className].filter(Boolean).join(" ")}
      data-voice-waveform-active={active ? "true" : "false"}
      data-voice-backend={backend}
      title={!supported ? [capabilityReason, operatorCommand && `Fix: ${operatorCommand}`].filter(Boolean).join(" ") : undefined}
    >
      <VoiceInputButton
        state={state}
        mode={persistentMode ? "always-on" : "timeout"}
        level={audioLevel}
        timeoutProgress={voiceActivity?.autoStopProgress ?? 0}
        size={size}
        surface="soft"
        shape="rounded"
        iconClassName={iconClassName}
        onExitPassive={onExitPassive}
        className={["voice-waveform-control", buttonClassName].filter(Boolean).join(" ")}
        style={{
          minInlineSize: 0,
          minBlockSize: 0,
          ...style,
        }}
        onStart={() => onStart?.()}
        onStop={onStop}
        onPrepare={onPrepare}
        data-testid={testId}
        aria-label={!supported ? ["Voice input unavailable", capabilityReason, operatorCommand && `Fix: ${operatorCommand}`].filter(Boolean).join(" — ") : undefined}
      />
      <svg
        data-voice-waveform
        aria-hidden="true"
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
      >
        <path
          data-voice-waveform-path
          d="M 0 94 Q 50 94 100 94"
          fill="none"
          stroke="currentColor"
          strokeWidth="3"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </div>
  );
}
