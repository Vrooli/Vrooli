import { useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown, Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "../../lib/classnames";
import { strings } from "../../consts/strings";
import { useAnchoredPopoverPosition, type FloatingPlacement } from "../../hooks/useFloatingPosition";
import { useEscapeKey } from "../../hooks/useEscapeKey";

export type SummarizationLevel = "light" | "moderate" | "heavy";

/** Anchored placement order for the mode menu opening above its trigger. */
const MENU_PLACEMENTS: FloatingPlacement[] = ["top-start", "top-end", "bottom-start", "bottom-end"];

const LEVEL_OPTIONS = [
  { value: "light", labelKey: strings.playbackMode.light, hintKey: strings.playbackMode.lightHint },
  { value: "moderate", labelKey: strings.playbackMode.moderate, hintKey: strings.playbackMode.moderateHint },
  { value: "heavy", labelKey: strings.playbackMode.heavy, hintKey: strings.playbackMode.heavyHint },
] as const satisfies readonly { value: SummarizationLevel; labelKey: string; hintKey: string }[];

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
 *   - isSummarized                         → "<Level> ▾"
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
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const buttonRef = useRef<HTMLButtonElement>(null);

  useEscapeKey(open, () => setOpen(false));

  // The menu anchors above the trigger, start-aligned, via the shared
  // anchored-floating math; it never renders narrower than the trigger.
  const menuRef = useRef<HTMLDivElement>(null);
  const anchoredStyle = useAnchoredPopoverPosition(open, buttonRef, menuRef, MENU_PLACEMENTS);
  const menuMinWidth = Math.max(180, buttonRef.current?.getBoundingClientRect().width ?? 0);

  // No control when there's neither a summary nor a way to get one.
  if (!hasOriginalVersion && !canSummarize) return null;

  const currentLevelEntry = LEVEL_OPTIONS.find(({ value }) => value === currentLevel);
  const currentLevelLabel = currentLevelEntry ? t(currentLevelEntry.labelKey) : t(strings.playbackMode.summarized);
  const label = isSummarized
    ? currentLevelLabel
    : hasOriginalVersion
      ? t(strings.playbackMode.original)
      : t(strings.playbackMode.summarize);
  const title = isSummarized
    ? t(strings.playbackMode.summaryTitle, { level: currentLevelLabel })
    : hasOriginalVersion
      ? t(strings.playbackMode.originalTitle)
      : t(strings.playbackMode.summarizeTitle);

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
        title={title}
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
            className="fixed inset-0 z-wc-popover-backdrop"
            onClick={() => setOpen(false)}
          />
          <div
            ref={menuRef}
            data-testid={`${testIdPrefix}-mode-menu`}
            role="menu"
            className="wc-stable-theme z-wc-popover rounded-xl border border-wc-default bg-wc-surface-raised p-1 shadow-lg"
            style={{ ...anchoredStyle, minWidth: menuMinWidth }}
          >
            {hasOriginalVersion && (
              <button
                type="button"
                role="menuitem"
                data-testid={`${testIdPrefix}-mode-option-original`}
                onClick={handleSelectOriginal}
                className={cn(
                  "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-start text-xs transition hover:bg-wc-accent/10",
                  !isSummarized ? "text-wc-accent" : "text-wc-text-muted",
                )}
              >
                <span className="flex h-3 w-3 items-center justify-center">
                  {!isSummarized && <Check className="h-3 w-3" />}
                </span>
                <span className="flex-1">{t(strings.playbackMode.original)}</span>
              </button>
            )}
            {hasOriginalVersion && (
              <div className="my-1 h-px bg-wc-default" />
            )}
            {LEVEL_OPTIONS.map(({ value, labelKey, hintKey }) => {
              const isActive = isSummarized && value === currentLevel;
              return (
                <button
                  key={value}
                  type="button"
                  role="menuitem"
                  data-testid={`${testIdPrefix}-mode-option-${value}`}
                  onClick={() => handleSelectLevel(value)}
                  className={cn(
                    "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-start text-xs transition hover:bg-amber-500/10",
                    isActive ? "text-amber-300" : "text-wc-text-muted",
                  )}
                >
                  <span className="flex h-3 w-3 items-center justify-center">
                    {isActive && <Check className="h-3 w-3" />}
                  </span>
                  <span className="flex-1">{t(labelKey)}</span>
                  <span className="text-[10px] text-wc-text-faint">{t(hintKey)}</span>
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
