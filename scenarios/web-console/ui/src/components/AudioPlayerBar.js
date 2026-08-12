import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useCallback, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Loader2, Pause, Play, Volume2, VolumeX, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useAnchoredPopoverPosition } from "../hooks/useFloatingPosition";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { AudioSettingsContent } from "./tts/AudioSettingsContent";
import { PlaybackModeControl } from "./tts/PlaybackModeControl";
import { getScrubClasses } from "./tts/scrubStyles";
import MessageJumpList from "./MessageJumpList";
/** Anchored placement order for popovers opening above their trigger. */
const ABOVE_ANCHOR_PLACEMENTS = ["top-end", "top-start", "bottom-end", "bottom-start"];
/** Format seconds as m:ss. Returns "--:--" when value is null or not finite. */
function formatTime(seconds) {
    if (seconds === null || !Number.isFinite(seconds))
        return "--:--";
    const totalSeconds = Math.round(seconds);
    const m = Math.floor(totalSeconds / 60);
    const s = totalSeconds % 60;
    return `${m}:${String(s).padStart(2, "0")}`;
}
/**
 * Global bottom bar for TTS playback controls.
 *
 * Renders pause/resume, stop, scrub bar, time display, mode control, and an
 * audio button that opens a popover (desktop) or bottom sheet (mobile) with
 * volume + speed settings.
 */
