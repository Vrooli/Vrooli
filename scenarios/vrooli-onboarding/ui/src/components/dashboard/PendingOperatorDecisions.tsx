import { useQuery } from "@tanstack/react-query";
import { listPendingOperatorDecisions } from "../../api/operatorDecisions";
import { Button } from "@vrooli/react-component-library/Button/2";
import { i18n } from "../../i18n";

export function PendingOperatorDecisions() {
  const query = useQuery({ queryKey: ["pending-operator-decisions"], queryFn: listPendingOperatorDecisions, refetchInterval: 30_000 });
  return (
    <section className="mt-8 rounded-xl border border-border bg-surface p-5" aria-labelledby="pending-decisions-heading" data-testid="pending-operator-decisions">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{i18n.t("onboarding.dashboard.pendingDecisionsEyebrow")}</p>
          <h2 id="pending-decisions-heading" className="mt-1 text-lg font-semibold">{i18n.t("onboarding.dashboard.pendingDecisionsHeading")}</h2>
          <p className="mt-1 text-sm text-muted">{i18n.t("onboarding.dashboard.pendingDecisionsIntro")}</p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => void query.refetch()} disabled={query.isFetching}>{i18n.t("onboarding.dashboard.refresh")}</Button>
      </div>
      {query.isLoading && <p className="mt-4 text-sm text-muted" role="status">{i18n.t("onboarding.dashboard.pendingDecisionsLoading")}</p>}
      {query.error && <p className="mt-4 text-sm text-danger" role="alert">{i18n.t("onboarding.dashboard.pendingDecisionsError")}</p>}
      {!query.isLoading && !query.error && query.data?.length === 0 && <p className="mt-4 text-sm text-muted">{i18n.t("onboarding.dashboard.pendingDecisionsEmpty")}</p>}
      {query.data && query.data.length > 0 && (
        <ul className="mt-4 divide-y divide-border" aria-label={i18n.t("onboarding.dashboard.pendingDecisionsList")}>
          {query.data.map((decision) => (
            <li key={decision.permissionId} className="flex flex-wrap items-center justify-between gap-3 py-3" data-testid="pending-decision-reference">
              <div>
                <p className="font-medium">{i18n.t("onboarding.dashboard.decisionUrgency", { source: decision.source, urgency: decision.urgency || "normal" })}</p>
                <p className="text-xs text-muted">{i18n.t("onboarding.dashboard.decisionRevision", { eventType: decision.sourceEventType, revision: decision.revision })}</p>
                <p className="text-xs text-muted">{i18n.t("onboarding.dashboard.decisionUpdated", { updatedAt: new Date(decision.updatedAt).toLocaleString(), deadline: decision.deadline ? i18n.t("onboarding.dashboard.decisionDue", { deadline: new Date(decision.deadline).toLocaleString() }) : "" })}</p>
              </div>
              <span className="rounded-full bg-warning/15 px-2.5 py-1 text-xs font-medium text-warning">{i18n.t("onboarding.dashboard.decisionPending")}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
