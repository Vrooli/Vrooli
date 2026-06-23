import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";

import {
  applyPolicyChange,
  fetchResolverStatus,
  previewPolicyChange,
  previewUpstreams,
  rollbackPolicyChange,
  type PolicyChange,
} from "../api/network";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const panelClass = "rounded-panel border border-app-border bg-app-surface p-4";
const buttonClass = "rounded-control bg-app-primary px-3 py-2 text-sm font-medium text-app-primary-foreground";
const secondaryButtonClass = "rounded-control border border-app-border px-3 py-2 text-sm font-medium hover:bg-app-surface-muted";
const POLICY_ACTIONS = [
  { value: "allowlist", label: strings.pages.resolver.actions.allowlist },
  { value: "denylist", label: strings.pages.resolver.actions.denylist },
  { value: "blocklist", label: strings.pages.resolver.actions.blocklist },
  { value: "pause", label: strings.pages.resolver.actions.pause },
  { value: "resume", label: strings.pages.resolver.actions.resume },
] as const;

export function ResolverPolicyPage() {
  const { t } = useTranslation();
  const [upstreams, setUpstreams] = useState("https://dns.example.test/dns-query,tls://dns-alt.example.test");
  const [target, setTarget] = useState("all-devices");
  const [action, setAction] = useState("denylist");
  const [values, setValues] = useState("example.test");
  const [preview, setPreview] = useState<PolicyChange | undefined>();
  const [applied, setApplied] = useState<PolicyChange | undefined>();

  const { data, isLoading, error } = useQuery({
    queryKey: ["network", "resolver"],
    queryFn: fetchResolverStatus,
  });
  const upstreamPreview = useMutation({
    mutationFn: () => previewUpstreams(splitValues(upstreams)),
  });
  const policyPreview = useMutation({
    mutationFn: () => previewPolicyChange({ target, action, values: splitValues(values) }),
    onSuccess: setPreview,
  });
  const policyApply = useMutation({
    mutationFn: (id: string) => applyPolicyChange(id),
    onSuccess: setApplied,
  });
  const policyRollback = useMutation({
    mutationFn: (id: string) => rollbackPolicyChange(id),
    onSuccess: setApplied,
  });

  return (
    <section data-testid={selectors.pages.resolver} aria-labelledby="resolver-heading" className="flex flex-col gap-4">
      <div>
        <h2 id="resolver-heading" className="text-2xl font-semibold">
          {t(strings.pages.resolver.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.resolver.description)}</p>
      </div>

      {isLoading && <p data-testid={selectors.network.loading}>{t(strings.network.loading)}</p>}
      {error && <p data-testid={selectors.network.error}>{t(strings.network.error)}</p>}

      <div className="grid gap-4 xl:grid-cols-2">
        <section data-testid={selectors.network.resolverStatus} className={panelClass}>
          <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
            {t(strings.pages.dashboard.resolverStatus)}
          </h3>
          <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
            <div>
              <dt className="text-app-muted-foreground">{t(strings.network.backend)}</dt>
              <dd className="font-semibold">{data?.backend || t(strings.network.unknown)}</dd>
            </div>
            <div>
              <dt className="text-app-muted-foreground">{t(strings.network.status)}</dt>
              <dd className="font-semibold">{data?.status || t(strings.network.unknown)}</dd>
            </div>
            <div>
              <dt className="text-app-muted-foreground">{t(strings.network.filtering)}</dt>
              <dd className="font-semibold">
                {data?.filteringEnabled ? t(strings.network.enabled) : t(strings.network.disabled)}
              </dd>
            </div>
          </dl>
          <h4 className="mt-5 text-sm font-semibold uppercase text-app-muted-foreground">
            {t(strings.pages.resolver.upstreams)}
          </h4>
          <ul className="mt-2 list-disc space-y-1 ps-5 text-sm">
            {(data?.upstreams.length ? data.upstreams : [t(strings.network.none)]).map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </section>

        <section className={panelClass}>
          <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
            {t(strings.pages.resolver.previewUpstreams)}
          </h3>
          <label className="mt-3 flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.upstreams)}
            <input
              className="rounded-control border border-app-border bg-app-background px-3 py-2"
              value={upstreams}
              onChange={(event) => setUpstreams(event.target.value)}
            />
          </label>
          <button type="button" className={`${secondaryButtonClass} mt-3`} onClick={() => upstreamPreview.mutate()}>
            {t(strings.pages.resolver.previewUpstreams)}
          </button>
          {upstreamPreview.data && (
            <ul className="mt-3 list-disc space-y-1 ps-5 text-sm">
              {upstreamPreview.data.map((change) => <li key={change}>{change}</li>)}
            </ul>
          )}
        </section>
      </div>

      <section data-testid={selectors.network.confirmation} className={panelClass}>
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.resolver.confirmation)}
        </h3>
        <div className="mt-3 grid gap-3 lg:grid-cols-3">
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.policyTarget)}
            <input className="rounded-control border border-app-border bg-app-background px-3 py-2" value={target} onChange={(event) => setTarget(event.target.value)} />
          </label>
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.policyAction)}
            <select className="rounded-control border border-app-border bg-app-background px-3 py-2" value={action} onChange={(event) => setAction(event.target.value)}>
              {POLICY_ACTIONS.map((policyAction) => (
                <option key={policyAction.value} value={policyAction.value}>
                  {t(policyAction.label)}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.policyValues)}
            <input className="rounded-control border border-app-border bg-app-background px-3 py-2" value={values} onChange={(event) => setValues(event.target.value)} />
          </label>
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          <button type="button" className={secondaryButtonClass} onClick={() => policyPreview.mutate()}>
            {t(strings.pages.resolver.previewPolicy)}
          </button>
          <button
            type="button"
            className={buttonClass}
            disabled={!preview}
            onClick={() => preview && policyApply.mutate(preview.id)}
          >
            {t(strings.pages.resolver.approveApply)}
          </button>
          <button
            type="button"
            className={secondaryButtonClass}
            disabled={!applied}
            onClick={() => applied && policyRollback.mutate(applied.id)}
          >
            {t(strings.pages.resolver.rollback)}
          </button>
        </div>
        {!preview && <p className="mt-3 text-sm text-app-muted-foreground">{t(strings.pages.resolver.previewRequired)}</p>}
        {preview && (
          <div data-testid={selectors.network.policyPreview} className="mt-4 rounded-control bg-app-surface-muted p-3 text-sm">
            <p className="font-semibold">{preview.action} · {preview.status}</p>
            <p>{preview.effects.join("; ") || t(strings.network.none)}</p>
            <p>{t(strings.network.rollbackSupported)}: {String(preview.rollbackSupported)}</p>
          </div>
        )}
      </section>
    </section>
  );
}

function splitValues(value: string): string[] {
  return value.split(",").map((entry) => entry.trim()).filter(Boolean);
}
