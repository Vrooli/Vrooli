import { Link } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ROUTES } from "../routes.generated";

export function NotFoundPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.notFound}
      aria-labelledby="not-found-heading"
      className="flex flex-col items-start gap-3"
    >
      <h2 id="not-found-heading" className="text-2xl font-semibold tracking-tight">
        {t(strings.pages.notFound.title)}
      </h2>
      <p className="text-sm text-app-muted-foreground">{t(strings.pages.notFound.description)}</p>
      <Link
        to={ROUTES.dashboard}
        className="rounded-control bg-app-primary px-3 py-2 text-sm font-medium text-app-primary-foreground hover:opacity-90"
      >
        {t(strings.pages.notFound.back)}
      </Link>
    </section>
  );
}
