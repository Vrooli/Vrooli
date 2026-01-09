/**
 * BrandingSection Component
 *
 * Branding and overlay settings. Includes:
 * - Watermark settings
 * - Intro card settings
 * - Outro card settings
 */

import { WatermarkSettings } from '@/domains/exports/replay/WatermarkSettings';
import { IntroCardSettings } from '@/domains/exports/replay/IntroCardSettings';
import { OutroCardSettings } from '@/domains/exports/replay/OutroCardSettings';
import { SettingSection } from '@/views/SettingsView/sections/shared';

export function BrandingSection() {
  // Note: Watermark, Intro, and Outro settings components use the settings store directly
  // This is intentional as they have complex internal state and asset management
  return (
    <div className="space-y-6">
      <SettingSection title="Watermark" tooltip="Add a logo overlay to your replays.">
        <WatermarkSettings />
      </SettingSection>

      <SettingSection title="Intro Card" tooltip="Show a title slide before the replay starts." defaultOpen={false}>
        <IntroCardSettings />
      </SettingSection>

      <SettingSection title="Outro Card" tooltip="Show a closing slide with CTA after the replay ends." defaultOpen={false}>
        <OutroCardSettings />
      </SettingSection>
    </div>
  );
}
