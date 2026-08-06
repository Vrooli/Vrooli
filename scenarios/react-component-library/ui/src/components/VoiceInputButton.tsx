/**
 * @vrooliComponentSource react-component-library:VoiceInputButton
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 228de948-666a-48fb-a8d8-00b57b6b9309
 * @vrooliComponentAppliedAt 2026-08-06T03:59:04Z
 * @vrooliComponentSourceSha256 54ef7ef16e9fc255f39c81c0293a3cc9ee79c4560f2338c0a5d8e99e35f74d9f
 * @vrooliComponentDriftHash c03762cf99cc22bf222af82ee3f820eae26dc2c1e4b9d1b5a522c9a6fe5cd28a
 * @vrooliComponentTokenTranslation bg-app-danger/20->bg-app-danger/20,bg-app-danger/30->bg-app-danger/30,bg-app-info/20->bg-app-info/20,bg-app-info/30->bg-app-info/30,bg-app-primary/20->bg-app-primary/20,bg-app-surface->bg-app-surface,bg-app-warning/10->bg-app-warning/10,bg-app-warning/15->bg-app-warning/15,border-app-border->border-app-border,border-app-danger->border-app-danger,border-app-info->border-app-info,border-app-primary->border-app-primary,border-app-warning->border-app-warning,border-app-warning/50->border-app-warning/50,ring-app-primary/50->ring-app-primary/50,text-app-danger->text-app-danger,text-app-info->text-app-info,text-app-muted-foreground->text-app-muted-foreground,text-app-primary->text-app-primary,text-app-warning->text-app-warning,text-app-warning/80->text-app-warning/80
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { AlertCircle, Loader2, Mic, X } from "lucide-react";
import { useCallback, useRef, useState, type ButtonHTMLAttributes, type PointerEvent } from "react";

export type VoiceInputButtonState =
  | "idle"
  | "preparing"
  | "recording"
  | "recovering"
  | "transcribing"
  | "unavailable"
  | "error";

export interface VoiceInputButtonProps
  extends Omit<
    ButtonHTMLAttributes<HTMLButtonElement>,
    "children" | "onPointerDown" | "onPointerUp"
  > {
  readonly state?: VoiceInputButtonState;
  readonly mode?: "always-on" | "timeout";
  readonly level?: number;
  readonly timeoutProgress?: number;
  readonly error?: string;
  readonly rejectionReason?: string;
  readonly partialTranscript?: string;
  readonly onStart?: () => void;
  readonly onStop?: () => void;
  readonly onPrepare?: () => void;
  readonly onDismissError?: () => void;
  readonly onTranscribeAnyway?: () => void;
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

/**
 * Presentation and pointer interaction parity with web-console's VoiceMicButton.
 * Capture and transport remain outside this component: pass the hook's state and
 * start/stop callbacks rather than importing scenario behaviour.
 */
