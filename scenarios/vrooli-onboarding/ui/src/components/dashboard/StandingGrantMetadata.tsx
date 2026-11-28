import { useQuery } from "@tanstack/react-query";
import { listStandingGrantMetadata } from "../../api/standingGrants";
import { Button } from "@vrooli/react-component-library/Button/2";
import { i18n } from "../../i18n";

export function StandingGrantMetadata() {
  const query = useQuery({ queryKey: ["agent-manager-standing-grants"], queryFn: listStandingGrantMetadata, refetchInterval: 30_000 });
  return (
    <section className="mt-8 rounded-xl border border-border bg-surface p-5" aria-labelledby="standing-grants-heading" data-testid="standing-grant-metadata">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{i18n.t("onboarding.dashboard.standingGrantsEyebrow")}</p>
          <h2 id="standing-grants-heading" className="mt-1 text-lg font-semibold">{i18n.t("onboarding.dashboard.standingGrantsHeading")}</h2>
          <p className="mt-1 text-sm text-muted">{i18n.t("onboarding.dashboard.standingGrantsIntro")}</p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => void query.refetch()} disabled={query.isFetching}>{i18n.t("onboarding.dashboard.standingGrantsRefresh")}</Button>
      </div>
      {query.isLoading && <p className="mt-4 text-sm text-muted" role="status">{i18n.t("onboarding.dashboard.standingGrantsLoading")}</p>}
      {query.error && <p className="mt-4 text-sm text-danger" role="alert">{i18n.t("onboarding.dashboard.standingGrantsError")}</p>}
      {!query.isLoading && !query.error && query.data && query.data.grants.length === 0 && query.data.scopes.length === 0 && <p className="mt-4 text-sm text-muted">{i18n.t("onboarding.dashboard.standingGrantsEmpty")}</p>}
      {query.data && query.data.scopes.length > 0 && <div className="mt-4">
        <h3 className="text-sm font-semibold">{i18n.t("onboarding.dashboard.agentManagerScopeCatalog")}</h3>
        <ul className="mt-2 divide-y divide-border" aria-label={i18n.t("onboarding.dashboard.agentManagerScopeCatalog")}>
          {query.data.scopes.map((scope) => <li key={scope.permissionId} className="py-2" data-testid="agent-manager-scope">
            <p className="font-medium">{scope.scope}</p>
            <p className="text-xs text-muted">{i18n.t("onboarding.dashboard.agentManagerScopeDetails", { effect: scope.effect, command: scope.command, eligibility: scope.runEligible ? i18n.t("onboarding.dashboard.agentManagerRunEligible") : i18n.t("onboarding.dashboard.agentManagerNotRunEligible") })}</p>
          </li>)}
        </ul>
      </div>}
      {query.data && query.data.grants.length > 0 && <div className="mt-4">
        <h3 className="text-sm font-semibold">{i18n.t("onboarding.dashboard.standingGrantsList")}</h3>
        <ul className="mt-2 divide-y divide-border" aria-label={i18n.t("onboarding.dashboard.standingGrantsList")}>
        {query.data.grants.map((grant) => <li key={grant.permissionId} className="py-3" data-testid="standing-grant-row">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="font-medium">{grant.subject}</p>
            <span className="rounded-full bg-surface-muted px-2.5 py-1 text-xs font-medium">{grant.state}</span>
          </div>
          <p className="mt-1 text-xs text-muted">Actions: {grant.operations.join(", ")} · Targets: {grant.targets.join(", ")}</p>
          <p className="text-xs text-muted">Approved by {grant.approver} · Expires {new Date(grant.expiresAt).toLocaleString()}</p>
          <p className="text-xs text-muted">Limits: {grant.maxConcurrentRuns} concurrent runs · ${grant.maxRunBudgetUsd} per run</p>
        </li>)}
      </ul>
      </div>}
    </section>
  );
}
