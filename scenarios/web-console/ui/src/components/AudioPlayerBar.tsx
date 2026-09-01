import { useCallback, useEffect, useRef, useState, type ChangeEvent, type KeyboardEvent } from "react";
import { createPortal } from "react-dom";
import { ChevronLeft, ChevronRight, Loader2, Pause, Play, Volume2, VolumeX, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { TTSPlaybackCapabilities } from "../audio-integration";
import type { ConversationEvent } from "../api/conversation";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useAnchoredPopoverPosition, type FloatingPlacement } from "../hooks/useFloatingPosition";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { AudioSettingsContent } from "./tts/AudioSettingsContent";
import { PlaybackModeControl, type SummarizationLevel } from "./tts/PlaybackModeControl";
import { getScrubClasses } from "./tts/scrubStyles";
import MessageJumpList from "./MessageJumpList";
import { IconButton } from "@vrooli/react-component-library/IconButton";

export interface AudioPlayerBarProps {
  isPaused: boolean;
  currentTime: number;
  duration: number | null;
  playbackRate: number;
  volume: number;
  isMuted: boolean;
  capabilities: TTSPlaybackCapabilities;
  /** Whether the currently playing content is a summarized version. */
  isSummarized?: boolean;
  /** Whether the active event has an original (unsummarized) version available. */
  hasOriginalVersion?: boolean;
  /** Whether summarization is available (summarizer configured on backend). */
  canSummarize?: boolean;
  /** Whether a summarization request is in progress. */
  isSummarizing?: boolean;
  /** Whether playback data or synthesized audio is being prepared. */
  isLoading?: boolean;
  /** Current global summarization level. */
  currentLevel?: SummarizationLevel;
  currentMessageLabel?: string | null;
  currentMessageId?: string | null;
  messageSelectorEvents?: ConversationEvent[];
  hasQueuedNext?: boolean;
  hasQueuedPrevious?: boolean;
  onPause: () => void;
  onResume: () => void;
  onSeek: (seconds: number) => void;
  onSetPlaybackRate: (rate: number) => void;
  onSetVolume: (level: number) => void;
  onSetMuted: (next: boolean) => void;
  onJumpToCurrentMessage?: () => void;
  onSelectMessage?: (eventId: string) => void;
  /** Called when the user wants to switch to the original (unsummarized) version. */
  onToggleSummarized?: (useSummarized: boolean) => void;
  /** Called when the user picks a summarization level (may be the current one). */
  onChangeLevel?: (level: SummarizationLevel) => void;
  /** Optional close control for manually-started playback surfaces. */
  onDismiss?: () => void;
  /** Whether the full playback controls are visible. */
  isExpanded?: boolean;
  /** Expand the compact now-playing line. */
  onExpand?: () => void;
  onPreviousMessage?: () => void;
  onNextMessage?: () => void;
  /** Honest explanation when auto playback selected the browser fallback. */
  backendReason?: string;
}

/** Anchored placement order for popovers opening above their trigger. */
const ABOVE_ANCHOR_PLACEMENTS: FloatingPlacement[] = ["top-end", "top-start", "bottom-end", "bottom-start"];

