import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import IntegrationsPanel from "../IntegrationsPanel";
import { SettingsIntro } from "./primitives";

export default function IntegrationsSection({ open }: { open: boolean }) {
  const { t } = useTranslation();
  return (
    <IntegrationsPanel
      open={open}
      header={
        <SettingsIntro
          eyebrow={t(strings.settings.integrationsSection.eyebrow)}
          title={t(strings.settings.integrationsSection.title)}
          description={t(strings.settings.integrationsSection.description)}
        />
      }
    />
  );
}
