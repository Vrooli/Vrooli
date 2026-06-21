import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { DriftPanel } from "../features/drift/DriftPanel";
import { Page } from "./Page";

/** Drift surface — live ingress classified against the manifest and ledger. */
export function DriftPage() {
  const { t } = useTranslation();
  return (
    <Page
      testId={selectors.pages.drift}
      headingId="drift-heading"
      title={t(strings.pages.drift.title)}
      description={t(strings.pages.drift.description)}
    >
      <DriftPanel />
    </Page>
  );
}
