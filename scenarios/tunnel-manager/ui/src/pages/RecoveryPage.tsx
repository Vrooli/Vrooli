import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { RecoveryPanel } from "../features/recovery/RecoveryPanel";
import { Page } from "./Page";

/** Recovery & Events surface — state machine, event timeline, manual recover. */
export function RecoveryPage() {
  const { t } = useTranslation();
  return (
    <Page
      testId={selectors.pages.recovery}
      headingId="recovery-heading"
      title={t(strings.pages.recovery.title)}
      description={t(strings.pages.recovery.description)}
    >
      <RecoveryPanel />
    </Page>
  );
}
