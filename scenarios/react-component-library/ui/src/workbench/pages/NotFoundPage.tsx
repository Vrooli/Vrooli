import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";

export function NotFoundPage() {
  const { t } = useTranslation();
  return (
    <div className="wb-page" data-testid="not-found-page">
      <p className="wb-kicker">{t("workbench.notFound.kicker", { defaultValue: "404 · Not found" })}</p>
      <h1 className="wb-h1">{t("workbench.notFound.title", { defaultValue: "That surface moved." })}</h1>
      <p className="wb-lede">
        {t("workbench.notFound.body", {
          defaultValue:
            "The route is not part of the workbench. Legacy catalog and design links redirect to their new homes; anything else is genuinely missing.",
        })}
      </p>
      <Link to={appRoutes.start} className="wb-btn primary">
        <ArrowLeft aria-hidden size={14} />
        {t("workbench.notFound.back", { defaultValue: "Back to Home" })}
      </Link>
    </div>
  );
}
