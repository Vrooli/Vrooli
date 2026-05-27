import { Panel } from "../../components/ui/panel";
import { PageHeader } from "../../components/composites/PageHeader";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

/**
 * Stub stream-config admin page. The runtime surface lives in
 * STTAdminService.{Get,Update}StreamConfig. A full UX (per-knob
 * editing, validate-against-protovalidate) is tracked as follow-up
 * work.
 */
export function StreamConfigPage() {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t(strings.streamConfigAdmin.pageTitle)}
        description={t(strings.streamConfigAdmin.pageDescription)}
      />
      <Panel title={t(strings.streamConfigAdmin.pageTitle)}>
        <p className="px-4 py-3 text-sm text-app-muted-foreground">
          {t(strings.streamConfigAdmin.notImplemented)}
        </p>
      </Panel>
    </div>
  );
}
