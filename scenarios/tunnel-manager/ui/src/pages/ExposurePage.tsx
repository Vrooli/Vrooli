import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ExposurePanel } from "../features/exposure/ExposurePanel";
import { Page } from "./Page";

/** Exposure surface — the core+leased exposure table and broker actions. */
export function ExposurePage() {
  const { t } = useTranslation();
  return (
    <Page
      testId={selectors.pages.exposure}
      headingId="exposure-heading"
      title={t(strings.pages.exposure.title)}
      description={t(strings.pages.exposure.description)}
    >
      <ExposurePanel />
    </Page>
  );
}
