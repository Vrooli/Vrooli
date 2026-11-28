import { useQuery } from "@tanstack/react-query";
import { listApprovedDependencies } from "../../api/approvedDependencies";
import { Button } from "@vrooli/react-component-library/Button/2";
import { i18n } from "../../i18n";

export function ApprovedDependencies() {
  const query = useQuery({ queryKey: ["approved-dependency-registry"], queryFn: listApprovedDependencies });
  return (
    <section className="mt-8 rounded-xl border border-border bg-surface p-5" aria-labelledby="approved-dependencies-heading" data-testid="approved-dependencies">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{i18n.t("onboarding.dashboard.approvedDependenciesEyebrow")}</p>
          <h2 id="approved-dependencies-heading" className="mt-1 text-lg font-semibold">{i18n.t("onboarding.dashboard.approvedDependenciesHeading")}</h2>
          <p className="mt-1 text-sm text-muted">{i18n.t("onboarding.dashboard.approvedDependenciesIntro")}</p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => void query.refetch()} disabled={query.isFetching}>{i18n.t("onboarding.dashboard.approvedDependenciesRefresh")}</Button>
      </div>
      {query.isLoading && <p className="mt-4 text-sm text-muted" role="status">{i18n.t("onboarding.dashboard.approvedDependenciesLoading")}</p>}
      {query.error && <p className="mt-4 text-sm text-danger" role="alert">{i18n.t("onboarding.dashboard.approvedDependenciesError")}</p>}
      {query.data && <>
        <p className="mt-3 text-xs text-muted">{i18n.t("onboarding.dashboard.approvedDependenciesSource", { version: query.data.version, mode: query.data.mode, digest: query.data.source_digest.slice(0, 19) })}</p>
        {query.data.guidance && <p className="mt-2 text-sm text-muted">{query.data.guidance}</p>}
        {query.data.records.length === 0 ? <p className="mt-4 text-sm text-muted">{i18n.t("onboarding.dashboard.approvedDependenciesEmpty")}</p> : (
          <ul className="mt-3 divide-y divide-border" aria-label={i18n.t("onboarding.dashboard.approvedDependenciesList")}>
            {query.data.records.map((record) => <li key={`${record.ecosystem}:${record.package_name}`} className="py-3" data-testid="approved-dependency-row">
              <div className="flex flex-wrap items-center justify-between gap-3"><p className="font-medium">{record.package_name}</p><span className="rounded-full bg-surface-muted px-2.5 py-1 text-xs font-medium">{record.state}</span></div>
              <p className="mt-1 text-xs text-muted">{record.ecosystem} · {record.version_range}{record.range_policy ? ` · ${record.range_policy}` : ""}</p>
              {record.rationale && <p className="mt-1 text-sm text-muted">{record.rationale}</p>}
              {(record.approved_by || record.review_expires) && <p className="mt-1 text-xs text-muted">{i18n.t("onboarding.dashboard.approvedDependenciesReview", { approvedBy: record.approved_by || i18n.t("onboarding.dashboard.approvedDependenciesUnknown"), expires: record.review_expires || i18n.t("onboarding.dashboard.approvedDependenciesNotSet") })}</p>}
            </li>)}
          </ul>
        )}
      </>}
    </section>
  );
}
