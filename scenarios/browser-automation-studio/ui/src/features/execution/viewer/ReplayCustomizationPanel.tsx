import { ChevronDown, Check } from "lucide-react";
import clsx from "clsx";
import {
  BACKGROUND_GROUP_ORDER,
  CURSOR_GROUP_ORDER,
  REPLAY_CHROME_OPTIONS,
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS,
  REPLAY_CURSOR_POSITIONS,
  CURSOR_SCALE_MIN,
  CURSOR_SCALE_MAX,
} from "../replay/replayThemeOptions";
import type { ReplayCustomizationController } from "./useReplayCustomization";

interface ReplayCustomizationPanelProps {
  controller: ReplayCustomizationController;
}

/**
 * ReplayCustomizationPanel renders the theming + cursor controls for the replay tab.
 * All state is managed by useReplayCustomization; this component is purely presentational.
 */
export function ReplayCustomizationPanel({ controller }: ReplayCustomizationPanelProps) {
  const {
    selectedChromeOption,
    selectedBackgroundOption,
    selectedCursorOption,
    selectedCursorPositionOption,
    selectedCursorClickAnimationOption,
    backgroundOptionsByGroup,
    cursorOptionsByGroup,
    replayChromeTheme,
    replayBackgroundTheme,
    replayCursorTheme,
    replayCursorInitialPosition,
    replayCursorClickAnimation,
    replayCursorScale,
    setReplayChromeTheme,
    setIsCustomizationCollapsed,
    isCustomizationCollapsed,
    isBackgroundMenuOpen,
    setIsBackgroundMenuOpen,
    isCursorMenuOpen,
    setIsCursorMenuOpen,
    isCursorPositionMenuOpen,
    setIsCursorPositionMenuOpen,
    isCursorClickAnimationMenuOpen,
    setIsCursorClickAnimationMenuOpen,
    backgroundSelectorRef,
    cursorSelectorRef,
    cursorPositionSelectorRef,
    cursorClickAnimationSelectorRef,
    handleBackgroundSelect,
    handleCursorThemeSelect,
    handleCursorPositionSelect,
    handleCursorClickAnimationSelect,
    handleCursorScaleChange,
  } = controller;

  return (
    <div className="rounded-2xl border border-white/5 bg-slate-950/40 px-4 py-3 shadow-inner">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
            Replay customization
          </span>
          <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-slate-500">
            <span>Chrome • {selectedChromeOption.label}</span>
            <span>Background • {selectedBackgroundOption.label}</span>
            <span>Cursor • {selectedCursorOption.label}</span>
            <span>Click • {selectedCursorClickAnimationOption.label}</span>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setIsCustomizationCollapsed(!isCustomizationCollapsed)}
          className="inline-flex h-8 w-8 items-center justify-center rounded-full border border-white/10 bg-white/5 text-slate-200 transition-colors transition-transform hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-900"
          aria-expanded={!isCustomizationCollapsed}
          aria-label={
            isCustomizationCollapsed
              ? "Expand replay customization"
              : "Collapse replay customization"
          }
        >
          <ChevronDown
            size={16}
            className={clsx("transition-transform duration-200", {
              "-rotate-180": !isCustomizationCollapsed,
            })}
          />
        </button>
      </div>
      {!isCustomizationCollapsed && (
        <div className="mt-4 space-y-4">
          <div>
            <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
              <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                Browser frame
              </span>
              <span className="text-[11px] text-slate-500">
                Customize the replay window
              </span>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {REPLAY_CHROME_OPTIONS.map((option) => (
                <button
                  key={option.id}
                  type="button"
                  onClick={() => setReplayChromeTheme(option.id)}
                  title={option.subtitle}
                  className={clsx(
                    "rounded-full px-3 py-1.5 text-xs font-medium transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                    replayChromeTheme === option.id
                      ? "bg-flow-accent text-white shadow-[0_12px_35px_rgba(59,130,246,0.45)]"
                      : "bg-slate-900/60 text-slate-300 hover:bg-slate-900/80",
                  )}
                >
                  {option.label}
                </button>
              ))}
            </div>
          </div>

          <div className="border-t border-white/5 pt-3 space-y-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                Background
              </span>
              <span className="text-[11px] text-slate-500">
                Set the stage behind the browser
              </span>
            </div>
            <div ref={backgroundSelectorRef} className="relative">
              <button
                type="button"
                className={clsx(
                  "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                  isBackgroundMenuOpen
                    ? "border-flow-accent/70 bg-slate-900/80 text-white"
                    : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                )}
                onClick={() => {
                  setIsBackgroundMenuOpen(!isBackgroundMenuOpen);
                  setIsCursorMenuOpen(false);
                  setIsCursorPositionMenuOpen(false);
                  setIsCursorClickAnimationMenuOpen(false);
                }}
                aria-haspopup="menu"
                aria-expanded={isBackgroundMenuOpen}
              >
                <span className="flex items-center gap-3">
                  <span
                    aria-hidden
                    className="relative h-10 w-16 overflow-hidden rounded-lg border border-white/10 shadow-inner"
                    style={selectedBackgroundOption.previewStyle}
                  >
                    {selectedBackgroundOption.previewNode}
                  </span>
                  <span className="flex flex-col text-xs leading-tight text-slate-300">
                    <span className="text-sm font-medium text-white">
                      {selectedBackgroundOption.label}
                    </span>
                    <span className="text-[11px] text-slate-400">
                      {selectedBackgroundOption.subtitle}
                    </span>
                  </span>
                </span>
                <ChevronDown
                  size={14}
                  className={clsx(
                    "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                    isBackgroundMenuOpen ? "rotate-180 text-white" : "",
                  )}
                />
              </button>

              {isBackgroundMenuOpen && (
                <div
                  role="menu"
                  className="absolute right-0 z-30 mt-2 w-full min-w-[260px] rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-2xl backdrop-blur-md sm:w-80"
                >
                  {BACKGROUND_GROUP_ORDER.map((group) => {
                    const options = backgroundOptionsByGroup[group.id];
                    if (!options || options.length === 0) {
                      return null;
                    }
                    return (
                      <div key={group.id} className="py-1">
                        <div className="px-2 pb-1 text-[10px] uppercase tracking-[0.24em] text-slate-500">
                          {group.label}
                        </div>
                        <div className="space-y-1">
                          {options.map((option) => {
                            const isActive = replayBackgroundTheme === option.id;
                            return (
                              <button
                                key={option.id}
                                type="button"
                                role="menuitemradio"
                                aria-checked={isActive}
                                onClick={() => handleBackgroundSelect(option)}
                                className={clsx(
                                  "flex w-full items-center gap-3 rounded-lg border px-2.5 py-2 text-left transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                                  isActive
                                    ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]"
                                    : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                                )}
                              >
                                <span
                                  className="relative h-10 w-16 flex-shrink-0 overflow-hidden rounded-lg border border-white/10 shadow-inner"
                                  style={option.previewStyle}
                                >
                                  {option.previewNode}
                                </span>
                                <span className="flex flex-1 flex-col text-xs text-slate-300">
                                  <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                    <span>{option.label}</span>
                                    {isActive && <Check size={14} className="text-flow-accent" />}
                                  </span>
                                  <span className="text-[11px] text-slate-400">
                                    {option.subtitle}
                                  </span>
                                </span>
                              </button>
                            );
                          })}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>

          <div className="border-t border-white/5 pt-3 space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                Cursor
              </span>
              <span className="text-[11px] text-slate-500">
                Style the virtual pointer overlay
              </span>
            </div>
            <div className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-start lg:justify-between">
              <div ref={cursorSelectorRef} className="relative flex-1 min-w-[220px]">
                <button
                  type="button"
                  className={clsx(
                    "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                    isCursorMenuOpen
                      ? "border-flow-accent/70 bg-slate-900/80 text-white"
                      : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                  )}
                  onClick={() => {
                    setIsCursorMenuOpen(!isCursorMenuOpen);
                    setIsBackgroundMenuOpen(false);
                    setIsCursorPositionMenuOpen(false);
                    setIsCursorClickAnimationMenuOpen(false);
                  }}
                  aria-haspopup="menu"
                  aria-expanded={isCursorMenuOpen}
                >
                  <span className="flex items-center gap-3">
                    <span className="relative flex h-10 w-12 items-center justify-center overflow-hidden rounded-lg border border-white/10 bg-slate-900/60">
                      {selectedCursorOption.preview}
                    </span>
                    <span className="flex flex-col text-xs leading-tight text-slate-300">
                      <span className="text-sm font-medium text-white">
                        {selectedCursorOption.label}
                      </span>
                      <span className="text-[11px] text-slate-400">
                        {selectedCursorOption.subtitle}
                      </span>
                    </span>
                  </span>
                  <ChevronDown
                    size={14}
                    className={clsx(
                      "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                      isCursorMenuOpen ? "rotate-180 text-white" : "",
                    )}
                  />
                </button>
                {isCursorMenuOpen && (
                  <div
                    role="menu"
                    className="absolute right-0 z-30 mt-2 w-full min-w-[240px] rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_20px_50px_rgba(15,23,42,0.55)] backdrop-blur"
                  >
                    {CURSOR_GROUP_ORDER.map((group) => {
                      const options = cursorOptionsByGroup[group.id];
                      if (!options || options.length === 0) {
                        return null;
                      }
                      return (
                        <div key={group.id} className="mb-2 last:mb-0">
                          <div className="px-2 pb-1 text-[10px] uppercase tracking-[0.24em] text-slate-500">
                            {group.label}
                          </div>
                          <div className="space-y-1">
                            {options.map((option) => {
                              const isActive = replayCursorTheme === option.id;
                              return (
                                <button
                                  key={option.id}
                                  type="button"
                                  role="menuitemradio"
                                  aria-checked={isActive}
                                  onClick={() => handleCursorThemeSelect(option.id)}
                                  className={clsx(
                                    "flex w-full items-center gap-3 rounded-lg border px-2.5 py-2 text-left transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                                    isActive
                                      ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]"
                                      : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                                  )}
                                >
                                  <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-900/70">
                                    {option.preview}
                                  </span>
                                  <span className="flex flex-1 flex-col text-xs text-slate-300">
                                    <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                      <span>{option.label}</span>
                                      {isActive && <Check size={14} className="text-flow-accent" />}
                                    </span>
                                    <span className="text-[11px] text-slate-400">
                                      {option.subtitle}
                                    </span>
                                  </span>
                                </button>
                              );
                            })}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
              <div ref={cursorPositionSelectorRef} className="relative flex flex-1 flex-col gap-2 lg:max-w-xs">
                <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">
                  Initial placement
                </span>
                <button
                  type="button"
                  className={clsx(
                    "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                    isCursorPositionMenuOpen
                      ? "border-flow-accent/70 bg-slate-900/80 text-white"
                      : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                  )}
                  onClick={() => {
                    setIsCursorPositionMenuOpen(!isCursorPositionMenuOpen);
                    setIsBackgroundMenuOpen(false);
                    setIsCursorMenuOpen(false);
                    setIsCursorClickAnimationMenuOpen(false);
                  }}
                  aria-haspopup="menu"
                  aria-expanded={isCursorPositionMenuOpen}
                >
                  <span className="flex flex-col text-xs leading-tight text-slate-300">
                    <span className="text-sm font-medium text-white">
                      {selectedCursorPositionOption.label}
                    </span>
                    <span className="text-[11px] text-slate-400">
                      {selectedCursorPositionOption.subtitle}
                    </span>
                  </span>
                  <ChevronDown
                    size={14}
                    className={clsx(
                      "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                      isCursorPositionMenuOpen ? "rotate-180 text-white" : "",
                    )}
                  />
                </button>
                {isCursorPositionMenuOpen && (
                  <div
                    role="menu"
                    className="absolute right-0 z-30 mt-2 w-full rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_18px_45px_rgba(15,23,42,0.55)] backdrop-blur"
                  >
                    <div className="space-y-1">
                      {REPLAY_CURSOR_POSITIONS.map((option) => {
                        const isActive = replayCursorInitialPosition === option.id;
                        return (
                          <button
                            key={option.id}
                            type="button"
                            role="menuitemradio"
                            aria-checked={isActive}
                            onClick={() => handleCursorPositionSelect(option.id)}
                            className={clsx(
                              "w-full rounded-lg border px-2.5 py-2 text-left text-xs transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                              isActive
                                ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]"
                                : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                            )}
                          >
                            <span className="flex flex-col text-slate-300">
                              <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                <span>{option.label}</span>
                                {isActive && <Check size={14} className="text-flow-accent" />}
                              </span>
                              <span className="text-[11px] text-slate-400">
                                {option.subtitle}
                              </span>
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  </div>
                )}
              </div>
              <div ref={cursorClickAnimationSelectorRef} className="relative flex flex-1 flex-col gap-2 lg:max-w-xs">
                <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">
                  Click animation
                </span>
                <button
                  type="button"
                  className={clsx(
                    "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                    isCursorClickAnimationMenuOpen
                      ? "border-flow-accent/70 bg-slate-900/80 text-white"
                      : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                  )}
                  onClick={() => {
                    setIsCursorClickAnimationMenuOpen(!isCursorClickAnimationMenuOpen);
                    setIsBackgroundMenuOpen(false);
                    setIsCursorMenuOpen(false);
                    setIsCursorPositionMenuOpen(false);
                  }}
                  aria-haspopup="menu"
                  aria-expanded={isCursorClickAnimationMenuOpen}
                >
                  <span className="flex flex-col text-xs leading-tight text-slate-300">
                    <span className="text-sm font-medium text-white">
                      {selectedCursorClickAnimationOption.label}
                    </span>
                    <span className="text-[11px] text-slate-400">
                      {selectedCursorClickAnimationOption.subtitle}
                    </span>
                  </span>
                  <ChevronDown
                    size={14}
                    className={clsx(
                      "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                      isCursorClickAnimationMenuOpen ? "rotate-180 text-white" : "",
                    )}
                  />
                </button>
                {isCursorClickAnimationMenuOpen && (
                  <div
                    role="menu"
                    className="absolute right-0 z-30 mt-2 w-full rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_18px_45px_rgba(15,23,42,0.55)] backdrop-blur"
                  >
                    <div className="space-y-1">
                      {REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.map((option) => {
                        const isActive = replayCursorClickAnimation === option.id;
                        return (
                          <button
                            key={option.id}
                            type="button"
                            role="menuitemradio"
                            aria-checked={isActive}
                            onClick={() => handleCursorClickAnimationSelect(option.id)}
                            className={clsx(
                              "w-full rounded-lg border px-2.5 py-2 text-left text-xs transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                              isActive
                                ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]"
                                : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                            )}
                          >
                            <span className="flex flex-col text-slate-300">
                              <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                <span>{option.label}</span>
                                {isActive && <Check size={14} className="text-flow-accent" />}
                              </span>
                              <span className="text-[11px] text-slate-400">
                                {option.subtitle}
                              </span>
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  </div>
                )}
              </div>
              <div className="flex flex-1 flex-col gap-2 lg:max-w-[260px]">
                <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">
                  Cursor scale
                </span>
                <input
                  type="range"
                  min={CURSOR_SCALE_MIN}
                  max={CURSOR_SCALE_MAX}
                  step={0.05}
                  value={replayCursorScale}
                  onChange={(event) => handleCursorScaleChange(Number(event.target.value))}
                  className="w-full accent-flow-accent"
                />
                <div className="flex items-center justify-between text-xs text-slate-400">
                  <span>Precision</span>
                  <span>Emphasis</span>
                </div>
                <div className="rounded-lg border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-slate-200">
                  Scale: <span className="font-semibold text-white">{Math.round(replayCursorScale * 100)}%</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default ReplayCustomizationPanel;