export function VoiceInputButton({
  state = "idle",
  mode = "always-on",
  level = 0,
  timeoutProgress = 0,
  error,
  rejectionReason,
  partialTranscript,
  onStart,
  onStop,
  onPrepare,
  onDismissError,
  onTranscribeAnyway,
  className = "",
  disabled,
  onPointerCancel,
  ...props
}: VoiceInputButtonProps) {
  const [dismissedError, setDismissedError] = useState<string | undefined>();
  const pressStartedAt = useRef(0);
  const pressIntent = useRef<"start" | "stop" | "none">("none");
  const active = state === "recording" || state === "recovering";
  const canInteract =
    !disabled && state !== "preparing" && state !== "transcribing" && state !== "unavailable";
  const visibleError = state === "error" && error && error !== dismissedError;
  const progress = Math.max(0, Math.min(1, timeoutProgress));
  const normalizedLevel = Math.max(0, Math.min(1, level));
  const handlePointerDown = useCallback(
    (event: PointerEvent<HTMLButtonElement>) => {
      event.preventDefault();
      if (!canInteract) return;
      pressStartedAt.current = Date.now();
      if (active) {
        pressIntent.current = "stop";
        return;
      }
      pressIntent.current = "start";
      onStart?.();
    },
    [active, canInteract, onStart],
  );
  const finishPointer = useCallback(() => {
    const intent = pressIntent.current;
    pressIntent.current = "none";
    if (intent === "stop") onStop?.();
    if (intent === "start" && Date.now() - pressStartedAt.current >= LONG_PRESS_MS) onStop?.();
  }, [onStop]);
  const stateClasses: Record<VoiceInputButtonState, string> = {
    preparing: "border-yellow-500/50 bg-yellow-500/10 text-yellow-500",
    recording:
      mode === "always-on"
        ? "border-app-info bg-app-info/20 text-app-info"
        : "border-app-danger bg-app-danger/20 text-app-danger",
    recovering: "border-app-info bg-app-info/20 text-app-info",
    transcribing: "border-app-primary bg-app-primary/20 text-app-primary",
    unavailable: "border-app-warning bg-app-warning/10 text-app-warning",
    error: "border-app-warning bg-app-warning/10 text-app-warning",
    idle: "border-app-border bg-app-surface text-app-muted-foreground",
  };

  return (
    <div className={`relative shrink-0 ${className}`}>
      <button
        type="button"
        aria-label={labels[state]}
        aria-pressed={active}
        title={labels[state]}
        disabled={disabled || state === "unavailable"}
        onFocus={onPrepare}
        onPointerEnter={onPrepare}
        onPointerDown={handlePointerDown}
        onPointerUp={finishPointer}
        onPointerCancel={(event) => {
          finishPointer();
          onPointerCancel?.(event);
        }}
        className={`relative shrink-0 overflow-hidden rounded border px-1.5 py-1 text-xs font-medium transition active:brightness-95 touch-manipulation focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:cursor-not-allowed disabled:opacity-60 ${stateClasses[state]}`}
        {...props}
      >
        {active && (
          <span
            aria-hidden="true"
            className={`absolute inset-x-0 bottom-0 rounded-[inherit] transition-[height] duration-75 ${state === "recovering" || mode === "always-on" ? "bg-app-info/30" : "bg-app-danger/30"}`}
            style={{ height: `${Math.round(normalizedLevel * 100)}%` }}
          />
        )}
        {state === "recording" && mode === "timeout" && (
          <svg
            aria-hidden="true"
            className="pointer-events-none absolute left-1/2 top-1/2 z-10 aspect-square h-[calc(100%-4px)] min-h-6 max-h-8 -translate-x-1/2 -translate-y-1/2 -rotate-90 overflow-visible"
            viewBox="0 0 44 44"
          >
            <circle
              cx="22"
              cy="22"
              r="18"
              fill="none"
              stroke="currentColor"
              strokeWidth="3"
              className="text-app-warning/80"
              strokeDasharray={RING_CIRCUMFERENCE}
              strokeDashoffset={RING_CIRCUMFERENCE * (1 - progress)}
              strokeLinecap="round"
            />
          </svg>
        )}
        {state === "preparing" || state === "recovering" ? (
          <Mic aria-hidden="true" className="relative h-3.5 w-3.5 animate-pulse" />
        ) : state === "transcribing" ? (
          <Loader2 aria-hidden="true" className="relative h-3.5 w-3.5 animate-spin" />
        ) : state === "error" || state === "unavailable" ? (
          <AlertCircle aria-hidden="true" className="relative h-3.5 w-3.5" />
        ) : (
          <Mic aria-hidden="true" className="relative h-3.5 w-3.5" />
        )}
      </button>
      {visibleError && (
        <div
          role="status"
          className="absolute bottom-full left-1/2 z-10 mb-1 flex w-52 -translate-x-1/2 items-start gap-1.5 rounded border border-app-warning/50 bg-app-surface px-2 py-1 text-[10px] text-app-warning shadow-lg"
        >
          <span className="min-w-0 flex-1 break-words">{error}</span>
          <button
            type="button"
            className="rounded p-0.5 hover:bg-app-warning/15"
            aria-label="Dismiss voice input error"
            onClick={() => {
              setDismissedError(error);
              onDismissError?.();
            }}
          >
            <X aria-hidden="true" className="h-3 w-3" />
          </button>
        </div>
      )}
      {active && partialTranscript && (
        <div className="pointer-events-none absolute bottom-full left-1/2 mb-1 max-w-[200px] -translate-x-1/2 overflow-hidden text-ellipsis whitespace-nowrap rounded border border-app-border bg-app-surface px-2 py-1 text-[10px] text-app-muted-foreground shadow-lg">
          {partialTranscript}
        </div>
      )}
      {rejectionReason && onTranscribeAnyway && (
        <button
          type="button"
          className="ml-2 text-sm text-app-primary underline"
          onClick={onTranscribeAnyway}
        >
          Transcribe anyway
        </button>
      )}
    </div>
  );
}