/** Format seconds as m:ss. Returns "--:--" when value is null or not finite. */
function formatTime(seconds: number | null): string {
  if (seconds === null || !Number.isFinite(seconds)) return "--:--";
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
export default function AudioPlayerBar({
  isPaused,
  currentTime,
  duration,
  playbackRate,
  volume,
  isMuted,
  capabilities,
  isSummarized = false,
  hasOriginalVersion = false,
  canSummarize = false,
  isSummarizing = false,
  isLoading = false,
  currentLevel = "moderate",
  currentMessageLabel = null,
  currentMessageId = null,
  messageSelectorEvents,
  hasQueuedNext = false,
  hasQueuedPrevious = false,
  onPause,
  onResume,
  onSeek,
  onSetPlaybackRate,
  onSetVolume,
  onSetMuted,
  onJumpToCurrentMessage,
  onSelectMessage,
  onToggleSummarized,
  onChangeLevel,
  onDismiss,
  isExpanded = true,
  onExpand,
  onPreviousMessage,
  onNextMessage,
  backendReason,
}: AudioPlayerBarProps) {
  const { t } = useTranslation();
  const [showPopover, setShowPopover] = useState(false);
  const [showMessageSelector, setShowMessageSelector] = useState(false);
  const [showRemainingTime, setShowRemainingTime] = useState(() => {
    try {
      return window.localStorage.getItem("vrooli.tts.showRemainingTime") === "true";
    } catch {
      return false;
    }
  });
  const isMobile = useMediaQuery("(max-width: 767px)");
  const audioButtonRef = useRef<HTMLButtonElement>(null);
  const currentMessageButtonRef = useRef<HTMLButtonElement>(null);
  const barRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const wasExpandedRef = useRef(isExpanded);

  const handlePlayPause = useCallback(() => {
    if (isPaused) onResume();
    else onPause();
  }, [isPaused, onPause, onResume]);

  const handleScrubChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      onSeek(Number(e.target.value));
    },
    [onSeek],
  );

  const scrubEnabled = capabilities.canSeek && duration !== null;
  // The bar is "idle" when no audio is loaded — replay mode, between events,
  // or before the first playback poll tick. In this state every control keeps
  // its shape but non-transport controls go visibly disabled to prevent the
  // layout from shifting between playing and idle.
  const isIdle = duration === null;
  const displayedTime = showRemainingTime && duration !== null
    ? Math.max(0, duration - currentTime)
    : currentTime;
  const browserFallbackActive = backendReason?.includes("browser speech synthesis is active")
    || backendReason?.includes("Browser handled playback")
    || false;

  useEffect(() => {
    if (isExpanded && !wasExpandedRef.current) {
      previousFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      requestAnimationFrame(() => {
        const first = barRef.current?.querySelector<HTMLElement>("button:not([disabled]), input:not([disabled])");
        first?.focus();
      });
    } else if (!isExpanded && wasExpandedRef.current) {
      previousFocusRef.current?.focus();
      previousFocusRef.current = null;
    }
    wasExpandedRef.current = isExpanded;
  }, [isExpanded]);

  const handleKeyDown = useCallback((event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Escape" && isExpanded) {
      event.preventDefault();
      onDismiss?.();
      return;
    }
    if (event.key !== "Tab" || !isExpanded || !barRef.current) return;
    const focusable = Array.from(barRef.current.querySelectorAll<HTMLElement>("button:not([disabled]), input:not([disabled])"));
    if (focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (!first || !last) return;
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }, [isExpanded, onDismiss]);

  // Desktop settings popover anchors above the audio button, end-aligned,
  // via the shared anchored-floating math (measure-then-position).
  const audioPopoverRef = useRef<HTMLDivElement>(null);
  const audioPopoverStyle = useAnchoredPopoverPosition(
    showPopover && !isMobile,
    audioButtonRef,
    audioPopoverRef,
    ABOVE_ANCHOR_PLACEMENTS,
  );

  return (
    <div
      ref={barRef}
      data-testid="audio-player-bar"
      data-audio-state="player"
      data-expanded={isExpanded ? "true" : "false"}
      role="region"
      aria-label="Audio playback"
      onKeyDown={handleKeyDown}
      onClick={(event) => {
        if (!isExpanded && !(event.target as HTMLElement).closest('[data-testid="tts-time"], [data-testid="tts-dismiss"]')) onExpand?.();
      }}
      data-loading={isLoading ? "true" : "false"}
      className="flex items-center gap-1.5 border-t border-wc-default bg-wc-surface-raised py-1.5 ps-[max(0.5rem,var(--wc-safe-left,0px))] pe-[max(0.5rem,var(--wc-safe-right,0px))] text-wc-text-primary animate-in slide-in-from-bottom-2 duration-200"
    >
      {browserFallbackActive && (
        <div data-testid="tts-browser-fallback-notice" className="absolute -top-7 start-2 rounded bg-wc-surface-raised px-2 py-1 text-[11px] text-wc-text-muted shadow">
          {backendReason}
        </div>
      )}
      {isExpanded && <PlaybackModeControl
        testIdPrefix="tts"
        isSummarized={isSummarized}
        hasOriginalVersion={hasOriginalVersion}
        canSummarize={canSummarize}
        isSummarizing={isSummarizing}
        currentLevel={currentLevel}
        disabled={isIdle}
        onToggleSummarized={onToggleSummarized}
        onChangeLevel={onChangeLevel}
      />}

      {!isPaused && (
        <span data-testid="tts-equalizer" aria-label="Playing" className="flex h-4 items-end gap-px px-0.5" aria-hidden="true">
          {["h-1.5", "h-3", "h-2", "h-4"].map((height, index) => (
            <span key={index} className={cn("w-0.5 animate-pulse rounded-sm bg-wc-accent motion-reduce:animate-none", height)} />
          ))}
        </span>
      )}

      {isExpanded && onPreviousMessage && (
        <IconButton
          data-testid="tts-previous-message"
          onClick={onPreviousMessage}
          disabled={!hasQueuedPrevious}
          size="sm"
          aria-label="Previous message"
        >
          <ChevronLeft />
        </IconButton>
      )}

      <IconButton
        data-testid="tts-play-pause"
        onClick={handlePlayPause}
        disabled={isLoading || !capabilities.canPause}
        size="sm"
        className={cn("shrink-0", !isLoading && !capabilities.canPause && "cursor-not-allowed")}
        aria-label={isLoading ? t(strings.app.loading) : isPaused ? t(strings.audioPlayerBar.resume) : t(strings.audioPlayerBar.pause)}
      >
        {isLoading
          ? <Loader2 data-testid="tts-playback-loading" className="animate-spin" />
          : isPaused ? <Play /> : <Pause />}
      </IconButton>

      {currentMessageLabel && (
        <button
          ref={currentMessageButtonRef}
          data-testid="tts-current-message"
          type="button"
          onClick={() => {
            if (messageSelectorEvents?.length && onSelectMessage) {
              setShowMessageSelector((prev) => !prev);
              return;
            }
            onJumpToCurrentMessage?.();
          }}
          className="inline-flex shrink-0 items-center gap-1 rounded-md bg-wc-surface-base px-1.5 py-1 text-[11px] font-medium text-wc-text-muted ring-1 ring-wc-default transition hover:bg-wc-surface-input"
          title={messageSelectorEvents?.length && onSelectMessage ? t(strings.audioPlayerBar.selectMessage) : t(strings.audioPlayerBar.jumpToCurrentMessage)}
        >
          <span>{currentMessageLabel}</span>
          {hasQueuedNext && <span className="text-[10px] text-amber-300">{t(strings.audioPlayerBar.nextBadge)}</span>}
          {isLoading && <Loader2 data-testid="tts-message-loading" className="h-3 w-3 animate-spin text-wc-accent" />}
        </button>
      )}

      {showMessageSelector && messageSelectorEvents?.length && onSelectMessage && (
        <MessageJumpList
          events={messageSelectorEvents}
          focusedEventId={currentMessageId}
          mode="playback-select"
          onSelect={(eventId) => {
            onSelectMessage(eventId);
            setShowMessageSelector(false);
          }}
          onClose={() => setShowMessageSelector(false)}
          desktopAnchorRef={currentMessageButtonRef}
          currentTime={currentTime}
          duration={duration}
          isPaused={isPaused}
          isSummarized={isSummarized}
          onPause={onPause}
          onResume={onResume}
          onSeek={onSeek}
          hasQueuedNext={hasQueuedNext}
        />
      )}

      {/* Scrub — always rendered in the same slot so the bar shape is stable
          across playing/idle states. Disabled with min=max=0 when no audio
          is loaded (replay mode, first-tick fallback, non-seekable backends). */}
      {isExpanded && <input
        data-testid="tts-scrub"
        type="range"
        min={0}
        max={scrubEnabled ? (duration as number) : 0}
        value={scrubEnabled ? currentTime : 0}
        step={0.1}
        disabled={!scrubEnabled}
        onChange={handleScrubChange}
        aria-label={t(strings.audioPlayerBar.seekAriaLabel)}
        className={getScrubClasses({
          isSummarized,
          enabled: scrubEnabled,
          extra: "mx-1 flex-1",
        })}
      />}

      <button
        data-testid="tts-time"
        type="button"
        onClick={(event) => {
          event.stopPropagation();
          if (!isExpanded) onExpand?.();
          setShowRemainingTime((previous) => {
            const next = !previous;
            try {
              window.localStorage.setItem("vrooli.tts.showRemainingTime", String(next));
            } catch {
              // Storage is optional (private browsing and embedded surfaces).
            }
            return next;
          });
        }}
        aria-label={showRemainingTime ? "Show elapsed time" : "Show remaining time"}
        title={showRemainingTime ? "Show elapsed time" : "Show remaining time"}
        className="shrink-0 whitespace-nowrap text-center text-[11px] tabular-nums text-wc-text-muted"
      >
        {showRemainingTime ? `-${formatTime(displayedTime)}` : formatTime(displayedTime)} / {formatTime(duration)}
        {hasQueuedNext && currentMessageLabel && <span data-testid="tts-queue-position"> · {currentMessageLabel}</span>}
      </button>

      {/* Audio button — context-sensitive: when muted, single-tap unmutes; when
          unmuted, opens the popover with volume/speed/mute controls. */}
      {isExpanded && capabilities.canAdjustVolume && (
        <button
          ref={audioButtonRef}
          data-testid="tts-audio-button"
          onClick={() => {
            if (isMuted) onSetMuted(false);
            else setShowPopover((prev) => !prev);
          }}
          className="shrink-0 rounded p-1 transition hover:bg-wc-accent/10"
          title={isMuted ? t(strings.audioPlayerBar.unmute) : t(strings.audioPlayerBar.audioSettings)}
        >
          {isMuted ? <VolumeX className="h-4 w-4" /> : <Volume2 className="h-4 w-4" />}
        </button>
      )}

      {isExpanded && onNextMessage && (
        <IconButton
          data-testid="tts-next-message"
          onClick={onNextMessage}
          disabled={!hasQueuedNext}
          size="sm"
          aria-label="Next message"
        >
          <ChevronRight />
        </IconButton>
      )}

      {!isExpanded && onExpand && (
        <IconButton
          data-testid="tts-expand"
          onClick={(event) => { event.stopPropagation(); onExpand(); }}
          size="sm"
          className="shrink-0"
          aria-label={t(strings.audioPlayerBar.audioSettings)}
        >
          <Volume2 />
        </IconButton>
      )}

      {onDismiss && (
        <IconButton
          data-testid="tts-dismiss"
          onClick={onDismiss}
          size="sm"
          className="shrink-0"
          aria-label={t(strings.audioPlayerBar.closePlayback)}
        >
          <X />
        </IconButton>
      )}

      {/* Popover / bottom sheet — always rendered via portal to escape terminal touch handlers */}
      {showPopover && createPortal(
        isMobile ? (
          // Mobile bottom sheet
          <div
            className="fixed inset-0 z-wc-popover-backdrop"
            role="dialog"
            aria-modal="true"
            aria-labelledby="audio-settings-heading"
            onMouseDown={(e) => e.preventDefault()}
          >
            <div
              data-testid="audio-sheet-backdrop"
              aria-hidden="true"
              className="absolute inset-0 bg-wc-backdrop"
              onClick={() => setShowPopover(false)}
            />
            <div
              data-testid="audio-popover"
              className="wc-stable-theme absolute bottom-0 left-0 right-0 z-wc-popover rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised p-4 pb-[max(1rem,var(--wc-safe-bottom))] ps-[max(1rem,var(--wc-safe-left,0px))] pe-[max(1rem,var(--wc-safe-right,0px))] shadow-2xl"
            >
              <div className="mb-3 flex justify-center">
                <div className="h-1 w-8 rounded-full bg-wc-text-muted/40" />
              </div>
              <h3 id="audio-settings-heading" className="mb-3 text-sm font-semibold text-wc-text-primary">{t(strings.audioPlayerBar.audioSettingsHeading)}</h3>
              <AudioSettingsContent
                testIdPrefix="tts"
                volume={volume}
                isMuted={isMuted}
                playbackRate={playbackRate}
                isSummarized={isSummarized}
                capabilities={capabilities}
                onVolumeChange={onSetVolume}
                onSetMuted={onSetMuted}
                onSetPlaybackRate={onSetPlaybackRate}
              />
              {isLoading && (
                <div data-testid="tts-audio-loading" className="mt-3 flex items-center gap-2 rounded-lg bg-wc-surface-base px-3 py-2 text-xs text-wc-text-muted">
                  <Loader2 className="h-3.5 w-3.5 animate-spin text-wc-accent" />
                  <span>{t(strings.app.loading)}</span>
                </div>
              )}
            </div>
          </div>
        ) : (
          <>
            <div
              data-testid="audio-popover-backdrop"
              className="fixed inset-0 z-wc-popover-backdrop"
              onClick={() => setShowPopover(false)}
            />
            <div
              ref={audioPopoverRef}
              data-testid="audio-popover"
              className="wc-stable-theme z-wc-popover w-60 rounded-xl border border-wc-default bg-wc-surface-raised p-3 shadow-lg"
              style={audioPopoverStyle}
            >
              <AudioSettingsContent
                testIdPrefix="tts"
                volume={volume}
                isMuted={isMuted}
                playbackRate={playbackRate}
                isSummarized={isSummarized}
                capabilities={capabilities}
                onVolumeChange={onSetVolume}
                onSetMuted={onSetMuted}
                onSetPlaybackRate={onSetPlaybackRate}
              />
              {isLoading && (
                <div data-testid="tts-audio-loading" className="mt-3 flex items-center gap-2 rounded-lg bg-wc-surface-base px-3 py-2 text-xs text-wc-text-muted">
                  <Loader2 className="h-3.5 w-3.5 animate-spin text-wc-accent" />
                  <span>{t(strings.app.loading)}</span>
                </div>
              )}
            </div>
          </>
        ),
        document.body,
      )}
    </div>
  );
}
