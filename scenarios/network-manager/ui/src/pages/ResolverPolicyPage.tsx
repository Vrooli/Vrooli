import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";

import {
  applyPolicyChange,
  diagnoseEncryptedDnsBypass,
  evaluatePolicySchedule,
  fetchEndpointDohGuidance,
  fetchPolicyProfiles,
  fetchResolverStatus,
  previewPolicyChange,
  previewUpstreams,
  rollbackPolicyChange,
  upsertPolicyProfile,
  type PolicyChange,
  type PolicyGuidanceReport,
  type PolicyProfile,
  type PolicyScheduleEvaluation,
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
  const [profileName, setProfileName] = useState("Kids");
  const [profileGroup, setProfileGroup] = useState("kids");
  const [profileStrength, setProfileStrength] = useState("strict");
  const [profileSchedule, setProfileSchedule] = useState("daily:20:00-07:00");
  const [profileOverride, setProfileOverride] = useState("parent_override");
  const [guidanceTarget, setGuidanceTarget] = useState("network");
  const [endpointPlatform, setEndpointPlatform] = useState("windows");
  const [endpointBrowser, setEndpointBrowser] = useState("chrome");
  const [endpointManagement, setEndpointManagement] = useState("group-policy");
  const [preview, setPreview] = useState<PolicyChange | undefined>();
  const [applied, setApplied] = useState<PolicyChange | undefined>();
  const [savedProfile, setSavedProfile] = useState<PolicyProfile | undefined>();
  const [scheduleEvaluation, setScheduleEvaluation] = useState<PolicyScheduleEvaluation | undefined>();
  const [guidanceReport, setGuidanceReport] = useState<PolicyGuidanceReport | undefined>();

  const { data, isLoading, error } = useQuery({
    queryKey: ["network", "resolver"],
    queryFn: fetchResolverStatus,
  });
  const profiles = useQuery({
    queryKey: ["network", "policy-profiles"],
    queryFn: () => fetchPolicyProfiles(),
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
  const profileSave = useMutation({
    mutationFn: () => upsertPolicyProfile({
      name: profileName,
      deviceGroup: profileGroup,
      filteringStrength: profileStrength,
      schedule: profileSchedule,
      overrideBehavior: profileOverride,
      status: "enabled",
    }),
    onSuccess: (profile) => {
      setSavedProfile(profile);
      void profiles.refetch();
    },
  });
  const scheduleCheck = useMutation({
    mutationFn: (profileId: string) => evaluatePolicySchedule(profileId, `group:${profileGroup}`),
    onSuccess: setScheduleEvaluation,
  });
  const bypassGuidance = useMutation({
    mutationFn: () => diagnoseEncryptedDnsBypass(guidanceTarget, false),
    onSuccess: setGuidanceReport,
  });
  const endpointGuidance = useMutation({
    mutationFn: () => fetchEndpointDohGuidance({
      platform: endpointPlatform,
      browser: endpointBrowser,
      managementMode: endpointManagement,
    }),
    onSuccess: setGuidanceReport,
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

      <section data-testid={selectors.network.policyProfiles} className={panelClass}>
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.resolver.profiles)}
        </h3>
        <div className="mt-3 grid gap-3 lg:grid-cols-5">
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.profileName)}
            <input className="rounded-control border border-app-border bg-app-background px-3 py-2" value={profileName} onChange={(event) => setProfileName(event.target.value)} />
          </label>
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.profileGroup)}
            <input className="rounded-control border border-app-border bg-app-background px-3 py-2" value={profileGroup} onChange={(event) => setProfileGroup(event.target.value)} />
          </label>
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.profileStrength)}
            <select className="rounded-control border border-app-border bg-app-background px-3 py-2" value={profileStrength} onChange={(event) => setProfileStrength(event.target.value)}>
              <option value="light">{t(strings.pages.resolver.strength.light)}</option>
              <option value="standard">{t(strings.pages.resolver.strength.standard)}</option>
              <option value="strict">{t(strings.pages.resolver.strength.strict)}</option>
              <option value="off">{t(strings.pages.resolver.strength.off)}</option>
            </select>
          </label>
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.profileSchedule)}
            <input className="rounded-control border border-app-border bg-app-background px-3 py-2" value={profileSchedule} onChange={(event) => setProfileSchedule(event.target.value)} />
          </label>
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.profileOverride)}
            <select className="rounded-control border border-app-border bg-app-background px-3 py-2" value={profileOverride} onChange={(event) => setProfileOverride(event.target.value)}>
              <option value="manual_required">{t(strings.pages.resolver.override.manualRequired)}</option>
              <option value="parent_override">{t(strings.pages.resolver.override.parentOverride)}</option>
              <option value="temporary_pause">{t(strings.pages.resolver.override.temporaryPause)}</option>
            </select>
          </label>
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          <button type="button" className={secondaryButtonClass} onClick={() => profileSave.mutate()}>
            {t(strings.pages.resolver.saveProfile)}
          </button>
          <button
            type="button"
            className={secondaryButtonClass}
            disabled={!savedProfile && !profiles.data?.[0]}
            onClick={() => scheduleCheck.mutate((savedProfile ?? profiles.data?.[0])?.id ?? "")}
          >
            {t(strings.pages.resolver.evaluateSchedule)}
          </button>
        </div>
        <ul className="mt-3 grid gap-2 text-sm md:grid-cols-2">
          {(profiles.data?.length ? profiles.data : savedProfile ? [savedProfile] : []).map((profile) => (
            <li key={profile.id} className="rounded-control bg-app-surface-muted p-3">
              <p className="font-semibold">{profile.name} · {profile.deviceGroup}</p>
              <p>{profile.filteringStrength} · {profile.schedule} · {profile.status}</p>
            </li>
          ))}
        </ul>
        {scheduleEvaluation && (
          <div className="mt-3 rounded-control bg-app-surface-muted p-3 text-sm">
            <p className="font-semibold">{t(strings.pages.resolver.scheduleEvaluation)}: {scheduleEvaluation.status}</p>
            <p>{scheduleEvaluation.effects.join("; ") || t(strings.network.none)}</p>
          </div>
        )}
      </section>

      <section data-testid={selectors.network.policyGuidance} className={panelClass}>
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.resolver.guidance)}
        </h3>
        <div className="mt-3 grid gap-3 lg:grid-cols-4">
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.guidanceTarget)}
            <input className="rounded-control border border-app-border bg-app-background px-3 py-2" value={guidanceTarget} onChange={(event) => setGuidanceTarget(event.target.value)} />
          </label>
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.endpointPlatform)}
            <input className="rounded-control border border-app-border bg-app-background px-3 py-2" value={endpointPlatform} onChange={(event) => setEndpointPlatform(event.target.value)} />
          </label>
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.endpointBrowser)}
            <input className="rounded-control border border-app-border bg-app-background px-3 py-2" value={endpointBrowser} onChange={(event) => setEndpointBrowser(event.target.value)} />
          </label>
          <label className="flex flex-col gap-2 text-sm">
            {t(strings.pages.resolver.managementMode)}
            <input className="rounded-control border border-app-border bg-app-background px-3 py-2" value={endpointManagement} onChange={(event) => setEndpointManagement(event.target.value)} />
          </label>
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          <button type="button" className={secondaryButtonClass} onClick={() => bypassGuidance.mutate()}>
            {t(strings.pages.resolver.bypassGuidance)}
          </button>
          <button type="button" className={secondaryButtonClass} onClick={() => endpointGuidance.mutate()}>
            {t(strings.pages.resolver.dohGuidance)}
          </button>
        </div>
        {guidanceReport && (
          <div className="mt-3 rounded-control bg-app-surface-muted p-3 text-sm">
            <p className="font-semibold">{guidanceReport.profile} · {guidanceReport.status}</p>
            <ul className="mt-2 list-disc space-y-1 ps-5">
              {guidanceReport.checks.map((check) => (
                <li key={check.id}>
                  <span className="font-medium">{check.title}</span>: {check.status}
                </li>
              ))}
              {guidanceReport.guardrails.map((guardrail) => (
                <li key={guardrail}>{guardrail}</li>
              ))}
            </ul>
          </div>
        )}
      </section>
    </section>
  );
}

function splitValues(value: string): string[] {
  return value.split(",").map((entry) => entry.trim()).filter(Boolean);
}
