import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import IntegrationsPanel from "../IntegrationsPanel";
import { SettingsCard, SettingsSectionIntro } from "./primitives";

export default function IntegrationsSection({ open }: { open: boolean }) {
  const { t } = useTranslation();
  return (
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow={t(strings.settings.integrationsSection.eyebrow)}
        title={t(strings.settings.integrationsSection.title)}
        description={t(strings.settings.integrationsSection.description)}
      />

      <SettingsCard>
        <IntegrationsPanel open={open} />
      </SettingsCard>
    </div>
  );
}
