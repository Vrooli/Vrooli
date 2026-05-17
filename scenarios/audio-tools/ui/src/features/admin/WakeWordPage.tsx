import { Panel } from "../../components/ui/panel";
import { PageHeader } from "../../components/composites/PageHeader";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

/**
 * Stub wake-word admin page. The runtime surface lives in
 * STTAdminService.{Get,Update,Delete}WakeWordTemplate. A full UX
 * (record-sample upload, threshold tuning, label editing) is tracked as
 * follow-up work; this page exists so the navigation surface and route
 * are wired up.
 */
export function WakeWordPage() {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t(strings.wakeWordAdmin.pageTitle)}
        description={t(strings.wakeWordAdmin.pageDescription)}
      />
      <Panel title={t(strings.wakeWordAdmin.pageTitle)}>
        <p className="px-4 py-3 text-sm text-app-muted-foreground">
          {t(strings.wakeWordAdmin.noTemplate)}
        </p>
      </Panel>
    </div>
  );
}
