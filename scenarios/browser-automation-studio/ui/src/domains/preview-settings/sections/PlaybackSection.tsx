/**
 * PlaybackSection Component
 *
 * Playback and dimension settings. Includes:
 * - Dimension presets and custom dimensions
 * - Frame duration
 * - Autoplay toggle
 * - Loop toggle
 */

import { useCallback } from 'react';
import { RangeSlider, ViewportPicker } from '@shared/ui';
import { SettingSection, ToggleSwitch } from '@/views/SettingsView/sections/shared';

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
  const handleViewportChange = useCallback(
    ({ width, height }: { width: number; height: number }) => {
      onPresentationWidthChange(width);
      onPresentationHeightChange(height);
    },
    [onPresentationWidthChange, onPresentationHeightChange]
  );

  return (
    <div className="space-y-6">
      <SettingSection title="Replay Dimensions" tooltip="Set the default canvas size for styled replays.">
        <ViewportPicker
          value={{ width: presentationWidth, height: presentationHeight }}
          onChange={handleViewportChange}
          useCustom={useCustomDimensions}
          onUseCustomChange={onUseCustomDimensionsChange}
          showCustomToggle={true}
          variant="dark"
        />
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
