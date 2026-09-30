import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { MetricsPanel } from "../features/metrics/MetricsPanel";
import { Page } from "./Page";

/** Metrics surface — tunnel metrics time-series + per-route probe history. */
export function MetricsPage() {
  const { t } = useTranslation();
  return (
    <Page
      testId={selectors.pages.metrics}
      headingId="metrics-heading"
      title={t(strings.pages.metrics.title)}
      description={t(strings.pages.metrics.description)}
    >
      <MetricsPanel />
    </Page>
  );
}
