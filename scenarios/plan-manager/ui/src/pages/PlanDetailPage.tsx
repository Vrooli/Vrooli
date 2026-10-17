import { Link, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { strings } from "../consts/strings";
import { PlanDetail } from "../features/plans/PlanDetail";
import { useTranslation } from "../i18n";

/** Plan detail page — reads the `:planId` route param and renders the detail. */
export function PlanDetailPage() {
  const { t } = useTranslation();
  const { planId = "" } = useParams<{ planId: string }>();

  return (
    <div className="flex flex-col gap-4">
      <Link
        to="/plans"
        className="inline-flex w-fit items-center gap-1 rounded-control text-sm text-app-muted-foreground hover:text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
      >
        <ArrowLeft aria-hidden="true" className="h-4 w-4" />
        {t(strings.layout.nav.plans)}
      </Link>
      <PlanDetail planId={planId} />
    </div>
  );
}
