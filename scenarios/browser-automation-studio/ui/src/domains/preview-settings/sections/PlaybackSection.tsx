/**
 * PlaybackSection Component
 *
 * Playback and dimension settings. Includes:
 * - Dimension presets and custom dimensions
 * - Frame duration
 * - Autoplay toggle
 * - Loop toggle
 */

import { Monitor, Smartphone, Tv } from 'lucide-react';
import { useMemo } from 'react';
import { RangeSlider } from '@shared/ui';
import { SettingSection, OptionGrid, ToggleSwitch } from '@/views/SettingsView/sections/shared';

const DIMENSION_PRESETS = [
  {
    id: 'widescreen-720',
    label: '1280 × 720',
    description: 'Best for YouTube, demos, decks',
    width: 1280,
    height: 720,
    icon: <Monitor size={16} />,
    ratioLabel: '16:9',
  },
  {
    id: 'full-hd',
    label: '1920 × 1080',
    description: 'Best for desktop screenshares',
    width: 1920,
    height: 1080,
    icon: <Tv size={16} />,
    ratioLabel: '16:9',
  },
  {
    id: 'instagram-feed',
    label: '1080 × 1350',
    description: 'Best for Instagram feed posts',
    width: 1080,
    height: 1350,
    icon: <Smartphone size={16} />,
    ratioLabel: '4:5',
  },
  {
    id: 'tiktok',
    label: '1080 × 1920',
    description: 'Best for TikTok, Reels, Shorts',
    width: 1080,
    height: 1920,
    icon: <Smartphone size={16} />,
    ratioLabel: '9:16',
  },
] as const;

interface PlaybackSectionProps {
  /** Presentation width */
  presentationWidth: number;
  /** Presentation height */
  presentationHeight: number;
  /** Whether custom dimensions are used */
  useCustomDimensions: boolean;
  /** Frame duration in ms */
  frameDuration: number;
  /** Whether to autoplay */
  autoPlay: boolean;
  /** Whether to loop */
  loop: boolean;
  /** Callbacks */
  onPresentationWidthChange: (value: number) => void;
  onPresentationHeightChange: (value: number) => void;
  onUseCustomDimensionsChange: (value: boolean) => void;
  onFrameDurationChange: (value: number) => void;
  onAutoPlayChange: (value: boolean) => void;
  onLoopChange: (value: boolean) => void;
}

export function PlaybackSection({
  presentationWidth,
  presentationHeight,
  useCustomDimensions,
  frameDuration,
  autoPlay,
  loop,
  onPresentationWidthChange,
  onPresentationHeightChange,
  onUseCustomDimensionsChange,
  onFrameDurationChange,
  onAutoPlayChange,
  onLoopChange,
}: PlaybackSectionProps) {
  const activeDimensionPreset = useMemo(() => {
    return DIMENSION_PRESETS.find(
      (preset) =>
        preset.width === presentationWidth &&
        preset.height === presentationHeight
    ) ?? DIMENSION_PRESETS[0];
  }, [presentationHeight, presentationWidth]);

  return (
    <div className="space-y-6">
      <SettingSection title="Replay Dimensions" tooltip="Set the default canvas size for styled replays.">
        <div className="space-y-4">
          <OptionGrid
            options={DIMENSION_PRESETS.map((preset) => ({
              id: preset.id,
              label: preset.label,
              description: preset.description,
              preview: (
                <div className="flex flex-col items-center gap-1 text-gray-400">
                  <div className="flex items-center gap-1 text-[11px]">
                    {preset.icon}
                    <span>{preset.ratioLabel}</span>
                  </div>
                  <div className="h-6 w-10 rounded border border-gray-600/80 bg-gray-900/60" />
                </div>
              ),
            }))}
            value={activeDimensionPreset.id}
            onChange={(id) => {
              const preset = DIMENSION_PRESETS.find((item) => item.id === id);
              if (!preset) return;
              onPresentationWidthChange(preset.width);
              onPresentationHeightChange(preset.height);
              onUseCustomDimensionsChange(false);
            }}
            columns={2}
          />

          <ToggleSwitch
            checked={useCustomDimensions}
            onChange={onUseCustomDimensionsChange}
            label="Use custom dimensions"
            description="Manually set width and height in pixels"
          />

          {useCustomDimensions && (
            <div className="grid grid-cols-2 gap-3">
              <label className="text-xs text-gray-400 flex flex-col gap-1">
                Width (px)
                <input
                  type="number"
                  min={320}
                  max={3840}
                  step={1}
                  value={presentationWidth}
                  onChange={(e) => {
                    const next = Math.round(Number(e.target.value));
                    const clamped = Number.isFinite(next) ? Math.min(3840, Math.max(320, next)) : 1280;
                    onPresentationWidthChange(clamped);
                  }}
                  className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700 rounded-lg text-surface focus:outline-none focus:ring-2 focus:ring-flow-accent/50"
                />
              </label>
              <label className="text-xs text-gray-400 flex flex-col gap-1">
                Height (px)
                <input
                  type="number"
                  min={320}
                  max={3840}
                  step={1}
                  value={presentationHeight}
                  onChange={(e) => {
                    const next = Math.round(Number(e.target.value));
                    const clamped = Number.isFinite(next) ? Math.min(3840, Math.max(320, next)) : 720;
                    onPresentationHeightChange(clamped);
                  }}
                  className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700 rounded-lg text-surface focus:outline-none focus:ring-2 focus:ring-flow-accent/50"
                />
              </label>
            </div>
          )}
        </div>
      </SettingSection>

      <SettingSection title="Timing" tooltip="Control timing and behavior of replay animations.">
        <div className="space-y-5">
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-300">Frame Duration</label>
              <span className="text-sm font-medium text-flow-accent">{(frameDuration / 1000).toFixed(1)}s</span>
            </div>
            <RangeSlider
              min={800}
              max={6000}
              step={100}
              value={frameDuration}
              onChange={onFrameDurationChange}
              ariaLabel="Frame duration"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>Fast (0.8s)</span>
              <span>Slow (6s)</span>
            </div>
          </div>
          <ToggleSwitch
            checked={autoPlay}
            onChange={onAutoPlayChange}
            label="Auto-play replays"
            description="Start playing immediately when opened"
          />
          <ToggleSwitch
            checked={loop}
            onChange={onLoopChange}
            label="Loop replays"
            description="Restart automatically when finished"
          />
        </div>
      </SettingSection>
    </div>
  );
}
