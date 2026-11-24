import { useQuery } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { listCredentialGrantMetadata } from "../../api/credentialGrants";
import { i18n } from "../../i18n";

export function SealedCredentialGrants() {
  const query = useQuery({ queryKey: ["sealed-credential-grant-metadata"], queryFn: listCredentialGrantMetadata, refetchInterval: 30_000 });
  return (
    <section className="mt-8 rounded-xl border border-border bg-surface p-5" aria-labelledby="credential-grants-heading" data-testid="sealed-credential-grants">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{i18n.t("onboarding.dashboard.credentialGrantsEyebrow")}</p>
          <h2 id="credential-grants-heading" className="mt-1 text-lg font-semibold">{i18n.t("onboarding.dashboard.credentialGrantsHeading")}</h2>
          <p className="mt-1 text-sm text-muted">{i18n.t("onboarding.dashboard.credentialGrantsIntro")}</p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => void query.refetch()} disabled={query.isFetching}>{i18n.t("onboarding.dashboard.refresh")}</Button>
      </div>
      {query.isLoading && <p className="mt-4 text-sm text-muted" role="status">{i18n.t("onboarding.dashboard.credentialGrantsLoading")}</p>}
      {query.error && <p className="mt-4 text-sm text-danger" role="alert">{i18n.t("onboarding.dashboard.credentialGrantsError")}</p>}
      {!query.isLoading && !query.error && query.data?.length === 0 && <p className="mt-4 text-sm text-muted">{i18n.t("onboarding.dashboard.credentialGrantsEmpty")}</p>}
      {query.data && query.data.length > 0 && (
        <ul className="mt-4 divide-y divide-border" aria-label={i18n.t("onboarding.dashboard.credentialGrantsList")}>
          {query.data.map((grant) => (
            <li key={grant.permissionId} className="py-3" data-testid="sealed-credential-grant-row">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="font-medium">{grant.logicalId} · {grant.field}</p>
                <span className="rounded-full bg-surface-muted px-2.5 py-1 text-xs font-medium">{grant.status}</span>
              </div>
              <p className="mt-1 text-xs text-muted">{i18n.t("onboarding.dashboard.credentialGrantTarget", { machine: grant.machineId, node: grant.nodeId, generation: grant.enrollmentGeneration })}</p>
              <p className="text-xs text-muted">{i18n.t("onboarding.dashboard.credentialGrantDetails", { grantClass: grant.grantClass, retention: grant.retention, generation: grant.generation, acked: grant.ackedGeneration })}</p>
              <p className="text-xs text-muted">{i18n.t("onboarding.dashboard.credentialGrantReceipt", { accepted: grant.receiptAccepted ? i18n.t("onboarding.dashboard.yes") : i18n.t("onboarding.dashboard.no"), at: grant.receiptAt ? new Date(grant.receiptAt).toLocaleString() : i18n.t("onboarding.dashboard.notAvailable"), purge: grant.purgeState || i18n.t("onboarding.dashboard.notAvailable") })}</p>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
