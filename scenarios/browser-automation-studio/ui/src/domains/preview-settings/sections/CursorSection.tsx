/**
 * CursorSection Component
 *
 * Cursor styling and motion settings. Includes:
 * - Cursor theme
 * - Initial position
 * - Click animation
 * - Cursor scale
 * - Speed profile
 * - Path style
 */

import { ReplayCursorSettings } from '@/domains/replay-style';
import type {
  ReplayCursorTheme,
  ReplayCursorInitialPosition,
  ReplayCursorClickAnimation,
} from '@/domains/replay-style';
import type { CursorSpeedProfile, CursorPathStyle } from '@/domains/exports/replay/ReplayPlayer';
import { SettingSection, OptionGrid } from '@/views/SettingsView/sections/shared';

const SPEED_PROFILE_OPTIONS: Array<{ id: CursorSpeedProfile; label: string; description: string }> = [
  { id: 'linear', label: 'Linear', description: 'Consistent motion between frames' },
  { id: 'easeIn', label: 'Ease In', description: 'Begin slowly, accelerate toward target' },
  { id: 'easeOut', label: 'Ease Out', description: 'Move quickly, settle into target' },
  { id: 'easeInOut', label: 'Ease In/Out', description: 'Smooth acceleration and deceleration' },
  { id: 'instant', label: 'Instant', description: 'Jump directly at the end' },
];

const PATH_STYLE_OPTIONS: Array<{ id: CursorPathStyle; label: string; description: string }> = [
  { id: 'linear', label: 'Linear', description: 'Direct line between positions' },
  { id: 'parabolicUp', label: 'Arc Up', description: 'Arcs upward before target' },
  { id: 'parabolicDown', label: 'Arc Down', description: 'Arcs downward before target' },
  { id: 'cubic', label: 'Cubic', description: 'Smooth S-curve motion' },
  { id: 'pseudorandom', label: 'Organic', description: 'Natural random waypoints' },
];

interface CursorSectionProps {
  /** Cursor theme */
  cursorTheme: ReplayCursorTheme;
  /** Initial cursor position */
  cursorInitialPosition: ReplayCursorInitialPosition;
  /** Click animation style */
  cursorClickAnimation: ReplayCursorClickAnimation;
  /** Cursor scale factor */
  cursorScale: number;
  /** Speed profile for cursor motion */
  cursorSpeedProfile: CursorSpeedProfile;
  /** Path style for cursor motion */
  cursorPathStyle: CursorPathStyle;
  /** Callbacks */
  onCursorThemeChange: (value: ReplayCursorTheme) => void;
  onCursorInitialPositionChange: (value: ReplayCursorInitialPosition) => void;
  onCursorClickAnimationChange: (value: ReplayCursorClickAnimation) => void;
  onCursorScaleChange: (value: number) => void;
  onCursorSpeedProfileChange: (value: CursorSpeedProfile) => void;
  onCursorPathStyleChange: (value: CursorPathStyle) => void;
}

export function CursorSection({
  cursorTheme,
  cursorInitialPosition,
  cursorClickAnimation,
  cursorScale,
  cursorSpeedProfile,
  cursorPathStyle,
  onCursorThemeChange,
  onCursorInitialPositionChange,
  onCursorClickAnimationChange,
  onCursorScaleChange,
  onCursorSpeedProfileChange,
  onCursorPathStyleChange,
}: CursorSectionProps) {
  return (
    <div className="space-y-6">
      <SettingSection title="Cursor Style" tooltip="Style the virtual cursor shown during replay.">
        <ReplayCursorSettings
          cursorTheme={cursorTheme}
          cursorInitialPosition={cursorInitialPosition}
          cursorClickAnimation={cursorClickAnimation}
          cursorScale={cursorScale}
          onCursorThemeChange={onCursorThemeChange}
          onCursorInitialPositionChange={onCursorInitialPositionChange}
          onCursorClickAnimationChange={onCursorClickAnimationChange}
          onCursorScaleChange={onCursorScaleChange}
          variant="settings"
        />
      </SettingSection>

      <SettingSection title="Cursor Motion" tooltip="Controls how the cursor moves between actions." defaultOpen={false}>
        <div className="space-y-4">
          <div>
            <label className="text-sm text-gray-400 mb-2 block">Speed Profile</label>
            <OptionGrid
              options={SPEED_PROFILE_OPTIONS}
              value={cursorSpeedProfile}
              onChange={onCursorSpeedProfileChange}
              columns={3}
            />
          </div>
          <div>
            <label className="text-sm text-gray-400 mb-2 block">Path Style</label>
            <OptionGrid
              options={PATH_STYLE_OPTIONS}
              value={cursorPathStyle}
              onChange={onCursorPathStyleChange}
              columns={3}
            />
          </div>
        </div>
      </SettingSection>
    </div>
  );
}
