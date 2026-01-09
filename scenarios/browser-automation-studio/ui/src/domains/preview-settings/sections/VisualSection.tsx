/**
 * VisualSection Component
 *
 * Visual styling settings for replay. Includes:
 * - Presentation mode (desktop, browser frame, device frame toggles)
 * - Background settings (presets, solid, gradient, pattern, image)
 * - Browser chrome theme
 * - Device frame theme
 */

import {
  ReplayPresentationModeSettings,
  ReplayBackgroundSettings,
  ReplayChromeSettings,
  ReplayDeviceFrameSettings,
} from '@/domains/replay-style';
import type {
  ReplayPresentationSettings,
  ReplayBackgroundSource,
  ReplayChromeTheme,
  ReplayDeviceFrameTheme,
} from '@/domains/replay-style';
import { SettingSection } from '@/views/SettingsView/sections/shared';

interface VisualSectionProps {
  /** Presentation mode settings */
  presentation: ReplayPresentationSettings;
  /** Background configuration */
  background: ReplayBackgroundSource;
  /** Browser chrome theme */
  chromeTheme: ReplayChromeTheme;
  /** Browser scale factor */
  browserScale: number;
  /** Device frame theme */
  deviceFrameTheme: ReplayDeviceFrameTheme;
  /** Callbacks */
  onPresentationChange: (value: ReplayPresentationSettings) => void;
  onBackgroundChange: (value: ReplayBackgroundSource) => void;
  onChromeThemeChange: (value: ReplayChromeTheme) => void;
  onBrowserScaleChange: (value: number) => void;
  onDeviceFrameThemeChange: (value: ReplayDeviceFrameTheme) => void;
}

export function VisualSection({
  presentation,
  background,
  chromeTheme,
  browserScale,
  deviceFrameTheme,
  onPresentationChange,
  onBackgroundChange,
  onChromeThemeChange,
  onBrowserScaleChange,
  onDeviceFrameThemeChange,
}: VisualSectionProps) {
  return (
    <div className="space-y-6">
      <SettingSection title="Presentation Mode" tooltip="Choose the overall replay framing style.">
        <ReplayPresentationModeSettings
          presentation={presentation}
          onChange={onPresentationChange}
          variant="settings"
        />
      </SettingSection>

      {presentation.showBrowserFrame && (
        <SettingSection title="Browser Chrome" tooltip="Choose how the browser window frame looks.">
          <ReplayChromeSettings
            chromeTheme={chromeTheme}
            browserScale={browserScale}
            onChromeThemeChange={onChromeThemeChange}
            onBrowserScaleChange={onBrowserScaleChange}
            variant="settings"
          />
        </SettingSection>
      )}

      {presentation.showDesktop && (
        <SettingSection title="Background" tooltip="The backdrop behind the browser window.">
          <ReplayBackgroundSettings
            background={background}
            onBackgroundChange={onBackgroundChange}
            variant="settings"
          />
        </SettingSection>
      )}

      {presentation.showDesktop && presentation.showDeviceFrame && (
        <SettingSection title="Device Frame" tooltip="A hardware-style frame around the desktop stage.">
          <ReplayDeviceFrameSettings
            deviceFrameTheme={deviceFrameTheme}
            onDeviceFrameThemeChange={onDeviceFrameThemeChange}
            variant="settings"
          />
        </SettingSection>
      )}
    </div>
  );
}
