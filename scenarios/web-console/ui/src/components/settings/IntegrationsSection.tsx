import IntegrationsPanel from "../IntegrationsPanel";
import { SettingsCard, SettingsSectionIntro } from "./primitives";

export default function IntegrationsSection({ open }: { open: boolean }) {
  return (
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow="Providers"
        title="Integrations"
        description="Configure external services and inspect provider availability from one place."
      />

      <SettingsCard>
        <IntegrationsPanel open={open} />
      </SettingsCard>
    </div>
  );
}
