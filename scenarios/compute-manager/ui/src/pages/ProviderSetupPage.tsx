import { useCallback, useEffect, useMemo, useState } from "react";
import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import { SetupService } from "@vrooli/proto-types/compute-manager/v1/setup/setup_pb";
import { BeginSetupStepRequestSchema, GetReadinessRequestSchema, ListProvidersRequestSchema, VerifyProviderRequestSchema } from "@vrooli/proto-types/compute-manager/v1/setup/setup_pb";
import { transport } from "../api/client";
import { akashSetupView } from "../features/provider-setup/akashReadiness";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const setup = createClient(SetupService, transport);
const minor = (value: bigint) => `${value / 100n}.${(value % 100n).toString().padStart(2, "0")}`;
const timestamp = (value?: { seconds: bigint; nanos: number }) => value ? new Date(Number(value.seconds) * 1000 + value.nanos / 1_000_000).toLocaleString() : "unknown";

export function ProviderSetupPage() {
  const { t } = useTranslation();
  const [providers, setProviders] = useState<Awaited<ReturnType<typeof setup.listProviders>>["providers"]>([]);
  const [readiness, setReadiness] = useState<Record<string, Awaited<ReturnType<typeof setup.getReadiness>>["readiness"]>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [working, setWorking] = useState<Record<string, boolean>>({});
  const [regions, setRegions] = useState<Record<string, string>>({});

  const refresh = useCallback(async (id: string) => {
    const result = await setup.getReadiness(create(GetReadinessRequestSchema, { providerId: id }));
    setReadiness((old) => ({ ...old, [id]: result.readiness }));
  }, []);

  useEffect(() => {
    let mounted = true;
    void setup.listProviders(create(ListProvidersRequestSchema)).then(async (result) => {
      if (!mounted) return;
      setProviders(result.providers);
      await Promise.all(result.providers.map((p) => refresh(p.id)));
    }).catch(() => { if (mounted) setErrors({ catalogue: "Provider setup is unavailable." }); });
    return () => { mounted = false; };
  }, [refresh]);

  async function beginStep(providerId: string, stepId: string) {
    const key = `${providerId}:${stepId}`;
    setWorking((old) => ({ ...old, [key]: true }));
    setErrors((old) => ({ ...old, [providerId]: "" }));
    try {
      await setup.beginSetupStep(create(BeginSetupStepRequestSchema, { providerId, stepId }));
      await refresh(providerId);
    } catch {
      setErrors((old) => ({ ...old, [providerId]: "Could not send this decision to Notification Hub. Try again when the service is available." }));
    } finally { setWorking((old) => ({ ...old, [key]: false })); }
  }

  async function verify(providerId: string, region: string, size: string) {
    setWorking((old) => ({ ...old, [`verify:${providerId}`]: true }));
    setErrors((old) => ({ ...old, [providerId]: "" }));
    try {
      const result = await setup.verifyProvider(create(VerifyProviderRequestSchema, { providerId, region, size }));
      setReadiness((old) => ({ ...old, [providerId]: result.readiness }));
    } catch {
      setErrors((old) => ({ ...old, [providerId]: "Verification could not complete. No provider changes were made." }));
    } finally { setWorking((old) => ({ ...old, [`verify:${providerId}`]: false })); }
  }

  const onboardingURL = useMemo(() => "/embedded/vrooli-onboarding/?target=local&step=credentials", []);

  return <main className="mx-auto flex w-full max-w-4xl flex-col gap-6 p-4 sm:p-8">
    <header><h1 className="text-2xl font-semibold">Provider setup</h1><p className="mt-2 text-sm text-slate-600">Complete supported setup steps here on your phone. Compute Manager records each decision; provider verification is separate and available only where a read-only check is configured.</p></header>
    {errors.catalogue && <p role="alert">{errors.catalogue}</p>}
    {providers.map((provider) => {
      const result = readiness[provider.id];
      const isAkash = provider.id === "akash";
      const akash = isAkash ? akashSetupView(provider, result) : undefined;
      return <section key={provider.id} className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900">
        <h2 className="text-lg font-semibold">{provider.name}</h2>
        <p className="mt-1 text-sm">Readiness: <strong>{(result?.current ?? "needs_console_steps").replace(/_/g, " ")}</strong></p>
        {akash && <dl className="mt-3 grid gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm dark:border-amber-900 dark:bg-amber-950/30" data-testid="akash-readiness">
          <div><dt className="inline font-medium">{t(strings.providerSetup.akash.network)} </dt><dd className="inline">{akash.network || t(strings.providerSetup.akash.notReported)}</dd></div>
          <div><dt className="inline font-medium">{t(strings.providerSetup.akash.effectMode)} </dt><dd className="inline">{akash.effectMode || t(strings.providerSetup.akash.notReported)}</dd></div>
          <div><dt className="inline font-medium">{t(strings.providerSetup.akash.signerStatus)} </dt><dd className="inline">{akash.signerStatus === "verified" ? t(strings.providerSetup.akash.signerVerified) : akash.signerStatus === "unverified" ? t(strings.providerSetup.akash.signerUnverified) : akash.signerStatus === "unreported" ? t(strings.providerSetup.akash.signerUnverified) : akash.signerStatus}</dd></div>
          <div><dt className="inline font-medium">{t(strings.providerSetup.akash.priceStatus)} </dt><dd className="inline">{akash.priceStatus === "fresh_sourced" ? t(strings.providerSetup.akash.priceFresh) : akash.priceStatus === "stale_or_invalid" ? t(strings.providerSetup.akash.priceStale) : akash.priceStatus === "no_fact" ? t(strings.providerSetup.akash.priceMissing) : akash.priceStatus}</dd></div>
          <div><dt className="inline font-medium">{t(strings.providerSetup.akash.blockedReason)} </dt><dd className="inline">{akash.blockedReason || t(strings.providerSetup.akash.blockedReasonMissing)}</dd></div>
          {!akash.setupActionsEnabled && <p className="font-medium">{t(strings.providerSetup.akash.actionsDisabled)}</p>}
        </dl>}
        <div className="mt-3 text-sm"><strong>Billing guidance</strong><ul className="list-disc pl-5">{Object.entries(provider.billingFacts).filter(([key]) => key !== "source").map(([key, value]) => <li key={key}>{key.replace(/_/g, " ")}: {value}</li>)}</ul>{provider.billingFacts.source && <a className="underline" href={provider.billingFacts.source} target="_blank" rel="noreferrer">Billing source</a>}</div>
        <div className="mt-3 text-sm"><strong>Catalogue offers</strong>{provider.offers.map((offer) => <p key={offer.id}>{offer.label} · {offer.cpu} · {offer.memory}{offer.regions.length ? ` · ${offer.regions.join(", ")}` : ""}</p>)}</div>
        <ol className="my-4 list-decimal space-y-4 pl-5">
          {provider.consoleSteps.map((step) => {
            const state = result?.evidence.find((e) => e.id === `console:${step.id}`);
            const active = result?.activeStep === step.id;
            return <li key={step.id}>
              <h3 className="font-medium">{step.title}</h3>
              <p className="text-sm text-slate-600">{step.instructions}</p>
              <a className="text-sm underline" href={step.url} target={isAkash ? undefined : "_blank"} rel={isAkash ? undefined : "noreferrer"}>{isAkash ? t(strings.providerSetup.akash.secureEntryAction) : "Open provider website"}</a>
              <p className="mt-1 text-sm">{state?.verified ? "Confirmed" : active ? "Waiting for your phone decision" : "Not confirmed"}</p>
              {step.id === "token" && onboardingURL && <p className="mt-1"><a className="text-sm underline" href={onboardingURL}>Enter the token in Vrooli Onboarding</a></p>}
              {!state?.verified && <button disabled={working[`${provider.id}:${step.id}`]} className="mt-2 rounded-md border px-3 py-1.5 text-sm disabled:opacity-60" onClick={() => void beginStep(provider.id, step.id)}>{working[`${provider.id}:${step.id}`] ? "Sending…" : active ? "Resend phone decision" : "Ask me on my phone"}</button>}
              <p className="mt-1 text-sm text-slate-600">{step.decision}</p>
            </li>;
          })}
        </ol>
        {isAkash && <div className="mt-4 rounded-md border border-slate-200 p-3 text-sm" data-testid="akash-wallet-entry">
          <p><strong>{t(strings.providerSetup.akash.walletDescriptor)}:</strong> {provider.billingFacts.credential_reference || `${provider.credentialLogicalId}/${provider.credentialField}`}</p>
          <p>{t(strings.providerSetup.akash.walletEntryStatus)}: {result?.credentialStatus ?? "checking"}</p>
          <p>{t(strings.providerSetup.akash.walletIdentifierCaveat)}</p>
          <a className="mt-2 inline-block underline" href={onboardingURL}>{t(strings.providerSetup.akash.secureEntryAction)}</a>
        </div>}
        {!isAkash && <>
          <p className="text-sm"><strong>Credential descriptor:</strong> {provider.credentialLabel} ({provider.credentialField})</p>
          <p className="text-sm"><strong>Required scopes:</strong> {provider.credentialScopes.join(", ")}</p>
          <p className="text-sm">Credential status: {result?.credentialStatus ?? "checking"} · <a className="underline" href={onboardingURL}>Open secure Vrooli Onboarding step</a></p>
        </>}
        <div className="mt-4 flex flex-wrap gap-2">
          {provider.offers[0]?.regions.length ? provider.offers[0].regions.map((region) => <button key={region} disabled={working[`verify:${provider.id}`] || (isAkash && !akash?.readOnlyProbeEnabled)} className="rounded-md bg-blue-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-60" onClick={() => void verify(provider.id, region, provider.offers[0]?.id ?? "")}>Verify {provider.offers[0]?.id} in {region}</button>) : isAkash ? <p className="text-sm text-slate-600">{t(strings.providerSetup.akash.noOffer)}</p> : <>
            <label className="flex items-center gap-2 text-sm">Region <input className="rounded border px-2 py-1 text-slate-900" value={regions[provider.id] ?? ""} onChange={(event) => setRegions((old) => ({ ...old, [provider.id]: event.target.value.trim() }))} placeholder="Enter provider region" /></label>
            <button disabled={working[`verify:${provider.id}`] || !regions[provider.id]} className="rounded-md bg-blue-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-60" onClick={() => void verify(provider.id, regions[provider.id] ?? "", provider.offers[0]?.id ?? "")}>Verify {provider.offers[0]?.id} in selected region</button>
          </>}
        </div>
        {errors[provider.id] && <p role="alert" className="mt-2 text-sm text-red-700">{errors[provider.id]}</p>}
        {isAkash && !result?.prices.length && <p className="mt-4 text-sm" data-testid="akash-no-price">{t(strings.providerSetup.akash.noPriceFact)}</p>}
        {result?.prices.map((price) => <div key={`${price.region}:${price.size}`} className="mt-4 rounded-md bg-slate-50 p-3 text-sm dark:bg-slate-800">
          <p><strong>{isAkash ? akash?.priceStatus === "fresh_sourced" ? t(strings.providerSetup.akash.priceFresh) : akash?.priceStatus === "stale_or_invalid" ? t(strings.providerSetup.akash.priceStale) : akash?.priceStatus : price.expiresAt && Date.now() >= Number(price.expiresAt.seconds) * 1000 ? "Expired observation" : "Fresh observed price"}:</strong> {minor(price.serverMinorPerMonth + price.ipv4MinorPerMonth)} {price.currency} / month before tax</p>
          <p>{price.size} in {price.region} · IPv4 {minor(price.ipv4MinorPerMonth)} {price.currency} · observed {timestamp(price.observedAt)}</p>
          <p>Source: <a className="underline" href={price.sourceUrl} target="_blank" rel="noreferrer">{price.sourceUrl}</a> · {price.sourceVersion}</p>
        </div>)}
        <p className="mt-3 text-sm">Purchase remains disabled. A phone confirmation records setup only; it never approves spending.</p>
      </section>;
    })}
    {!errors.catalogue && !providers.some((provider) => provider.id === "akash") && <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900" data-testid="akash-not-listed">
      <h2 className="text-lg font-semibold">{t(strings.providerSetup.akash.title)}</h2>
      <p className="mt-2 text-sm">{t(strings.providerSetup.akash.descriptorMissing)}</p>
      <p className="mt-2 text-sm font-medium">{t(strings.providerSetup.akash.noActions)}</p>
    </section>}
  </main>;
}
