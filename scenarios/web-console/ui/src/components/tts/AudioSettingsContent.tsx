import { useCallback, type ChangeEvent } from "react";
import type { TTSPlaybackCapabilities } from "../../hooks/tts/types";
import { cn } from "../../lib/classnames";

const SPEED_PRESETS = [0.5, 0.75, 1, 1.25, 1.5, 2] as const;

export interface AudioSettingsContentProps {
  /** Prefix for `data-testid` attributes so multiple instances can coexist. */
  testIdPrefix: string;
  volume: number;
  playbackRate: number;
  isSummarized: boolean;
  capabilities: TTSPlaybackCapabilities;
  onVolumeChange: (level: number) => void;
  onSetPlaybackRate: (rate: number) => void;
}

/**
 * Shared body of the TTS settings popover (desktop) or bottom sheet (mobile).
 * Renders volume slider and speed presets. Mode/level switching lives in
 * `PlaybackModeControl`, rendered on the bar itself.
 */
export function AudioSettingsContent({
  testIdPrefix,
  volume,
  playbackRate,
  isSummarized,
  capabilities,
  onVolumeChange,
  onSetPlaybackRate,
}: AudioSettingsContentProps) {
  const handleVolumeChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => onVolumeChange(Number(e.target.value)),
    [onVolumeChange],
  );

  return (
    <div className="space-y-3">
      {capabilities.canAdjustVolume && (
        <div>
          <label className="mb-1.5 block text-[10px] font-medium uppercase tracking-wider text-wc-text-faint">
            Volume
          </label>
          <input
            data-testid={`${testIdPrefix}-volume-slider`}
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

      {capabilities.canAdjustSpeed && (
        <div className={cn(capabilities.canAdjustVolume && "border-t border-wc-default pt-3")}>
          <label className="mb-1.5 block text-[10px] font-medium uppercase tracking-wider text-wc-text-faint">
            Speed
          </label>
          <div className="grid grid-cols-6 gap-1">
            {SPEED_PRESETS.map((rate) => {
              const isActive = rate === playbackRate;
              return (
                <button
                  key={rate}
                  type="button"
                  data-testid={`${testIdPrefix}-speed-preset-${rate}`}
                  onClick={() => onSetPlaybackRate(rate)}
                  className={cn(
                    "rounded-md px-1 py-1 text-[10px] font-medium tabular-nums transition",
                    isActive
                      ? "bg-wc-accent/20 text-wc-accent ring-1 ring-wc-accent/40"
                      : "bg-wc-surface-base text-wc-text-muted hover:bg-wc-surface-input",
                  )}
                >
                  {rate}x
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
