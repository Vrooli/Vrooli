import { useCallback, useRef, useState, type ChangeEvent } from "react";
import { createPortal } from "react-dom";
import { Pause, Play, Square, Volume2, VolumeX } from "lucide-react";
import type { TTSPlaybackCapabilities } from "../hooks/tts/types";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { cn } from "../lib/classnames";

const SPEED_PRESETS = [0.5, 0.75, 1, 1.25, 1.5, 2] as const;

export interface AudioPlayerBarProps {
  isPaused: boolean;
  currentTime: number;
  duration: number | null;
  playbackRate: number;
  volume: number;
  capabilities: TTSPlaybackCapabilities;
  /** Whether the currently playing content is a summarized version. */
  isSummarized?: boolean;
  /** Whether the active event has an original (unsummarized) version available. */
  hasOriginalVersion?: boolean;
  /** Whether summarization is available (summarizer configured on backend). */
  canSummarize?: boolean;
  /** Whether a summarization request is in progress. */
  isSummarizing?: boolean;
  onPause: () => void;
  onResume: () => void;
  onSeek: (seconds: number) => void;
  onSetPlaybackRate: (rate: number) => void;
  onSetVolume: (level: number) => void;
  onStop: () => void;
  /** Called when the user wants to switch between summarized and original playback. */
  onToggleSummarized?: (useSummarized: boolean) => void;
  /** Called when the user requests on-demand summarization for the active event. */
  onRequestSummarize?: () => void;
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
 * Audio settings content — shared between the desktop popover and the
 * mobile bottom sheet. Renders volume slider and summarization controls.
 */
function AudioSettingsContent({
  volume,
  isSummarized,
  hasOriginalVersion,
  canSummarize,
  isSummarizing,
  capabilities,
  onVolumeChange,
  onToggleSummarized,
  onRequestSummarize,
}: {
  volume: number;
  isSummarized: boolean;
  hasOriginalVersion: boolean;
  canSummarize: boolean;
  isSummarizing: boolean;
  capabilities: TTSPlaybackCapabilities;
  onVolumeChange: (level: number) => void;
  onToggleSummarized?: (useSummarized: boolean) => void;
  onRequestSummarize?: () => void;
}) {
  const handleVolumeChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => onVolumeChange(Number(e.target.value)),
    [onVolumeChange],
  );

  return (
    <div className="space-y-3">
      {/* Volume */}
      {capabilities.canAdjustVolume && (
        <div>
          <label className="mb-1.5 block text-[10px] font-medium uppercase tracking-wider text-wc-text-faint">
            Volume
          </label>
          <input
            data-testid="tts-volume-slider"
            type="range"
            min={0}
            max={1}
            step={0.05}
            value={volume}
            onChange={handleVolumeChange}
            className={cn(
              "h-1.5 w-full cursor-pointer rounded-full",
              isSummarized
                ? "[&::-webkit-slider-thumb]:bg-amber-400 accent-amber-400"
                : "accent-wc-accent",
            )}
          />
          <div className="mt-0.5 flex justify-between text-[10px] text-wc-text-faint">
            <span>0</span>
            <span>{Math.round(volume * 100)}%</span>
          </div>
        </div>
      )}

      {/* Summarization toggle — shown when event already has both versions */}
      {hasOriginalVersion && onToggleSummarized && (
        <div className="border-t border-wc-default pt-3">
          <label className="mb-1.5 block text-[10px] font-medium uppercase tracking-wider text-wc-text-faint">
            Playback version
          </label>
          <div className="flex gap-1">
            <button
              data-testid="tts-play-summarized"
              className={cn(
                "flex-1 rounded-lg px-2 py-1.5 text-xs font-medium transition",
                isSummarized
                  ? "bg-amber-500/20 text-amber-300 ring-1 ring-amber-500/40"
                  : "bg-wc-surface-base text-wc-text-muted hover:bg-wc-surface-input",
              )}
              onClick={() => onToggleSummarized(true)}
            >
              Summarized
            </button>
            <button
              data-testid="tts-play-original"
              className={cn(
                "flex-1 rounded-lg px-2 py-1.5 text-xs font-medium transition",
                !isSummarized
                  ? "bg-wc-accent/20 text-wc-accent ring-1 ring-wc-accent/40"
                  : "bg-wc-surface-base text-wc-text-muted hover:bg-wc-surface-input",
              )}
              onClick={() => onToggleSummarized(false)}
            >
              Original
            </button>
          </div>
        </div>
      )}

      {/* Request summarization — shown when no summary exists yet but summarizer is available */}
      {!hasOriginalVersion && canSummarize && onRequestSummarize && (
        <div className="border-t border-wc-default pt-3">
          <button
            data-testid="tts-request-summarize"
            disabled={isSummarizing}
            className={cn(
              "w-full rounded-lg px-3 py-2 text-xs font-medium transition",
              isSummarizing
                ? "bg-wc-surface-base text-wc-text-faint cursor-wait"
                : "bg-amber-500/15 text-amber-300 hover:bg-amber-500/25",
            )}
            onClick={onRequestSummarize}
          >
            {isSummarizing ? "Summarizing…" : "Summarize for playback"}
          </button>
        </div>
      )}
    </div>
  );
}

