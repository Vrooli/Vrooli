/**
 * @libraryId react-component-library:VoiceInputButton
 * @version 4.1.0
 * @status released
 * @deps {"react":"^18"}
 */
import { cn } from "../../../../foundations/ClassMerge/versions/1.0.0/ClassMerge";
import {
  useCallback,
  useRef,
  type ButtonHTMLAttributes,
  type CSSProperties,
  type PointerEvent,
} from "react";
import { IconButton } from "../../../IconButton/versions/2.0.0/IconButton";
import type {
  ControlDensity,
  ControlSize,
} from "../../../ControlBase/versions/1.0.0/ControlBase";
import {
  VoiceInputButtonGlyph as Glyph,
  type VoiceInputGlyphKind,
} from "./VoiceInputButtonGlyph";
import { voiceInputButtonStyles } from "./styles";

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
  iconClassName,
  disabled,
  onPointerCancel,
  className,
  style,
  ...props
}: VoiceInputButtonProps) {
  const pressStartedAt = useRef(0);
  const pressIntent = useRef<"start" | "stop" | "exit-passive" | "none">(
    "none",
  );
  const skipClick = useRef(false);
  const active = state === "recording" || state === "recovering";
  const canInteract =
    !disabled &&
    state !== "preparing" &&
    state !== "transcribing" &&
    state !== "unavailable";
  const progress = Math.max(0, Math.min(1, timeoutProgress));
  const normalizedLevel = Math.max(0, Math.min(1, level));
  const accessibleLabel = props["aria-label"] ?? labels[state];
  const handlePointerDown = useCallback(
    (event: PointerEvent<HTMLButtonElement>) => {
      event.preventDefault();
      if (!canInteract) return;
      skipClick.current = true;
      pressStartedAt.current = Date.now();
      if (active) {
        pressIntent.current =
          state === "recovering" && onExitPassive ? "exit-passive" : "stop";
        return;
      }
      pressIntent.current = "start";
      onStart?.();
    },
    [active, canInteract, onExitPassive, onStart, state],
  );
  const finishPointer = useCallback(() => {
    const intent = pressIntent.current;
    pressIntent.current = "none";
    if (intent === "stop") onStop?.();
    if (intent === "exit-passive") onExitPassive?.();
    if (
      intent === "start" &&
      Date.now() - pressStartedAt.current >= LONG_PRESS_MS
    )
      onStop?.();
  }, [onExitPassive, onStop]);
  const handleClick = useCallback(() => {
    if (skipClick.current) {
      skipClick.current = false;
      return;
    }
    if (!canInteract) return;
    if (active) {
      if (state === "recovering" && onExitPassive) onExitPassive();
      else onStop?.();
    } else onStart?.();
  }, [active, canInteract, onExitPassive, onStart, onStop, state]);
  const glyph: VoiceInputGlyphKind =
    state === "transcribing"
      ? "loader"
      : state === "error" || state === "unavailable"
        ? "alert"
        : "mic";
  const controlStyle: CSSProperties = {
    background: "var(--rcl-voice-surface)",
    borderColor: "var(--rcl-voice-border)",
    color: "var(--rcl-voice-accent)",
    ...style,
  };
  const stateClassName =
    state === "recording"
      ? mode === "always-on"
        ? "border-app-info bg-app-info/20 text-app-info"
        : "border-app-danger bg-app-danger/20 text-app-danger"
      : active
        ? "border-app-info bg-app-info/20 text-app-info"
        : state === "preparing"
          ? "border-app-warning/50 bg-app-warning/10 text-app-warning"
          : state === "transcribing"
            ? "border-app-primary bg-app-primary/20 text-app-primary"
            : state === "unavailable" || state === "error"
              ? "border-app-warning bg-app-warning/10 text-app-warning"
              : "border-app-border bg-app-surface text-app-muted-foreground";

  return (
    <>
      <style
        data-rcl-voice-input-styles
        dangerouslySetInnerHTML={{ __html: voiceInputButtonStyles }}
      />
      <IconButton
        {...props}
        data-rcl-voice-input
        data-state={state}
        data-mode={mode}
        data-size={size}
        data-density={density}
        data-testid={
          (props as VoiceInputButtonProps & { "data-testid"?: string })[
            "data-testid"
          ] ?? "voice-input-control"
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
        className={cn(
          "relative overflow-hidden touch-manipulation",
          stateClassName,
          state === "recording" && mode === "timeout"
            ? "border-app-danger"
            : active
              ? "border-app-info"
              : undefined,
          className,
        )}
        style={controlStyle}
      >
        <Glyph kind={glyph} className={iconClassName} />
        {active ? (
          <span
            data-rcl-voice-level
            aria-hidden="true"
            className={
              mode === "always-on" ? "bg-app-primary/60" : "bg-app-warning/70"
            }
            style={{
              blockSize: `${Math.round(normalizedLevel * 100)}%`,
              height: `${Math.round(normalizedLevel * 100)}%`,
            }}
          />
        ) : null}
        {state === "recording" && mode === "timeout" ? (
          <svg
            data-rcl-voice-timeout-ring
            aria-hidden="true"
            viewBox="0 0 44 44"
          >
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
        <span data-rcl-voice-visually-hidden>{labels[state]}</span>
      </IconButton>
    </>
  );
}
