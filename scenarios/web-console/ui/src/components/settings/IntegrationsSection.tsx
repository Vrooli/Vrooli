import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import IntegrationsPanel from "../IntegrationsPanel";

import { SettingsList } from "@vrooli/react-component-library/SettingsList/0.1.5";

export default function IntegrationsSection({ open }: { open: boolean }) {
  const { t } = useTranslation();
  return (
    <SettingsList>
      <SettingsList.Intro
        eyebrow={t(strings.settings.integrationsSection.eyebrow)}
        title={t(strings.settings.integrationsSection.title)}
        description={t(strings.settings.integrationsSection.description)}
      />

      <SettingsList.Group>
        <IntegrationsPanel open={open} />
      </SettingsList.Group>
    </SettingsList>
  );
}
