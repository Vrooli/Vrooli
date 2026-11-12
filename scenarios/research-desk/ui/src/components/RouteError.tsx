import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { useRouteError } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * A screen that failed to render, inside the shell so every other tab still
 * works: the router catches render errors before the app-level boundary, and
 * its default page is a raw stack wider than a phone.
 */
export function RouteError() {
  const { t } = useTranslation();
  const error = useRouteError();
  const detail = error instanceof Error ? error.message : "";
  return (
    <div className="rd-page" data-testid={selectors.errorBoundary.root}>
      <Alert
        tone="danger"
        title={t(strings.errorBoundary.title)}
        description={
          <span className="flex flex-col gap-space-2xs">
            <span>{t(strings.errorBoundary.message)}</span>
            {detail ? <code>{detail}</code> : null}
          </span>
        }
        actions={
          <Button data-testid={selectors.errorBoundary.retryButton} variant="secondary" onClick={() => window.location.reload()}>
            {t(strings.errorBoundary.retry)}
          </Button>
        }
      />
    </div>
  );
}
