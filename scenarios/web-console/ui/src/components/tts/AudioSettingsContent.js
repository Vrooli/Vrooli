import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback } from "react";
import { Volume2, VolumeX } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "../../lib/classnames";
import { strings } from "../../consts/strings";
import { getAccentClasses } from "./scrubStyles";
const SPEED_PRESETS = [0.5, 0.75, 1, 1.25, 1.5, 2];
/**
 * Shared body of the TTS settings popover (desktop) or bottom sheet (mobile).
 * Renders volume slider, mute toggle, and speed presets. Mode/level switching
 * lives in `PlaybackModeControl`, rendered on the bar itself.
 */
export function AudioSettingsContent({ testIdPrefix, volume, isMuted, playbackRate, isSummarized, capabilities, onVolumeChange, onSetMuted, onSetPlaybackRate, }) {
    const { t } = useTranslation();
    const handleVolumeChange = useCallback((e) => {
        // Dragging the slider while muted auto-unmutes. Order matters: clear the
        // mute flag first so the provider doesn't briefly receive the new
        // configured volume while the hook still treats the session as muted.
        if (isMuted && onSetMuted)
            onSetMuted(false);
        onVolumeChange(Number(e.target.value));
    }, [isMuted, onSetMuted, onVolumeChange]);
    const showMuteToggle = isMuted !== undefined && onSetMuted !== undefined;
    return (_jsxs("div", { className: "space-y-3", children: [capabilities.canAdjustVolume && (_jsxs("div", { children: [_jsxs("div", { className: "mb-1.5 flex items-center justify-between", children: [_jsx("label", { className: "block text-[10px] font-medium uppercase tracking-wider text-wc-text-faint", children: t(strings.audioSettings.volume) }), showMuteToggle && onSetMuted && (_jsxs("button", { type: "button", "data-testid": `${testIdPrefix}-mute-toggle`, onClick: () => onSetMuted(!isMuted), className: cn("flex items-center gap-1 rounded px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider transition hover:bg-wc-accent/10", isMuted ? "text-wc-accent" : "text-wc-text-muted"), title: isMuted ? t(strings.audioSettings.unmuteTitle) : t(strings.audioSettings.muteTitle), "aria-pressed": isMuted, "aria-label": isMuted ? t(strings.audioSettings.unmuteAria) : t(strings.audioSettings.muteAria), children: [isMuted ? _jsx(VolumeX, { className: "h-3.5 w-3.5" }) : _jsx(Volume2, { className: "h-3.5 w-3.5" }), _jsx("span", { children: isMuted ? t(strings.audioSettings.muted) : t(strings.audioSettings.mute) })] }))] }), _jsx("input", { "data-testid": `${testIdPrefix}-volume-slider`, type: "range", min: 0, max: 1, step: 0.05, value: volume, onChange: handleVolumeChange, className: cn("h-1.5 w-full cursor-pointer rounded-full", isMuted && "opacity-50", getAccentClasses(isSummarized)) }), _jsxs("div", { className: "mt-0.5 flex justify-between text-[10px] text-wc-text-faint", children: [_jsx("span", { children: "0" }), _jsxs("span", { children: [Math.round(volume * 100), "%"] })] })] })), capabilities.canAdjustSpeed && (_jsxs("div", { className: cn(capabilities.canAdjustVolume && "border-t border-wc-default pt-3"), children: [_jsx("label", { className: "mb-1.5 block text-[10px] font-medium uppercase tracking-wider text-wc-text-faint", children: t(strings.audioSettings.speed) }), _jsx("div", { className: "grid grid-cols-6 gap-1", children: SPEED_PRESETS.map((rate) => {
                            const isActive = rate === playbackRate;
                            return (_jsxs("button", { type: "button", "data-testid": `${testIdPrefix}-speed-preset-${rate}`, onClick: () => onSetPlaybackRate(rate), className: cn("rounded-md px-1 py-1 text-[10px] font-medium tabular-nums transition", isActive
                                    ? "bg-wc-accent/20 text-wc-accent ring-1 ring-wc-accent/40"
                                    : "bg-wc-surface-base text-wc-text-muted hover:bg-wc-surface-input"), children: [rate, "x"] }, rate));
                        }) })] }))] }));
}