export default function AudioPlayerBar({ isPaused, currentTime, duration, playbackRate, volume, isMuted, capabilities, isSummarized = false, hasOriginalVersion = false, canSummarize = false, isSummarizing = false, isLoading = false, currentLevel = "moderate", currentMessageLabel = null, currentMessageId = null, messageSelectorEvents, hasQueuedNext = false, onPause, onResume, onSeek, onSetPlaybackRate, onSetVolume, onSetMuted, onJumpToCurrentMessage, onSelectMessage, onToggleSummarized, onChangeLevel, onDismiss, }) {
    const { t } = useTranslation();
    const [showPopover, setShowPopover] = useState(false);
    const [showMessageSelector, setShowMessageSelector] = useState(false);
    const isMobile = useMediaQuery("(max-width: 767px)");
    const audioButtonRef = useRef(null);
    const currentMessageButtonRef = useRef(null);
    const handlePlayPause = useCallback(() => {
        if (isPaused)
            onResume();
        else
            onPause();
    }, [isPaused, onPause, onResume]);
    const handleScrubChange = useCallback((e) => {
        onSeek(Number(e.target.value));
    }, [onSeek]);
    const scrubEnabled = capabilities.canSeek && duration !== null;
    // The bar is "idle" when no audio is loaded — replay mode, between events,
    // or before the first playback poll tick. In this state every control keeps
    // its shape but non-transport controls go visibly disabled to prevent the
    // layout from shifting between playing and idle.
    const isIdle = duration === null;
    // Desktop settings popover anchors above the audio button, end-aligned,
    // via the shared anchored-floating math (measure-then-position).
    const audioPopoverRef = useRef(null);
    const audioPopoverStyle = useAnchoredPopoverPosition(showPopover && !isMobile, audioButtonRef, audioPopoverRef, ABOVE_ANCHOR_PLACEMENTS);
    return (_jsxs("div", { "data-testid": "audio-player-bar", "data-audio-state": "player", "data-loading": isLoading ? "true" : "false", className: "flex items-center gap-1.5 border-t border-wc-default bg-wc-surface-raised py-1.5 ps-[max(0.5rem,var(--wc-safe-left,0px))] pe-[max(0.5rem,var(--wc-safe-right,0px))] text-wc-text-primary animate-in slide-in-from-bottom-2 duration-200", children: [_jsx(PlaybackModeControl, { testIdPrefix: "tts", isSummarized: isSummarized, hasOriginalVersion: hasOriginalVersion, canSummarize: canSummarize, isSummarizing: isSummarizing, currentLevel: currentLevel, disabled: isIdle, onToggleSummarized: onToggleSummarized, onChangeLevel: onChangeLevel }), _jsx("button", { "data-testid": "tts-play-pause", onClick: handlePlayPause, disabled: isLoading || !capabilities.canPause, className: cn("shrink-0 rounded p-1 transition hover:bg-wc-accent/10", isLoading && "cursor-wait text-wc-accent", (isLoading || !capabilities.canPause) && "opacity-60", !isLoading && !capabilities.canPause && "cursor-not-allowed"), title: isLoading ? t(strings.app.loading) : isPaused ? t(strings.audioPlayerBar.resume) : t(strings.audioPlayerBar.pause), children: isLoading
                    ? _jsx(Loader2, { "data-testid": "tts-playback-loading", className: "h-4 w-4 animate-spin" })
                    : isPaused ? _jsx(Play, { className: "h-4 w-4" }) : _jsx(Pause, { className: "h-4 w-4" }) }), currentMessageLabel && (_jsxs("button", { ref: currentMessageButtonRef, "data-testid": "tts-current-message", type: "button", onClick: () => {
                    if (messageSelectorEvents?.length && onSelectMessage) {
                        setShowMessageSelector((prev) => !prev);
                        return;
                    }
                    onJumpToCurrentMessage?.();
                }, className: "inline-flex shrink-0 items-center gap-1 rounded-md bg-wc-surface-base px-1.5 py-1 text-[11px] font-medium text-wc-text-muted ring-1 ring-wc-default transition hover:bg-wc-surface-input", title: messageSelectorEvents?.length && onSelectMessage ? t(strings.audioPlayerBar.selectMessage) : t(strings.audioPlayerBar.jumpToCurrentMessage), children: [_jsx("span", { children: currentMessageLabel }), hasQueuedNext && _jsx("span", { className: "text-[10px] text-amber-300", children: t(strings.audioPlayerBar.nextBadge) }), isLoading && _jsx(Loader2, { "data-testid": "tts-message-loading", className: "h-3 w-3 animate-spin text-wc-accent" })] })), showMessageSelector && messageSelectorEvents?.length && onSelectMessage && (_jsx(MessageJumpList, { events: messageSelectorEvents, focusedEventId: currentMessageId, mode: "playback-select", onSelect: (eventId) => {
                    onSelectMessage(eventId);
                    setShowMessageSelector(false);
                }, onClose: () => setShowMessageSelector(false), desktopAnchorRef: currentMessageButtonRef, currentTime: currentTime, duration: duration, isPaused: isPaused, isSummarized: isSummarized, onPause: onPause, onResume: onResume, onSeek: onSeek, hasQueuedNext: hasQueuedNext })), _jsx("input", { "data-testid": "tts-scrub", type: "range", min: 0, max: scrubEnabled ? duration : 0, value: scrubEnabled ? currentTime : 0, step: 0.1, disabled: !scrubEnabled, onChange: handleScrubChange, "aria-label": t(strings.audioPlayerBar.seekAriaLabel), className: getScrubClasses({
                    isSummarized,
                    enabled: scrubEnabled,
                    extra: "mx-1 flex-1",
                }) }), _jsxs("span", { "data-testid": "tts-time", className: "shrink-0 whitespace-nowrap text-center text-[11px] tabular-nums text-wc-text-muted", children: [formatTime(currentTime), " / ", formatTime(duration)] }), capabilities.canAdjustVolume && (_jsx("button", { ref: audioButtonRef, "data-testid": "tts-audio-button", onClick: () => {
                    if (isMuted)
                        onSetMuted(false);
                    else
                        setShowPopover((prev) => !prev);
                }, className: "shrink-0 rounded p-1 transition hover:bg-wc-accent/10", title: isMuted ? t(strings.audioPlayerBar.unmute) : t(strings.audioPlayerBar.audioSettings), children: isMuted ? _jsx(VolumeX, { className: "h-4 w-4" }) : _jsx(Volume2, { className: "h-4 w-4" }) })), onDismiss && (_jsx("button", { "data-testid": "tts-dismiss", type: "button", onClick: onDismiss, className: "shrink-0 rounded p-1 text-wc-text-muted transition hover:bg-wc-accent/10 hover:text-wc-text-primary", title: t(strings.audioPlayerBar.closePlayback), "aria-label": t(strings.audioPlayerBar.closePlayback), children: _jsx(X, { className: "h-4 w-4" }) })), showPopover && createPortal(isMobile ? (
            // Mobile bottom sheet
            _jsxs("div", { className: "fixed inset-0 z-wc-popover-backdrop", onMouseDown: (e) => e.preventDefault(), children: [_jsx("div", { "data-testid": "audio-sheet-backdrop", className: "absolute inset-0 bg-wc-backdrop", onClick: () => setShowPopover(false) }), _jsxs("div", { "data-testid": "audio-popover", className: "wc-stable-theme absolute bottom-0 left-0 right-0 z-wc-popover rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised p-4 pb-[max(1rem,var(--wc-safe-bottom))] ps-[max(1rem,var(--wc-safe-left,0px))] pe-[max(1rem,var(--wc-safe-right,0px))] shadow-2xl", children: [_jsx("div", { className: "mb-3 flex justify-center", children: _jsx("div", { className: "h-1 w-8 rounded-full bg-wc-text-muted/40" }) }), _jsx("h3", { className: "mb-3 text-sm font-semibold text-wc-text-primary", children: t(strings.audioPlayerBar.audioSettingsHeading) }), _jsx(AudioSettingsContent, { testIdPrefix: "tts", volume: volume, isMuted: isMuted, playbackRate: playbackRate, isSummarized: isSummarized, capabilities: capabilities, onVolumeChange: onSetVolume, onSetMuted: onSetMuted, onSetPlaybackRate: onSetPlaybackRate }), isLoading && (_jsxs("div", { "data-testid": "tts-audio-loading", className: "mt-3 flex items-center gap-2 rounded-lg bg-wc-surface-base px-3 py-2 text-xs text-wc-text-muted", children: [_jsx(Loader2, { className: "h-3.5 w-3.5 animate-spin text-wc-accent" }), _jsx("span", { children: t(strings.app.loading) })] }))] })] })) : (_jsxs(_Fragment, { children: [_jsx("div", { "data-testid": "audio-popover-backdrop", className: "fixed inset-0 z-wc-popover-backdrop", onClick: () => setShowPopover(false) }), _jsxs("div", { ref: audioPopoverRef, "data-testid": "audio-popover", className: "wc-stable-theme z-wc-popover w-60 rounded-xl border border-wc-default bg-wc-surface-raised p-3 shadow-lg", style: audioPopoverStyle, children: [_jsx(AudioSettingsContent, { testIdPrefix: "tts", volume: volume, isMuted: isMuted, playbackRate: playbackRate, isSummarized: isSummarized, capabilities: capabilities, onVolumeChange: onSetVolume, onSetMuted: onSetMuted, onSetPlaybackRate: onSetPlaybackRate }), isLoading && (_jsxs("div", { "data-testid": "tts-audio-loading", className: "mt-3 flex items-center gap-2 rounded-lg bg-wc-surface-base px-3 py-2 text-xs text-wc-text-muted", children: [_jsx(Loader2, { className: "h-3.5 w-3.5 animate-spin text-wc-accent" }), _jsx("span", { children: t(strings.app.loading) })] }))] })] })), document.body)] }));
}
