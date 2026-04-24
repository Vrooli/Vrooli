import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown, Loader2 } from "lucide-react";
import { cn } from "../../lib/classnames";

export type SummarizationLevel = "light" | "moderate" | "heavy";

const LEVEL_OPTIONS: { value: SummarizationLevel; label: string; hint: string }[] = [
  { value: "light", label: "Light", hint: "~60% of original" },
  { value: "moderate", label: "Moderate", hint: "~40% of original" },
  { value: "heavy", label: "Heavy", hint: "2-3 sentences" },
];

export interface PlaybackModeControlProps {
  testIdPrefix: string;
  isSummarized: boolean;
  hasOriginalVersion: boolean;
  canSummarize: boolean;
  isSummarizing: boolean;
  currentLevel: SummarizationLevel;
  /**
   * When true, the control renders visible but non-interactive — used on the
   * playback bar in idle/replay states so the bar's shape stays constant.
   */
  disabled?: boolean;
  onToggleSummarized?: (useSummarized: boolean) => void;
  onChangeLevel?: (level: SummarizationLevel) => void;
}

/**
 * Compact inline control that replaces both the pill-badge indicator and the
 * popover Summarized/Original toggle. Shows current mode and opens a dropdown
 * for switching between Original and summarization levels.
 *
 * Rendering rules:
 *   - isSummarized                         → "Summarized ▾"
 *   - !isSummarized && hasOriginalVersion  → "Original ▾"
 *   - !hasOriginal && canSummarize         → "Summarize ▾"
 *   - !hasOriginal && !canSummarize        → nothing
 */
export function PlaybackModeControl({
  testIdPrefix,
  isSummarized,
  hasOriginalVersion,
  canSummarize,
  isSummarizing,
  currentLevel,
  disabled: disabledProp = false,
  onToggleSummarized,
  onChangeLevel,
}: PlaybackModeControlProps) {
  const [open, setOpen] = useState(false);
  const buttonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open]);

  const getMenuStyle = useCallback((): React.CSSProperties => {
    const btn = buttonRef.current;
    if (!btn) return { position: "fixed", bottom: 48, left: 16 };
    const rect = btn.getBoundingClientRect();
    return {
      position: "fixed",
      bottom: window.innerHeight - rect.top + 6,
      left: Math.max(8, rect.left),
      minWidth: Math.max(180, rect.width),
    };
  }, []);

  // No control when there's neither a summary nor a way to get one.
  if (!hasOriginalVersion && !canSummarize) return null;

  const label = isSummarized
    ? "Summarized"
    : hasOriginalVersion
      ? "Original"
      : "Summarize";

  const handleSelectOriginal = () => {
    setOpen(false);
    if (hasOriginalVersion) onToggleSummarized?.(false);
  };

  const handleSelectLevel = (level: SummarizationLevel) => {
    setOpen(false);
    const alreadyAtLevel = isSummarized && level === currentLevel;
    if (alreadyAtLevel) return;
    onChangeLevel?.(level);
  };

  const disabled = isSummarizing || disabledProp;

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        data-testid={`${testIdPrefix}-mode-control`}
        aria-haspopup="menu"
        aria-expanded={open}
        disabled={disabled}
        onClick={() => setOpen((prev) => !prev)}
        className={cn(
          "inline-flex shrink-0 items-center gap-0.5 rounded-md px-1.5 py-1 text-[11px] font-medium transition",
          isSummarized
            ? "bg-amber-500/15 text-amber-300 hover:bg-amber-500/25 ring-1 ring-amber-500/30"
            : "bg-wc-surface-base text-wc-text-muted hover:bg-wc-surface-input ring-1 ring-wc-default",
          isSummarizing && "cursor-wait",
          disabled && "opacity-60",
        )}
        title={isSummarized ? "Summarized — click to switch" : hasOriginalVersion ? "Original — click to switch" : "Click to summarize"}
      >
        {isSummarizing
          ? <Loader2 className="h-3 w-3 animate-spin" />
          : <ChevronDown className="h-3 w-3" />}
        <span>{label}</span>
      </button>

      {open && createPortal(
        <>
          <div
            data-testid={`${testIdPrefix}-mode-menu-backdrop`}
            className="fixed inset-0 z-[60]"
            onClick={() => setOpen(false)}
          />
          <div
            data-testid={`${testIdPrefix}-mode-menu`}
            role="menu"
            className="z-[61] rounded-xl border border-wc-default bg-wc-surface-raised p-1 shadow-lg"
            style={getMenuStyle()}
          >
            {hasOriginalVersion && (
              <button
                type="button"
                role="menuitem"
                data-testid={`${testIdPrefix}-mode-option-original`}
                onClick={handleSelectOriginal}
                className={cn(
                  "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs transition hover:bg-wc-accent/10",
                  !isSummarized ? "text-wc-accent" : "text-wc-text-muted",
                )}
              >
                <span className="flex h-3 w-3 items-center justify-center">
                  {!isSummarized && <Check className="h-3 w-3" />}
                </span>
                <span className="flex-1">Original</span>
              </button>
            )}
            {hasOriginalVersion && (
              <div className="my-1 h-px bg-wc-default" />
            )}
            {LEVEL_OPTIONS.map(({ value, label: levelLabel, hint }) => {
              const isActive = isSummarized && value === currentLevel;
              return (
                <button
                  key={value}
                  type="button"
                  role="menuitem"
                  data-testid={`${testIdPrefix}-mode-option-${value}`}
                  onClick={() => handleSelectLevel(value)}
                  className={cn(
                    "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs transition hover:bg-amber-500/10",
                    isActive ? "text-amber-300" : "text-wc-text-muted",
                  )}
                >
                  <span className="flex h-3 w-3 items-center justify-center">
                    {isActive && <Check className="h-3 w-3" />}
                  </span>
                  <span className="flex-1">{levelLabel}</span>
                  <span className="text-[10px] text-wc-text-faint">{hint}</span>
                </button>
              );
            })}
          </div>
        </>,
        document.body,
      )}
    </>
  );
}
