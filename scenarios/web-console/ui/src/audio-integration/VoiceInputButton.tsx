/**
 * @vrooliComponentSource react-component-library:VoiceInputButton
 * @vrooliComponentVersion 4.2.0
 * @vrooliComponentAdoption fca0af9a-3a97-46e6-b43a-b8c6504d9361
 * @vrooliComponentAppliedAt 2026-08-20T01:50:37Z
 * @vrooliComponentSourceSha256 014efbbd48444f064802cf949f935d3e14679f55ceff0f0eb7b76c55bf3eccab
 * @vrooliComponentDriftHash e91aa365f3f49a1ba2ed289ad80494d759a7f21aaed5e4e5ae9a271249d2109d
 * @vrooliComponentTokenTranslation bg-app-danger/20->bg-wc-error-surface/20,bg-app-info/20->bg-wc-accent/20,bg-app-primary/20->bg-wc-accent-active/20,bg-app-primary/60->bg-wc-accent-active/60,bg-app-surface->bg-wc-surface-raised,bg-app-warning/10->bg-wc-accent/10,bg-app-warning/70->bg-wc-accent/70,border-app-border->border-wc-default,border-app-danger->border-wc-error,border-app-info->border-wc-accent,border-app-primary->border-wc-accent,border-app-warning->border-wc-accent,border-app-warning/50->border-wc-accent/50,text-app-danger->text-wc-error-text,text-app-info->text-wc-accent,text-app-muted-foreground->text-wc-text-muted,text-app-primary->text-wc-accent,text-app-warning->text-wc-accent
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
  /** Cancel an in-flight transcription without adding a second control. */
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
        ? "border-wc-accent bg-wc-accent/20 text-wc-accent"
        : "border-wc-error bg-wc-error-surface/20 text-wc-error-text"
      : active
        ? "border-wc-accent bg-wc-accent/20 text-wc-accent"
        : state === "preparing"
          ? "border-wc-accent/50 bg-wc-accent/10 text-wc-accent"
          : state === "transcribing"
            ? "border-wc-accent bg-wc-accent-active/20 text-wc-accent"
            : state === "unavailable" || state === "error"
              ? "border-wc-accent bg-wc-accent/10 text-wc-accent"
              : "border-wc-default bg-wc-surface-raised text-wc-text-muted";

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
            ? "border-wc-error"
            : active
              ? "border-wc-accent"
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
              mode === "always-on" ? "bg-wc-accent-active/60" : "bg-wc-accent/70"
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