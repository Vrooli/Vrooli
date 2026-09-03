/**
 * @libraryId react-component-library:VoiceInputButton
 * @displayName Voice Input Button
 * @version 4.3.1
 * @tags ["voice","microphone","input","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useCallback, useRef, type ButtonHTMLAttributes, type PointerEvent } from "react";
import { IconButton } from "@vrooli/react-component-library/IconButton/3.1.5";
import type {
  ControlDensity,
  ControlSize,
} from "@vrooli/react-component-library/ControlBase/1.1.0";
import { VoiceInputButtonGlyph as Glyph, type VoiceInputGlyphKind } from "../../../../support/VoiceInputButtonGlyph/versions/4.3.1/VoiceInputButtonGlyph";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
export const voiceInputButtonStyles = `
[data-rcl-voice-input] { --rcl-voice-accent: var(--color-foreground); --rcl-voice-surface: var(--color-surface); --rcl-voice-border: var(--color-border); position: relative; background: var(--rcl-voice-surface); border-color: var(--rcl-voice-border); color: var(--rcl-voice-accent); }
[data-rcl-voice-input][data-state="preparing"] { --rcl-voice-accent: var(--color-warning); --rcl-voice-surface: color-mix(in srgb, var(--color-warning) 10%, var(--color-surface)); --rcl-voice-border: color-mix(in srgb, var(--color-warning) 50%, var(--color-border)); }
[data-rcl-voice-input][data-state="recording"], [data-rcl-voice-input][data-state="recovering"] { --rcl-voice-accent: var(--color-info); --rcl-voice-surface: color-mix(in srgb, var(--color-info) 16%, var(--color-surface)); --rcl-voice-border: var(--color-info); }
[data-rcl-voice-input][data-state="recording"][data-mode="timeout"] { --rcl-voice-accent: var(--color-danger); --rcl-voice-surface: color-mix(in srgb, var(--color-danger) 14%, var(--color-surface)); --rcl-voice-border: var(--color-danger); }
[data-rcl-voice-input][data-state="transcribing"] { --rcl-voice-accent: var(--color-primary); --rcl-voice-surface: color-mix(in srgb, var(--color-primary) 14%, var(--color-surface)); --rcl-voice-border: var(--color-primary); }
[data-rcl-voice-input][data-state="unavailable"], [data-rcl-voice-input][data-state="error"] { --rcl-voice-accent: var(--color-warning); --rcl-voice-surface: color-mix(in srgb, var(--color-warning) 10%, var(--color-surface)); --rcl-voice-border: var(--color-warning); }
[data-rcl-voice-glyph] { inline-size: var(--control-icon-size); block-size: var(--control-icon-size); flex: 0 0 auto; }
[data-rcl-voice-input][data-state="preparing"] [data-rcl-voice-glyph], [data-rcl-voice-input][data-state="recovering"] [data-rcl-voice-glyph] { animation: rcl-voice-pulse var(--dur-slow) var(--ease-standard) infinite; }
[data-rcl-voice-input][data-state="transcribing"] [data-rcl-voice-glyph][data-rcl-voice-glyph-role="primary"] { animation: rcl-voice-spin var(--dur-moderate) linear infinite; }
[data-rcl-voice-glyph][data-rcl-voice-glyph-role="cancel"] { display: none; } [data-rcl-voice-input][data-cancellable="true"] { cursor: pointer; }
[data-rcl-voice-clip] { position: absolute; inset: 0; z-index: 0; border-radius: inherit; overflow: hidden; pointer-events: none; }
[data-rcl-voice-level] { position: absolute; inset-inline: 0; inset-block-end: 0; z-index: 0; block-size: 0; border-radius: inherit; background: color-mix(in srgb, var(--rcl-voice-accent) 58%, transparent); pointer-events: none; transition: block-size var(--dur-quick) var(--ease-standard); } [data-rcl-voice-level][data-mode="always-on"] { background: color-mix(in srgb, var(--color-primary) 60%, transparent); } [data-rcl-voice-level][data-mode="timeout"] { background: color-mix(in srgb, var(--color-warning) 70%, transparent); }
[data-rcl-voice-timeout-ring] { position: absolute; inset: 0; z-index: 1; inline-size: calc(100% - (var(--space-2xs) * 2)); block-size: calc(100% - (var(--space-2xs) * 2)); margin: auto; transform: rotate(-90deg); pointer-events: none; }
[data-rcl-voice-timeout-ring] circle { opacity: .82; } @keyframes rcl-voice-pulse { 50% { opacity: .58; transform: scale(.92); } } @keyframes rcl-voice-spin { to { transform: rotate(360deg); } }
`;
export type VoiceInputButtonState =
  | "idle"
  | "preparing"
  | "recording"
  | "recovering"
  | "transcribing"
  | "unavailable"
  | "error";
export type VoiceInputButtonMode = "always-on" | "timeout";
export type ButtonSize = ControlSize;
export interface VoiceInputButtonProps
  extends Omit<
    ButtonHTMLAttributes<HTMLButtonElement>,
    "children" | "onPointerDown" | "onPointerUp" | "size"
  > {
  readonly state?: VoiceInputButtonState;
  readonly mode?: VoiceInputButtonMode;
  readonly size?: ControlSize;
  readonly density?: ControlDensity;
  readonly level?: number;
  readonly timeoutProgress?: number;
  readonly onStart?: () => void;
  readonly onStop?: () => void;
  readonly onPrepare?: () => void;
  readonly onExitPassive?: () => void;
  readonly onCancel?: () => void;
  readonly iconClassName?: string;
}

const LONG_PRESS_MS = 300;
const RING_CIRCUMFERENCE = 2 * Math.PI * 18;
const labels: Record<VoiceInputButtonState, string> = {
  idle: "Start voice input",
  preparing: "Preparing microphone",
  recording: "Stop voice input",
  recovering: "Listening for voice input",
  transcribing: "Transcribing voice input",
  unavailable: "Voice input unavailable",
  error: "Voice input error",
};

export function VoiceInputButton({
  state = "idle",
  mode = "always-on",
  size = "sm",
  density = "comfortable",
  level = 0,
  timeoutProgress = 0,
  onStart,
  onStop,
  onPrepare,
  onExitPassive,
  onCancel,
  iconClassName,
  disabled,
  onPointerCancel,
  className,
  ...props
}: VoiceInputButtonProps) {
  useLibraryStyleSheet("voice-input-button", voiceInputButtonStyles);
  const pressStartedAt = useRef(0);
  const pressIntent = useRef<"start" | "stop" | "exit-passive" | "cancel" | "none">("none");
  const skipClick = useRef(false);
  const active = state === "recording" || state === "recovering";
  const cancellable = state === "transcribing" && !disabled && Boolean(onCancel);
  const canInteract =
    !disabled &&
    state !== "preparing" &&
    (state !== "transcribing" || cancellable) &&
    state !== "unavailable";
  const progress = Math.max(0, Math.min(1, timeoutProgress));
  const normalizedLevel = Math.max(0, Math.min(1, level));
  const accessibleLabel =
    props["aria-label"] ?? (cancellable ? "Cancel transcription" : labels[state]);
  const handlePointerDown = useCallback(
    (event: PointerEvent<HTMLButtonElement>) => {
      event.preventDefault();
      if (!canInteract) return;
      skipClick.current = true;
      pressStartedAt.current = Date.now();
      if (cancellable) {
        pressIntent.current = "cancel";
        return;
      }
      if (active) {
        pressIntent.current = state === "recovering" && onExitPassive ? "exit-passive" : "stop";
        return;
      }
      pressIntent.current = "start";
      onStart?.();
    },
    [active, canInteract, cancellable, onExitPassive, onStart, state],
  );
  const finishPointer = useCallback(() => {
    const intent = pressIntent.current;
    pressIntent.current = "none";
    if (intent === "cancel") onCancel?.();
    if (intent === "stop") onStop?.();
    if (intent === "exit-passive") onExitPassive?.();
    if (intent === "start" && Date.now() - pressStartedAt.current >= LONG_PRESS_MS) onStop?.();
  }, [onCancel, onExitPassive, onStop]);
  const handleClick = useCallback(() => {
    if (skipClick.current) {
      skipClick.current = false;
      return;
    }
    if (!canInteract) return;
    if (cancellable) onCancel?.();
    else if (active) {
      if (state === "recovering" && onExitPassive) onExitPassive();
      else onStop?.();
    } else onStart?.();
  }, [active, canInteract, cancellable, onCancel, onExitPassive, onStart, onStop, state]);
  const glyph: VoiceInputGlyphKind =
    state === "transcribing"
      ? "loader"
      : state === "error" || state === "unavailable"
        ? "alert"
        : "mic";

  return (
    <IconButton
      {...props}
      data-rcl-voice-input
      data-state={state}
      data-cancellable={cancellable ? "true" : undefined}
      data-mode={mode}
      data-size={size}
      data-density={density}
      data-testid={
        (props as VoiceInputButtonProps & { "data-testid"?: string })["data-testid"] ??
        "voice-input-control"
      }
      type="button"
      aria-label={accessibleLabel}
      aria-pressed={active}
      disableTooltip
      disabled={disabled || state === "unavailable"}
      onFocus={onPrepare}
      onPointerEnter={onPrepare}
      onPointerDown={handlePointerDown}
      onPointerUp={finishPointer}
      onClick={handleClick}
      onPointerCancel={(event) => {
        finishPointer();
        onPointerCancel?.(event);
      }}
      size={size}
      density={density}
      variant="secondary"
      className={className}
      style={{ overflow: "visible", touchAction: "manipulation", ...props.style }}
    >
      <Glyph
        kind={glyph}
        className={iconClassName}
        style={{ position: "relative", zIndex: 1 }}
        data-rcl-voice-glyph-role="primary"
      />
      {cancellable ? (
        <Glyph
          kind="cancel"
          className={iconClassName}
          style={{ position: "relative", zIndex: 1 }}
          data-rcl-voice-glyph-role="cancel"
        />
      ) : null}
      <span data-rcl-voice-clip>
        {active ? (
          <span
            data-rcl-voice-level
            data-mode={mode}
            aria-hidden="true"
            style={{
              blockSize: `${Math.round(normalizedLevel * 100)}%`,
              height: `${Math.round(normalizedLevel * 100)}%`,
            }}
          />
        ) : null}
        {state === "recording" && mode === "timeout" ? (
          <svg data-rcl-voice-timeout-ring aria-hidden="true" viewBox="0 0 44 44">
            <circle
              cx="22"
              cy="22"
              r="18"
              fill="none"
              stroke="currentColor"
              strokeWidth="3"
              strokeDasharray={RING_CIRCUMFERENCE}
              strokeDashoffset={RING_CIRCUMFERENCE * (1 - progress)}
              strokeLinecap="round"
            />
          </svg>
        ) : null}
      </span>
      <span
        data-rcl-voice-visually-hidden
        style={{
          position: "absolute",
          inlineSize: 1,
          blockSize: 1,
          overflow: "hidden",
          clipPath: "inset(50%)",
          whiteSpace: "nowrap",
        }}
      >
        {labels[state]}
      </span>
    </IconButton>
  );
}
