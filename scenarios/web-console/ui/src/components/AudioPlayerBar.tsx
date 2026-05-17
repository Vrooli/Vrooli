import { useCallback, useRef, useState, type CSSProperties, type ChangeEvent } from "react";
import { createPortal } from "react-dom";
import { Loader2, Pause, Play, Volume2, VolumeX, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { TTSPlaybackCapabilities } from "../audio-integration";
import type { ConversationEvent } from "../api/conversation";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { AudioSettingsContent } from "./tts/AudioSettingsContent";
import { PlaybackModeControl, type SummarizationLevel } from "./tts/PlaybackModeControl";
import { getScrubClasses } from "./tts/scrubStyles";
import MessageJumpList from "./MessageJumpList";

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
}

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
}: AudioPlayerBarProps) {
  const { t } = useTranslation();
  const [showPopover, setShowPopover] = useState(false);
  const [showMessageSelector, setShowMessageSelector] = useState(false);
  const isMobile = useMediaQuery("(max-width: 767px)");
  const audioButtonRef = useRef<HTMLButtonElement>(null);
  const currentMessageButtonRef = useRef<HTMLButtonElement>(null);

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

  // Compute popover position anchored above the audio button
  const getPopoverStyle = useCallback((): CSSProperties => {
    const btn = audioButtonRef.current;
    if (!btn) return { position: "fixed", bottom: 48, right: 16 };
    const rect = btn.getBoundingClientRect();
    return {
      position: "fixed",
      bottom: window.innerHeight - rect.top + 8,
      right: Math.max(8, window.innerWidth - rect.right),
    };
  }, []);

  const getMessageSelectorStyle = useCallback((): CSSProperties => {
    const btn = currentMessageButtonRef.current;
    if (!btn) return { bottom: 48, right: 16 };
    const rect = btn.getBoundingClientRect();
    return {
      bottom: window.innerHeight - rect.top + 8,
      right: Math.max(8, window.innerWidth - rect.right),
    };
  }, []);

  return (
    <div
      data-testid="audio-player-bar"
      data-loading={isLoading ? "true" : "false"}
      className="flex items-center gap-1.5 border-t border-wc-default bg-wc-surface-raised py-1.5 ps-[max(0.5rem,var(--wc-safe-left,0px))] pe-[max(0.5rem,var(--wc-safe-right,0px))] text-wc-text-primary animate-in slide-in-from-bottom-2 duration-200"
    >
      <PlaybackModeControl
        testIdPrefix="tts"
        isSummarized={isSummarized}
        hasOriginalVersion={hasOriginalVersion}
        canSummarize={canSummarize}
        isSummarizing={isSummarizing}
        currentLevel={currentLevel}
        disabled={isIdle}
        onToggleSummarized={onToggleSummarized}
        onChangeLevel={onChangeLevel}
      />

      <button
        data-testid="tts-play-pause"
        onClick={handlePlayPause}
        disabled={isLoading || !capabilities.canPause}
        className={cn(
          "shrink-0 rounded p-1 transition hover:bg-wc-accent/10",
          isLoading && "cursor-wait text-wc-accent",
          (isLoading || !capabilities.canPause) && "opacity-60",
          !isLoading && !capabilities.canPause && "cursor-not-allowed",
        )}
        title={isLoading ? t(strings.app.loading) : isPaused ? t(strings.audioPlayerBar.resume) : t(strings.audioPlayerBar.pause)}
      >
        {isLoading
          ? <Loader2 data-testid="tts-playback-loading" className="h-4 w-4 animate-spin" />
          : isPaused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
      </button>

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
          onSelect={(eventId) => {
            onSelectMessage(eventId);
            setShowMessageSelector(false);
          }}
          onClose={() => setShowMessageSelector(false)}
          desktopStyle={getMessageSelectorStyle()}
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
      <input
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
      />

      <span
        data-testid="tts-time"
        className="shrink-0 whitespace-nowrap text-center text-[11px] tabular-nums text-wc-text-muted"
      >
        {formatTime(currentTime)} / {formatTime(duration)}
      </span>

      {/* Audio button — context-sensitive: when muted, single-tap unmutes; when
          unmuted, opens the popover with volume/speed/mute controls. */}
      {capabilities.canAdjustVolume && (
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

      {onDismiss && (
        <button
          data-testid="tts-dismiss"
          type="button"
          onClick={onDismiss}
          className="shrink-0 rounded p-1 text-wc-text-muted transition hover:bg-wc-accent/10 hover:text-wc-text-primary"
          title={t(strings.audioPlayerBar.closePlayback)}
          aria-label={t(strings.audioPlayerBar.closePlayback)}
        >
          <X className="h-4 w-4" />
        </button>
      )}

      {/* Popover / bottom sheet — always rendered via portal to escape terminal touch handlers */}
      {showPopover && createPortal(
        isMobile ? (
          // Mobile bottom sheet
          <div className="fixed inset-0 z-[60]" onMouseDown={(e) => e.preventDefault()}>
            <div
              data-testid="audio-sheet-backdrop"
              className="absolute inset-0 bg-wc-backdrop"
              onClick={() => setShowPopover(false)}
            />
            <div
              data-testid="audio-popover"
              className="absolute bottom-0 left-0 right-0 z-[61] rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised p-4 pb-[max(1rem,var(--wc-safe-bottom))] ps-[max(1rem,var(--wc-safe-left,0px))] pe-[max(1rem,var(--wc-safe-right,0px))] shadow-2xl"
            >
              <div className="mb-3 flex justify-center">
                <div className="h-1 w-8 rounded-full bg-wc-text-muted/40" />
              </div>
              <h3 className="mb-3 text-sm font-semibold text-wc-text-primary">{t(strings.audioPlayerBar.audioSettingsHeading)}</h3>
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
              className="fixed inset-0 z-[60]"
              onClick={() => setShowPopover(false)}
            />
            <div
              data-testid="audio-popover"
              className="z-[61] w-60 rounded-xl border border-wc-default bg-wc-surface-raised p-3 shadow-lg"
              style={getPopoverStyle()}
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
