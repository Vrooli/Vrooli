import { Link, useLocation } from "react-router-dom";

import { Page } from "../components/Page";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { preserveRoutedTestMode } from "../features/routedTestMode";
import { useTranslation } from "../i18n";

const HEADING_ID = "not-found-heading";

/** An address the Hub does not serve: say so inside the shell, with one way back. */
export function NotFoundPage() {
  const { t } = useTranslation();
  const { search } = useLocation();
  return (
    <Page title={t(strings.pages.notFound.title)} description={t(strings.pages.notFound.detail)} headingId={HEADING_ID} testId={selectors.pages.notFound}>
      <Link to={preserveRoutedTestMode("/asks", search)} className="inline-flex min-h-11 items-center self-start font-medium text-app-primary underline-offset-4 hover:underline">
        {t(strings.pages.notFound.home)}
      </Link>
    </Page>
  );
}
