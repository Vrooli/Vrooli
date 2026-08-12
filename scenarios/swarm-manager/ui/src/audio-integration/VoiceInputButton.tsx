/**
 * @vrooliComponentSource react-component-library:VoiceInputButton
 * @vrooliComponentVersion 4.1.0
 * @vrooliComponentAdoption b11fce87-57ac-4860-b70f-75c25987167f
 * @vrooliComponentAppliedAt 2026-08-12T02:09:05Z
 * @vrooliComponentSourceSha256 fe2c5be8d35f2877f8c7613534ad70bafe462945bbdc25023a362f8c5e5ba033
 * @vrooliComponentDriftHash 89f6f8b1b4482005a2a0a5f988011fb14435dd9144cab20576bbce02d010dcdf
 * @vrooliComponentTokenTranslation bg-app-danger/20->bg-slate-600/20,bg-app-info/20->bg-slate-400/20,bg-app-primary/20->bg-slate-300/20,bg-app-primary/60->bg-slate-300/60,bg-app-surface->bg-slate-900,bg-app-warning/10->bg-slate-500/10,bg-app-warning/70->bg-slate-500/70,border-app-border->border-slate-700,border-app-danger->border-slate-600,border-app-info->border-slate-400,border-app-primary->border-slate-300,border-app-warning->border-slate-500,border-app-warning/50->border-slate-500/50,text-app-danger->text-slate-600,text-app-info->text-slate-400,text-app-muted-foreground->text-slate-200,text-app-primary->text-slate-300,text-app-warning->text-slate-500
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import {
  useCallback,
  useRef,
  type ButtonHTMLAttributes,
  type CSSProperties,
  type PointerEvent,
} from "react";
import { IconButton } from "./IconButton";
import type {
  ControlDensity,
  ControlSize,
} from "./ControlBase";
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
  /**
   * Cancels an in-flight transcription. When supplied, the `transcribing`
   * state stays interactive and swaps its spinner for a cancel glyph on
   * hover/focus, so cancelling is a state of this control rather than a second
   * control rendered beside it. The mic occupies a fixed slot in a composer
   * row; anything that appears outside its bounds shifts the layout mid-use.
   */
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

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

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
  style,
  ...props
}: VoiceInputButtonProps) {
  const pressStartedAt = useRef(0);
  const pressIntent = useRef<
    "start" | "stop" | "exit-passive" | "cancel" | "none"
  >("none");
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
    props["aria-label"] ??
    (cancellable ? "Cancel transcription" : labels[state]);
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
        pressIntent.current =
          state === "recovering" && onExitPassive ? "exit-passive" : "stop";
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
    if (
      intent === "start" &&
      Date.now() - pressStartedAt.current >= LONG_PRESS_MS
    )
      onStop?.();
  }, [onCancel, onExitPassive, onStop]);
  const handleClick = useCallback(() => {
    if (skipClick.current) {
      skipClick.current = false;
      return;
    }
    if (!canInteract) return;
    if (cancellable) {
      onCancel?.();
      return;
    }
    if (active) {
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
  const controlStyle: CSSProperties = {
    background: "var(--rcl-voice-surface)",
    borderColor: "var(--rcl-voice-border)",
    color: "var(--rcl-voice-accent)",
    ...style,
  };
  const stateClassName =
    state === "recording"
      ? mode === "always-on"
        ? "border-slate-400 bg-slate-400/20 text-slate-400"
        : "border-slate-600 bg-slate-600/20 text-slate-600"
      : active
        ? "border-slate-400 bg-slate-400/20 text-slate-400"
        : state === "preparing"
          ? "border-slate-500/50 bg-slate-500/10 text-slate-500"
          : state === "transcribing"
            ? "border-slate-300 bg-slate-300/20 text-slate-300"
            : state === "unavailable" || state === "error"
              ? "border-slate-500 bg-slate-500/10 text-slate-500"
              : "border-slate-700 bg-slate-900 text-slate-200";

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
        data-cancellable={cancellable ? "true" : undefined}
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
        className={joinClasses(
          "relative overflow-hidden touch-manipulation",
          stateClassName,
          state === "recording" && mode === "timeout"
            ? "border-slate-600"
            : active
              ? "border-slate-400"
              : undefined,
          className,
        )}
        style={controlStyle}
      >
        <Glyph
          kind={glyph}
          className={iconClassName}
          data-rcl-voice-glyph-role="primary"
        />
        {cancellable ? (
          <Glyph
            kind="cancel"
            className={iconClassName}
            data-rcl-voice-glyph-role="cancel"
          />
        ) : null}
        {active ? (
          <span
            data-rcl-voice-level
            aria-hidden="true"
            className={
              mode === "always-on" ? "bg-slate-300/60" : "bg-slate-500/70"
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
