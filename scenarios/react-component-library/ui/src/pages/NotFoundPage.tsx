/** @vrooliComponentSource react-component-library:EmptyState */
import { Link } from "react-router-dom";

import { useTranslation } from "../i18n";

export function NotFoundPage() {
  const { t } = useTranslation();
  return (
    <div data-testid="not-found-page" className="flex flex-col items-start gap-space-xs">
      <h1 className="text-2xl font-semibold text-app-foreground">
        {t("notFound.title", { defaultValue: "Page not found" })}
      </h1>
      <p className="text-sm text-app-muted-foreground">
        {t("notFound.description", {
          defaultValue: "The page you requested doesn't exist or has moved.",
        })}
      </p>
      <Link
        to="/"
        data-testid="not-found-home"
        className="inline-flex h-9 items-center rounded-control bg-app-primary px-space-sm text-sm font-medium text-app-primary-foreground hover:brightness-95"
      >
        {t("notFound.home", { defaultValue: "Back to dashboard" })}
      </Link>
    </div>
  );
}
