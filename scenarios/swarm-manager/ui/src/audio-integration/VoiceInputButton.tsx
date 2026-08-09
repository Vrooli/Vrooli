/**
 * @vrooliComponentSource react-component-library:VoiceInputButton
 * @vrooliComponentVersion 4.0.0
 * @vrooliComponentAdoption b11fce87-57ac-4860-b70f-75c25987167f
 * @vrooliComponentAppliedAt 2026-08-09T14:56:12Z
 * @vrooliComponentSourceSha256 2abb7da2a6fd2ee0e8527869fb291913bf26ce8dc47495af4e4ec56a82129a8c
 * @vrooliComponentDriftHash ad08c59ec05f828029a2e0475a63f50f7e5eebc8647956d3e47c2e78c43d16e2
 * @vrooliComponentTokenTranslation bg-app-danger/20->bg-slate-600/20,bg-app-info/20->bg-slate-400/20,bg-app-primary/20->bg-slate-300/20,bg-app-primary/60->bg-slate-300/60,bg-app-surface->bg-slate-900,bg-app-warning/10->bg-slate-500/10,bg-app-warning/70->bg-slate-500/70,border-app-border->border-slate-700,border-app-danger->border-slate-600,border-app-info->border-slate-400,border-app-primary->border-slate-300,border-app-warning->border-slate-500,border-app-warning/50->border-slate-500/50,text-app-danger->text-slate-600,text-app-info->text-slate-400,text-app-muted-foreground->text-slate-200,text-app-primary->text-slate-300,text-app-warning->text-slate-500,text-app-warning/80->text-slate-500/80
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import {
  useCallback,
  useRef,
  type ButtonHTMLAttributes,
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

/** One fixed-footprint microphone control. Recovery belongs to the consumer. */
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
  ...props
}: VoiceInputButtonProps) {
  const pressStartedAt = useRef(0);
  const pressIntent = useRef<"start" | "stop" | "exit-passive" | "none">("none");
  const skipClick = useRef(false);
  const active = state === "recording" || state === "recovering";
  const canInteract = !disabled && state !== "preparing" && state !== "transcribing" && state !== "unavailable";
  const progress = Math.max(0, Math.min(1, timeoutProgress));
  const normalizedLevel = Math.max(0, Math.min(1, level));
  const accessibleLabel = props["aria-label"] ?? labels[state];

  const handlePointerDown = useCallback((event: PointerEvent<HTMLButtonElement>) => {
    event.preventDefault();
    if (!canInteract) return;
    skipClick.current = true;
    pressStartedAt.current = Date.now();
    if (active) {
      pressIntent.current = state === "recovering" && onExitPassive ? "exit-passive" : "stop";
      return;
    }
    pressIntent.current = "start";
    onStart?.();
  }, [active, canInteract, onExitPassive, onStart, state]);

  const finishPointer = useCallback(() => {
    const intent = pressIntent.current;
    pressIntent.current = "none";
    if (intent === "stop") onStop?.();
    if (intent === "exit-passive") onExitPassive?.();
    if (intent === "start" && Date.now() - pressStartedAt.current >= LONG_PRESS_MS) onStop?.();
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

  const stateClasses: Record<VoiceInputButtonState, string> = {
    preparing: "border-slate-500/50 bg-slate-500/10 text-slate-500",
    recording: mode === "always-on" ? "border-slate-400 bg-slate-400/20 text-slate-400" : "border-slate-600 bg-slate-600/20 text-slate-600",
    recovering: "border-slate-400 bg-slate-400/20 text-slate-400",
    transcribing: "border-slate-300 bg-slate-300/20 text-slate-300",
    unavailable: "border-slate-500 bg-slate-500/10 text-slate-500",
    error: "border-slate-500 bg-slate-500/10 text-slate-500",
    idle: "border-slate-700 bg-slate-900 text-slate-200",
  };
  const glyph: VoiceInputGlyphKind = state === "transcribing" ? "loader" : state === "error" || state === "unavailable" ? "alert" : "mic";
  const glyphClass = `relative z-10 ${size === "xs" ? "h-3 w-3" : size === "icon" ? "h-5 w-5" : size === "lg" || size === "xl" ? "h-5 w-5" : "h-4 w-4"} ${state === "preparing" || state === "recovering" ? "animate-pulse" : state === "transcribing" ? "animate-spin" : ""} ${iconClassName ?? ""}`;

  return (
    <IconButton
      {...props}
      data-testid={(props as VoiceInputButtonProps & { "data-testid"?: string })["data-testid"] ?? "voice-input-control"}
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
      onPointerCancel={(event) => { finishPointer(); onPointerCancel?.(event); }}
      size={size}
      density={density}
      variant="secondary"
      className={`relative overflow-hidden touch-manipulation ${stateClasses[state]} ${props.className ?? ""}`}
    >
      <Glyph kind={glyph} className={glyphClass} />
      {active && <span aria-hidden="true" className={`pointer-events-none absolute inset-x-0 bottom-0 z-0 rounded-[inherit] transition-[height] duration-75 ${state === "recovering" || mode === "always-on" ? "bg-slate-300/60" : "bg-slate-500/70"}`} style={{ height: `${Math.round(normalizedLevel * 100)}%` }} />}
      {state === "recording" && mode === "timeout" && <svg aria-hidden="true" className="pointer-events-none absolute left-1/2 top-1/2 z-10 aspect-square h-[calc(100%-4px)] min-h-6 max-h-8 -translate-x-1/2 -translate-y-1/2 -rotate-90 overflow-visible" viewBox="0 0 44 44"><circle cx="22" cy="22" r="18" fill="none" stroke="currentColor" strokeWidth="3" className="text-slate-500/80" strokeDasharray={RING_CIRCUMFERENCE} strokeDashoffset={RING_CIRCUMFERENCE * (1 - progress)} strokeLinecap="round" /></svg>}
      <span className="sr-only">{labels[state]}</span>
    </IconButton>
  );
}
