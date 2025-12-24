import { ChevronDown } from "lucide-react";
import clsx from "clsx";
import {
  ReplayBackgroundSettings,
  ReplayChromeSettings,
  ReplayCursorSettings,
  ReplayDeviceFrameSettings,
  ReplayPresentationModeSettings,
  getReplayChromeOption,
  getReplayCursorClickAnimationOption,
  getReplayCursorOption,
  getReplayDeviceFrameOption,
} from "@/domains/replay-style";
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
    replayChromeTheme,
    replayPresentation,
    replayBackground,
    replayCursorTheme,
    replayCursorInitialPosition,
    replayCursorClickAnimation,
    replayCursorScale,
    replayBrowserScale,
    replayDeviceFrameTheme,
    setReplayChromeTheme,
    setReplayPresentation,
    setReplayBackground,
    setReplayCursorTheme,
    setReplayCursorInitialPosition,
    setReplayCursorClickAnimation,
    setReplayCursorScale,
    setReplayDeviceFrameTheme,
    setIsCustomizationCollapsed,
    isCustomizationCollapsed,
    setReplayBrowserScale,
  } = controller;

  const chromeOption = getReplayChromeOption(replayChromeTheme);
  const deviceFrameOption = getReplayDeviceFrameOption(replayDeviceFrameTheme);
  const cursorOption = getReplayCursorOption(replayCursorTheme);
  const clickAnimationOption = getReplayCursorClickAnimationOption(replayCursorClickAnimation);

  const modeSummary = [
    replayPresentation.showDesktop ? "Desktop" : null,
    replayPresentation.showBrowserFrame ? "Browser" : null,
    replayPresentation.showDesktop && replayPresentation.showDeviceFrame ? "Device Frame" : null,
  ]
    .filter(Boolean)
    .join(" + ") || "Content only";

  const backgroundSummary =
    !replayPresentation.showDesktop
      ? "Off"
      : replayBackground.type === "theme" && replayBackground.id === "none"
      ? "Disabled"
      : replayBackground.type === "image"
        ? "Image"
        : replayBackground.type === "gradient"
          ? "Custom gradient"
          : "Theme";
  const chromeSummary = replayPresentation.showBrowserFrame ? chromeOption.label : "Off";
  const deviceFrameSummary =
    replayPresentation.showDesktop && replayPresentation.showDeviceFrame
      ? deviceFrameOption.label
      : "Off";

  return (
    <div className="rounded-2xl border border-white/5 bg-slate-950/40 px-4 py-3 shadow-inner">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
            Replay customization
          </span>
          <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-slate-500">
            <span>Mode • {modeSummary}</span>
            <span>Chrome • {chromeSummary}</span>
            <span>Background • {backgroundSummary}</span>
            <span>Device • {deviceFrameSummary}</span>
            <span>Cursor • {cursorOption.label}</span>
            <span>Click • {clickAnimationOption.label}</span>
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
                Replay mode
              </span>
              <span className="text-[11px] text-slate-500">
                Choose how the replay is framed
              </span>
            </div>
            <ReplayPresentationModeSettings
              presentation={replayPresentation}
              onChange={setReplayPresentation}
              variant="compact"
            />
          </div>
          {replayPresentation.showBrowserFrame && (
            <div>
              <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                  Browser frame
                </span>
                <span className="text-[11px] text-slate-500">
                  Customize the replay window
                </span>
              </div>
              <ReplayChromeSettings
                chromeTheme={replayChromeTheme}
                browserScale={replayBrowserScale}
                onChromeThemeChange={setReplayChromeTheme}
                onBrowserScaleChange={setReplayBrowserScale}
                variant="compact"
              />
            </div>
          )}

          {replayPresentation.showDesktop && (
            <div className="border-t border-white/5 pt-3 space-y-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                  Background
                </span>
                <span className="text-[11px] text-slate-500">
                  Set the stage behind the browser
                </span>
              </div>
              <ReplayBackgroundSettings
                background={replayBackground}
                onBackgroundChange={setReplayBackground}
                variant="compact"
              />
            </div>
          )}

          {replayPresentation.showDesktop && replayPresentation.showDeviceFrame && (
            <div className="border-t border-white/5 pt-3 space-y-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                  Device frame
                </span>
                <span className="text-[11px] text-slate-500">
                  Subtle bezel around the stage
                </span>
              </div>
              <ReplayDeviceFrameSettings
                deviceFrameTheme={replayDeviceFrameTheme}
                onDeviceFrameThemeChange={setReplayDeviceFrameTheme}
                variant="compact"
              />
            </div>
          )}
          <div className="border-t border-white/5 pt-3 space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                Cursor
              </span>
              <span className="text-[11px] text-slate-500">
                Style the virtual pointer overlay
              </span>
            </div>
            <ReplayCursorSettings
              cursorTheme={replayCursorTheme}
              cursorInitialPosition={replayCursorInitialPosition}
              cursorClickAnimation={replayCursorClickAnimation}
              cursorScale={replayCursorScale}
              onCursorThemeChange={setReplayCursorTheme}
              onCursorInitialPositionChange={setReplayCursorInitialPosition}
              onCursorClickAnimationChange={setReplayCursorClickAnimation}
              onCursorScaleChange={setReplayCursorScale}
              variant="compact"
            />
          </div>
        </div>
      )}

    </div>
  );
}

export default ReplayCustomizationPanel;
