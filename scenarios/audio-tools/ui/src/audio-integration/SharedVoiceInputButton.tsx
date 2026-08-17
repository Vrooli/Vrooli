/**
 * @vrooliComponentSource react-component-library:VoiceInputButton
 * @vrooliComponentVersion 3.0.0
 * @vrooliComponentAdoption 0ee254b3-a523-4ebf-b17f-8a763bcef3b5
 * @vrooliComponentAppliedAt 2026-08-05T04:36:00Z
 * @vrooliComponentSourceSha256 cec62efe50b286699552e6c4678f7e6e9a2db998ef6bacf3082bfe86b9010c6b
 * @vrooliComponentDriftHash bec785d91eb22af5c7bf7855b59022ab1b1dd04f4b805a1ab47ea61682a73928
 * @vrooliComponentTokenTranslation bg-app-danger/20->bg-app-danger/20,bg-app-info/20->bg-app-info/20,bg-app-primary/20->bg-app-primary/20,bg-app-primary/60->bg-app-primary/60,bg-app-surface->bg-app-surface,bg-app-warning/10->bg-app-warning/10,bg-app-warning/70->bg-app-warning/70,border-app-border->border-app-border,border-app-danger->border-app-danger,border-app-info->border-app-info,border-app-primary->border-app-primary,border-app-warning->border-app-warning,border-app-warning/50->border-app-warning/50,text-app-danger->text-app-danger,text-app-info->text-app-info,text-app-muted-foreground->text-app-muted-foreground,text-app-primary->text-app-primary,text-app-warning->text-app-warning,text-app-warning/80->text-app-warning/80
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 *
 * LOCAL EDIT — fixed footprint. The 3.0.0 snapshot rendered an error popover,
 * a partial-transcript tooltip, a "Transcribe anyway" link, and a recovery
 * action strip inside this control's own wrapper, so the control's footprint
 * changed with its state. Neither audio-tools call site ever supplied the props
 * that armed those branches, so they were unreachable here; they are removed
 * rather than left as a trap. Library 4.0.0+ made the same change upstream, and
 * 4.1.0 declares `fixed-footprint` in its experience contract. This file cannot
 * be re-adopted from 4.1.0 yet because the catalog maturity floor for adoption
 * is `verified` and every catalog asset is currently `scaffolded`.
 *
 * Status text and recovery actions are app chrome. See `LiveTry` and
 * `DictationRecorder`, which already render their own.
 */
import { useCallback, useRef, type ButtonHTMLAttributes, type PointerEvent } from "react";
import { IconButton } from "./IconButton";
import type { ControlDensity, ControlSize } from "./ControlBase";
import { VoiceInputButtonGlyph as Glyph, type VoiceInputGlyphKind } from "./VoiceInputButtonGlyph";

export type VoiceInputButtonState = "idle" | "preparing" | "recording" | "recovering" | "transcribing" | "unavailable" | "error";
export type VoiceInputButtonMode = "always-on" | "timeout";
export type ButtonSize = ControlSize;

export interface VoiceInputButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children" | "onPointerDown" | "onPointerUp" | "size"> {
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
  readonly backend?: string;
  readonly wrapperClassName?: string;
  readonly iconClassName?: string;
}

const LONG_PRESS_MS = 300;
const RING_CIRCUMFERENCE = 2 * Math.PI * 18;
const labels: Record<VoiceInputButtonState, string> = {
  idle: "Start voice input", preparing: "Preparing microphone", recording: "Stop voice input", recovering: "Listening for voice input", transcribing: "Transcribing voice input", unavailable: "Voice input unavailable", error: "Voice input error",
};

export function VoiceInputButton({ state = "idle", mode = "always-on", size = "sm", density = "comfortable", level = 0, timeoutProgress = 0, onStart, onStop, onPrepare, onExitPassive, backend, className, wrapperClassName, iconClassName, disabled, onPointerCancel, ...props }: VoiceInputButtonProps) {
  const testId = (props as VoiceInputButtonProps & { "data-testid"?: string })["data-testid"];
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
    if (active) { pressIntent.current = state === "recovering" && onExitPassive ? "exit-passive" : "stop"; return; }
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
    if (skipClick.current) { skipClick.current = false; return; }
    if (!canInteract) return;
    if (active) {
      if (state === "recovering" && onExitPassive) onExitPassive(); else onStop?.();
    } else onStart?.();
  }, [active, canInteract, onExitPassive, onStart, onStop, state]);
  const stateClasses: Record<VoiceInputButtonState, string> = {
    preparing: "border-app-warning/50 bg-app-warning/10 text-app-warning",
    recording: mode === "always-on" ? "border-app-info bg-app-info/20 text-app-info" : "border-app-danger bg-app-danger/20 text-app-danger",
    recovering: "border-app-info bg-app-info/20 text-app-info",
    transcribing: "border-app-primary bg-app-primary/20 text-app-primary",
    unavailable: "border-app-warning bg-app-warning/10 text-app-warning",
    error: "border-app-warning bg-app-warning/10 text-app-warning",
    idle: "border-app-border bg-app-surface text-app-muted-foreground",
  };
  const glyph: VoiceInputGlyphKind = state === "transcribing" ? "loader" : state === "error" || state === "unavailable" ? "alert" : "mic";
  const glyphClass = `relative z-10 ${size === "xs" ? "h-3 w-3" : size === "icon" ? "h-5 w-5" : size === "lg" || size === "xl" ? "h-5 w-5" : "h-4 w-4"} ${state === "preparing" || state === "recovering" ? "animate-pulse" : state === "transcribing" ? "animate-spin" : ""} ${iconClassName ?? ""}`;

  return <div className={`relative shrink-0 ${wrapperClassName ?? ""}`} data-voice-backend={backend}>
    <IconButton
      {...props}
      data-testid={testId ?? "voice-input-control"}
      type="button"
      aria-label={accessibleLabel}
      aria-pressed={active}
      title={labels[state]}
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
      className={`relative overflow-hidden touch-manipulation ${stateClasses[state]} ${className ?? ""}`}
    >
      <Glyph kind={glyph} className={glyphClass} />
      {active && <span aria-hidden="true" className={`pointer-events-none absolute inset-x-0 bottom-0 z-0 rounded-[inherit] transition-[height] duration-75 ${state === "recovering" || mode === "always-on" ? "bg-app-primary/60" : "bg-app-warning/70"}`} style={{ height: `${Math.round(normalizedLevel * 100)}%` }} />}
      {state === "recording" && mode === "timeout" && <svg aria-hidden="true" className="pointer-events-none absolute left-1/2 top-1/2 z-10 aspect-square h-[calc(100%-4px)] min-h-6 max-h-8 -translate-x-1/2 -translate-y-1/2 -rotate-90 overflow-visible" viewBox="0 0 44 44"><circle cx="22" cy="22" r="18" fill="none" stroke="currentColor" strokeWidth="3" className="text-app-warning/80" strokeDasharray={RING_CIRCUMFERENCE} strokeDashoffset={RING_CIRCUMFERENCE * (1 - progress)} strokeLinecap="round" /></svg>}
      <span className="sr-only">{labels[state]}</span>
    </IconButton>
  </div>;
}
