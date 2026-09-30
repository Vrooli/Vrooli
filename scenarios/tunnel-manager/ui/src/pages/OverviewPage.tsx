import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { OverviewPanel } from "../features/overview/OverviewPanel";
import { Page } from "./Page";

/** Overview surface — tunnel health, exposure split, and recovery at a glance. */
export function OverviewPage() {
  const { t } = useTranslation();
  return (
    <Page
      testId={selectors.pages.overview}
      headingId="overview-heading"
      title={t(strings.pages.overview.title)}
      description={t(strings.pages.overview.description)}
    >
      <OverviewPanel />
    </Page>
  );
}
