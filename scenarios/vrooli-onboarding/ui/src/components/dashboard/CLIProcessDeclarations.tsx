import { useQuery } from "@tanstack/react-query";
import { listProcessSpawnDeclarations } from "../../api/cliManifestDeclarations";
import { i18n } from "../../i18n";

export function CLIProcessDeclarations() {
  const query = useQuery({ queryKey: ["cli-process-declarations"], queryFn: listProcessSpawnDeclarations });
  return (
    <section className="mt-8 rounded-xl border border-border bg-surface p-5" aria-labelledby="cli-process-declarations-heading" data-testid="cli-process-declarations">
      <p className="text-xs font-medium uppercase tracking-wide text-muted">{i18n.t("onboarding.dashboard.cliDeclarationsEyebrow")}</p>
      <h2 id="cli-process-declarations-heading" className="mt-1 text-lg font-semibold">{i18n.t("onboarding.dashboard.cliDeclarationsHeading")}</h2>
      <p className="mt-1 text-sm text-muted">{i18n.t("onboarding.dashboard.cliDeclarationsIntro")}</p>
      {query.isLoading && <p className="mt-4 text-sm text-muted" role="status">{i18n.t("onboarding.dashboard.cliDeclarationsLoading")}</p>}
      {query.error && <p className="mt-4 text-sm text-danger" role="alert">{i18n.t("onboarding.dashboard.cliDeclarationsError")}</p>}
      {query.data && <>
        <p className="mt-3 text-sm text-muted">{query.data.authorization_note}</p>
        <p className="mt-2 text-xs text-muted">{i18n.t("onboarding.dashboard.cliDeclarationsSource", { schema: query.data.schema_id, digest: query.data.source_digest.slice(0, 19) })}</p>
        {query.data.declarations.length === 0 ? <p className="mt-4 text-sm text-muted">{i18n.t("onboarding.dashboard.cliDeclarationsEmpty")}</p> : (
          <ul className="mt-3 divide-y divide-border" aria-label={i18n.t("onboarding.dashboard.cliDeclarationsList")}>
            {query.data.declarations.map((item) => <li key={`${item.manifest}:${item.command}`} className="py-2" data-testid="cli-process-declaration">
              <p className="font-medium">{item.declaring_owner} · {item.command}</p>
              <p className="text-xs text-muted">{i18n.t("onboarding.dashboard.cliDeclarationsDetails", { effect: item.declared_effect, scope: item.derived_scope, process: item.declared_permissions.join(", ") || i18n.t("onboarding.dashboard.cliDeclarationsNone"), eligibility: i18n.t(item.declared_run_eligible ? "onboarding.dashboard.cliDeclarationsEligible" : "onboarding.dashboard.cliDeclarationsNotEligible") })}</p>
            </li>)}
          </ul>
        )}
      </>}
    </section>
  );
}
