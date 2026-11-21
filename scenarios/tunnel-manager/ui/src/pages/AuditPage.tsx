import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { AuditPanel } from "../features/audit/AuditPanel";
import { Page } from "./Page";

/** Audit surface — port-compliance findings across the manifest. */
export function AuditPage() {
  const { t } = useTranslation();
  return (
    <Page
      testId={selectors.pages.audit}
      headingId="audit-heading"
      title={t(strings.pages.audit.title)}
      description={t(strings.pages.audit.description)}
    >
      <AuditPanel />
    </Page>
  );
}
