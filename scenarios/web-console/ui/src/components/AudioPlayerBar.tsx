import { useCallback, useRef, useState, type ChangeEvent } from "react";
import { Pause, Play, Square, Volume2, VolumeX } from "lucide-react";
import type { TTSPlaybackCapabilities } from "../hooks/tts/types";
import { cn } from "../lib/classnames";

const SPEED_PRESETS = [0.5, 0.75, 1, 1.25, 1.5, 2] as const;

export interface AudioPlayerBarProps {
  isPaused: boolean;
  currentTime: number;
  duration: number | null;
  playbackRate: number;
  volume: number;
  capabilities: TTSPlaybackCapabilities;
  onPause: () => void;
  onResume: () => void;
  onSeek: (seconds: number) => void;
  onSetPlaybackRate: (rate: number) => void;
  onSetVolume: (level: number) => void;
  onStop: () => void;
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
 * Renders pause/resume, stop, scrub bar, time display, speed selector,
 * and volume toggle. Controls that the active provider doesn't support
 * are hidden based on the `capabilities` prop.
 *
 * Follows the Spotify mini-player pattern: visible when audio is playing,
 * positioned at the bottom of the Workspace above the MobileToolbar.
 */
export default function AudioPlayerBar({
  isPaused,
  currentTime,
  duration,
  playbackRate,
  volume,
  capabilities,
  onPause,
  onResume,
  onSeek,
  onSetPlaybackRate,
  onSetVolume,
  onStop,
}: AudioPlayerBarProps) {
  const [showVolumeSlider, setShowVolumeSlider] = useState(false);
  const volumeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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

  const handleVolumeChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      onSetVolume(Number(e.target.value));
    },
    [onSetVolume],
  );

  const handleVolumeToggle = useCallback(() => {
    // Toggle mute: if volume > 0 → mute, else restore to 1
    onSetVolume(volume > 0 ? 0 : 1);
  }, [volume, onSetVolume]);

  const handleVolumeEnter = useCallback(() => {
    if (volumeTimerRef.current) {
      clearTimeout(volumeTimerRef.current);
      volumeTimerRef.current = null;
    }
    setShowVolumeSlider(true);
  }, []);

  const handleVolumeLeave = useCallback(() => {
    volumeTimerRef.current = setTimeout(() => setShowVolumeSlider(false), 300);
  }, []);

  const isMuted = volume === 0;
  const showScrub = capabilities.canSeek && duration !== null;

  return (
    <div
      data-testid="audio-player-bar"
      className="flex items-center gap-2 border-t border-wc-default bg-wc-surface-raised px-3 py-1.5 text-wc-text-primary animate-in slide-in-from-bottom-2 duration-200"
    >
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
          className="mx-1 h-1 flex-1 cursor-pointer accent-wc-accent"
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

      {/* Volume */}
      {capabilities.canAdjustVolume && (
        <div
          className="relative flex items-center"
          onMouseEnter={handleVolumeEnter}
          onMouseLeave={handleVolumeLeave}
        >
          <button
            data-testid="tts-volume-toggle"
            onClick={handleVolumeToggle}
            className="rounded p-1 transition hover:bg-wc-accent/10"
            title={isMuted ? "Unmute" : "Mute"}
          >
            {isMuted ? <VolumeX className="h-4 w-4" /> : <Volume2 className="h-4 w-4" />}
          </button>
          {showVolumeSlider && (
            <input
              data-testid="tts-volume-slider"
              type="range"
              min={0}
              max={1}
              step={0.05}
              value={volume}
              onChange={handleVolumeChange}
              className="ml-1 h-1 w-20 cursor-pointer accent-wc-accent"
            />
          )}
        </div>
      )}
    </div>
  );
}