/**
 * Global bottom bar for TTS playback controls.
 *
 * Renders pause/resume, stop, scrub bar, time display, speed selector,
 * and an audio button that opens a popover (desktop) or bottom sheet (mobile)
 * with volume slider and summarization controls.
 */
export default function AudioPlayerBar({
  isPaused,
  currentTime,
  duration,
  playbackRate,
  volume,
  capabilities,
  isSummarized = false,
  hasOriginalVersion = false,
  canSummarize = false,
  isSummarizing = false,
  onPause,
  onResume,
  onSeek,
  onSetPlaybackRate,
  onSetVolume,
  onStop,
  onToggleSummarized,
  onRequestSummarize,
}: AudioPlayerBarProps) {
  const [showPopover, setShowPopover] = useState(false);
  const isMobile = useMediaQuery("(max-width: 767px)");
  const audioButtonRef = useRef<HTMLButtonElement>(null);

  const handlePlayPause = useCallback(() => {
    if (isPaused) onResume();
    else onPause();
  }, [isPaused, onPause, onResume]);

  const handleSpeedCycle = useCallback(() => {
    const currentIndex = SPEED_PRESETS.indexOf(playbackRate as typeof SPEED_PRESETS[number]);
    const nextIndex = (currentIndex + 1) % SPEED_PRESETS.length;
    const nextSpeed = SPEED_PRESETS[nextIndex] ?? 1;
    onSetPlaybackRate(nextSpeed);
  }, [playbackRate, onSetPlaybackRate]);

  const handleScrubChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      onSeek(Number(e.target.value));
    },
    [onSeek],
  );

  const isMuted = volume === 0;
  const showScrub = capabilities.canSeek && duration !== null;

  // Compute popover position anchored above the audio button
  const getPopoverStyle = useCallback((): React.CSSProperties => {
    const btn = audioButtonRef.current;
    if (!btn) return { position: "fixed", bottom: 48, right: 16 };
    const rect = btn.getBoundingClientRect();
    return {
      position: "fixed",
      bottom: window.innerHeight - rect.top + 8,
      right: Math.max(8, window.innerWidth - rect.right),
    };
  }, []);

  return (
    <div
      data-testid="audio-player-bar"
      className={cn(
        "flex items-center gap-2 border-t px-3 py-1.5 text-wc-text-primary animate-in slide-in-from-bottom-2 duration-200",
        isSummarized
          ? "border-amber-500/30 bg-amber-950/30"
          : "border-wc-default bg-wc-surface-raised",
      )}
    >
      {/* Summarized indicator */}
      {isSummarized && (
        <span
          data-testid="tts-summarized-badge"
          className="rounded-full bg-amber-500/20 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider text-amber-400"
        >
          Summarized
        </span>
      )}

      {/* Play / Pause */}
      <button
        data-testid="tts-play-pause"
        onClick={handlePlayPause}
        disabled={!capabilities.canPause}
        className={cn(
          "rounded p-1 transition hover:bg-wc-accent/10",
          !capabilities.canPause && "opacity-40 cursor-not-allowed",
        )}
        title={isPaused ? "Resume" : "Pause"}
      >
        {isPaused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
      </button>

      {/* Stop */}
      <button
        data-testid="tts-stop"
        onClick={onStop}
        className="rounded p-1 transition hover:bg-wc-accent/10"
        title="Stop"
      >
        <Square className="h-4 w-4" />
      </button>

      {/* Scrub bar */}
      {showScrub && (
        <input
          data-testid="tts-scrub"
          type="range"
          min={0}
          max={duration}
          value={currentTime}
          step={0.1}
          onChange={handleScrubChange}
          className={cn(
            "mx-1 h-1 flex-1 cursor-pointer",
            isSummarized ? "accent-amber-400" : "accent-wc-accent",
          )}
        />
      )}

      {/* Time display */}
      <span data-testid="tts-time" className="min-w-[5rem] text-center text-xs tabular-nums text-wc-text-muted">
        {formatTime(currentTime)} / {formatTime(duration)}
      </span>

      {/* Speed selector */}
      {capabilities.canAdjustSpeed && (
        <button
          data-testid="tts-speed"
          onClick={handleSpeedCycle}
          className="rounded px-1.5 py-0.5 text-xs font-medium tabular-nums transition hover:bg-wc-accent/10"
          title="Playback speed"
        >
          {playbackRate}x
        </button>
      )}

      {/* Audio button — opens popover/sheet via portal */}
      {capabilities.canAdjustVolume && (
        <button
          ref={audioButtonRef}
          data-testid="tts-audio-button"
          onClick={() => setShowPopover((prev) => !prev)}
          className={cn(
            "rounded p-1 transition",
            isSummarized
              ? "text-amber-400 hover:bg-amber-500/10"
              : "hover:bg-wc-accent/10",
          )}
          title="Audio settings"
        >
          {isMuted ? <VolumeX className="h-4 w-4" /> : <Volume2 className="h-4 w-4" />}
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
              className="absolute bottom-0 left-0 right-0 z-[61] rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised p-4 shadow-2xl"
            >
              <div className="mb-3 flex justify-center">
                <div className="h-1 w-8 rounded-full bg-wc-text-muted/40" />
              </div>
              <h3 className="mb-3 text-sm font-semibold text-wc-text-primary">Audio Settings</h3>
              <AudioSettingsContent
                volume={volume}
                isSummarized={isSummarized}
                hasOriginalVersion={hasOriginalVersion}
                canSummarize={canSummarize}
                isSummarizing={isSummarizing}
                capabilities={capabilities}
                onVolumeChange={onSetVolume}
                onToggleSummarized={onToggleSummarized}
                onRequestSummarize={onRequestSummarize}
              />
            </div>
          </div>
        ) : (
          // Desktop popover — positioned above button via portal
          <>
            <div
              data-testid="audio-popover-backdrop"
              className="fixed inset-0 z-[60]"
              onClick={() => setShowPopover(false)}
            />
            <div
              data-testid="audio-popover"
              className="z-[61] w-56 rounded-xl border border-wc-default bg-wc-surface-raised p-3 shadow-lg"
              style={getPopoverStyle()}
            >
              <AudioSettingsContent
                volume={volume}
                isSummarized={isSummarized}
                hasOriginalVersion={hasOriginalVersion}
                canSummarize={canSummarize}
                isSummarizing={isSummarizing}
                capabilities={capabilities}
                onVolumeChange={onSetVolume}
                onToggleSummarized={onToggleSummarized}
                onRequestSummarize={onRequestSummarize}
              />
            </div>
          </>
        ),
        document.body,
      )}
    </div>
  );
}
