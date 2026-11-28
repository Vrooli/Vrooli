import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";

import { fetchSpendReceipt, requestInstance } from "../api/compute";
import { fetchProviderCatalogue } from "../api/provider-catalogue";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const REQUEST_KEY_STORAGE = "compute-manager.pending-spend-request";
const A2_CAP_MINOR = 709n;
type Receipt = NonNullable<Awaited<ReturnType<typeof fetchSpendReceipt>>["receipt"]>;
type ProviderCatalogue = Awaited<ReturnType<typeof fetchProviderCatalogue>>;

function timestampMillis(value?: { seconds: bigint; nanos: number }) {
  return value ? Number(value.seconds) * 1000 + value.nanos / 1_000_000 : Number.NaN;
}

function minor(value: bigint, currency: string) {
  const whole = value / 100n;
  const fraction = (value % 100n).toString().padStart(2, "0");
  return `${currency} ${whole}.${fraction}`;
}

function formatTimestamp(value?: { seconds: bigint; nanos: number }) {
  if (!value) return "—";
  return new Date(Number(value.seconds) * 1000 + value.nanos / 1_000_000).toLocaleString();
}

export function RequestCapacityPage() {
  const { t } = useTranslation();
  const [catalogue, setCatalogue] = useState<ProviderCatalogue>();
  const [catalogueError, setCatalogueError] = useState("");
  const [providerId, setProviderId] = useState("");
  const [region, setRegion] = useState("");
  const [size, setSize] = useState("");
  const [lifetime, setLifetime] = useState("3600");
  const [state, setState] = useState<"idle" | "submitting" | "success" | "error" | "validation">("idle");
  const [requestKey, setRequestKey] = useState(() => sessionStorage.getItem(REQUEST_KEY_STORAGE) ?? "");
  const [receipt, setReceipt] = useState<Receipt>();
  const [receiptMissing, setReceiptMissing] = useState(false);
  const hydratedSession = useRef(false);

  useEffect(() => {
    let active = true;
    void fetchProviderCatalogue().then((result) => {
      if (!active) return;
      setCatalogue(result);
      if (sessionStorage.getItem(REQUEST_KEY_STORAGE)) return;
      const first = result.providers[0];
      if (first) {
        const fact = result.readiness[first.id]?.prices[0];
        setProviderId(first.id);
        setSize(first.offers[0]?.id ?? fact?.size ?? "");
        setRegion(first.offers[0]?.regions[0] ?? fact?.region ?? "");
      }
    }).catch(() => { if (active) setCatalogueError("Provider catalogue/readiness is unavailable. No source can be requested until current facts load."); });
    return () => { active = false; };
  }, []);

  const loadReceipt = useCallback(async (key: string): Promise<Receipt | undefined> => {
    try {
      const response = await fetchSpendReceipt(key);
      setReceipt(response.receipt);
      setReceiptMissing(!response.receipt);
      if (response.receipt) {
        setProviderId(response.receipt.provider);
        setSize(response.receipt.size);
        setRegion(response.receipt.region);
      }
      return response.receipt;
    } catch {
      setReceipt(undefined);
      setReceiptMissing(true);
      return undefined;
    }
  }, []);

  useEffect(() => {
    if (hydratedSession.current) return;
    hydratedSession.current = true;
    if (requestKey) void loadReceipt(requestKey).then((loaded) => setState(loaded?.requestState === "fulfilled" ? "success" : "error"));
  }, [loadReceipt, requestKey]);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const lifetimeSeconds = Number(lifetime);
    if (!Number.isFinite(lifetimeSeconds) || lifetimeSeconds <= 0) {
      setState("validation");
      return;
    }
    const key = requestKey || `ui-${crypto.randomUUID()}`;
    setRequestKey(key);
    sessionStorage.setItem(REQUEST_KEY_STORAGE, key);
    setState("submitting");
    setReceiptMissing(false);
    try {
      await requestInstance({
        idempotencyKey: key,
        provider: providerId,
        region,
        size,
        lifetimeSeconds: BigInt(Math.floor(lifetimeSeconds)),
      });
      setState("success");
    } catch {
      setState("error");
    } finally {
      await loadReceipt(key);
    }
  };

  const startAnother = () => {
    sessionStorage.removeItem(REQUEST_KEY_STORAGE);
    setRequestKey("");
    setReceipt(undefined);
    setReceiptMissing(false);
    setState("idle");
  };

  const expiresAt = receipt?.expiresAt ? Number(receipt.expiresAt.seconds) * 1000 : 0;
  const hasPriceObservation = !!receipt?.sourceUrl && !!receipt.currency && !!receipt.sourceVersion && !!receipt.observedAt && !!receipt.expiresAt;
  const fresh = hasPriceObservation && expiresAt > Date.now();
  const receiptIsTerminal = receipt?.requestState === "fulfilled" || receipt?.requestState === "refused" || receipt?.requestState === "abandoned";
  const selectedProvider = catalogue?.providers.find((provider) => provider.id === providerId);
  const readiness = providerId ? catalogue?.readiness[providerId] : undefined;
  const offers = selectedProvider?.offers ?? [];
  const sizes = [...new Set([...offers.map((offer) => offer.id), ...(readiness?.prices.map((fact) => fact.size) ?? [])])];
  const sizeOffer = offers.find((offer) => offer.id === size);
  const regions = [...new Set([...(sizeOffer?.regions ?? []), ...(readiness?.prices.filter((fact) => fact.size === size).map((fact) => fact.region) ?? [])])];
  const price = readiness?.prices.find((fact) => fact.provider === providerId && fact.size === size && fact.region === region);
  const decentralizedSource = !!(selectedProvider?.billingFacts.network || selectedProvider?.billingFacts.effect_mode);
  const freshPrice = !!price && !!price.currency && !!price.taxBasis && !!price.sourceUrl && !!price.sourceVersion && timestampMillis(price.observedAt) <= Date.now() && timestampMillis(price.expiresAt) > Date.now() && price.serverMinorPerMonth >= 0n && price.ipv4MinorPerMonth >= 0n;
  const totalMinor = freshPrice && price ? price.serverMinorPerMonth + price.ipv4MinorPerMonth : undefined;
  const admissionReasons: string[] = [];
  if (!freshPrice) admissionReasons.push("No fresh persisted price fact matches this exact source, offer, and region; price is unknown, not zero.");
  if (readiness?.credentialStatus !== "configured") admissionReasons.push("Provider credential/readiness is incomplete.");
  if (!readiness?.purchaseEnabled) admissionReasons.push("Compute Manager has not enabled purchase for this source.");
  if (decentralizedSource && selectedProvider?.billingFacts.effect_mode !== "live") admissionReasons.push(selectedProvider?.billingFacts.blocked_reason || "The decentralized source effect gate is closed.");
  if (freshPrice && price && (price.currency !== "USD" || price.taxBasis !== "pre-tax")) admissionReasons.push("Currency or tax basis does not match the approved A2 cap.");
  if (totalMinor !== undefined && price?.currency === "USD" && price.taxBasis === "pre-tax" && totalMinor > A2_CAP_MINOR) admissionReasons.push("This exact quote is over the approved A2 monthly cap.");
  const canRequest = admissionReasons.length === 0 && !!selectedProvider && !!size && !!region && !requestKey && state !== "submitting";

  const selectProvider = (id: string) => {
    if (id === "adopted-vps") { setProviderId(id); return; }
    const next = catalogue?.providers.find((item) => item.id === id);
    if (!next) return;
    const firstFact = catalogue?.readiness[id]?.prices[0];
    setProviderId(id);
    setSize(next.offers[0]?.id ?? firstFact?.size ?? "");
    setRegion(next.offers[0]?.regions[0] ?? firstFact?.region ?? "");
  };

  return (
    <section data-testid={selectors.pages.request} aria-labelledby="request-heading" className="flex flex-col gap-space-md">
      <PageHeader headingId="request-heading" title={t(strings.pages.request.title)} description={t(strings.pages.request.description)} />
      <label className="grid max-w-2xl gap-space-xs" htmlFor="request-provider">Compute source
        <select id="request-provider" className="rounded border bg-surface px-3 py-2" value={providerId} onChange={(event) => selectProvider(event.target.value)} disabled={!!requestKey || state === "submitting" || !catalogue}>
          {(catalogue?.providers ?? []).map((provider) => <option key={provider.id} value={provider.id}>{provider.name}</option>)}
          <option value="adopted-vps">Adopt an existing VPS</option>
        </select>
      </label>
      {catalogueError && <p role="alert">{catalogueError}</p>}
      {providerId === "adopted-vps" ? <section className="grid max-w-2xl gap-2 rounded-card border border-border bg-surface p-space-md" data-testid="adopted-source">
        <h2 className="font-semibold">Adopt an existing VPS</h2>
        <p className="text-sm">Use the existing typed adoption flow to declare the total monthly amount and tax basis. An explicit zero means you declare that the provider charges nothing; unknown cost is never treated as zero.</p>
        <a className="underline" href="/adopt">Continue to adopted VPS and stored admission receipt</a>
      </section> : <>
      <form data-testid={selectors.pages.requestForm} onSubmit={(event) => void submit(event)} className="grid max-w-2xl gap-space-md rounded-card border border-border bg-surface p-space-md">
        <label className="grid gap-space-xs" htmlFor="request-region">{t(strings.pages.request.region)}<select id="request-region" className="rounded border bg-surface px-3 py-2" value={region} onChange={(event) => setRegion(event.target.value)} disabled={!!requestKey || state === "submitting" || regions.length === 0}>{regions.map((value) => <option key={value} value={value}>{value}</option>)}</select></label>
        <label className="grid gap-space-xs" htmlFor="request-size">{t(strings.pages.request.size)}<select id="request-size" className="rounded border bg-surface px-3 py-2" value={size} onChange={(event) => setSize(event.target.value)} disabled={!!requestKey || state === "submitting" || sizes.length === 0}>{sizes.map((value) => <option key={value} value={value}>{value}</option>)}</select></label>
        <label className="grid gap-space-xs" htmlFor="request-lifetime">{t(strings.pages.request.lifetime)}<Input id="request-lifetime" type="number" min="1" value={lifetime} onChange={(event) => setLifetime(event.target.value)} disabled={!!requestKey || state === "submitting"} /></label>
        {readiness && <section className="grid gap-2 rounded border p-3 text-sm" data-testid="catalogue-admission">
          <p>Readiness: <strong>{readiness.current.replace(/_/g, " ")}</strong> · credential {readiness.credentialStatus}</p>
          {freshPrice && price && totalMinor !== undefined ? <>
            <p>Offer {price.offerId} · {price.size} in {price.region}. Fresh stored catalogue observation (read-only): {minor(price.serverMinorPerMonth, price.currency)} server + {minor(price.ipv4MinorPerMonth, price.currency)} IPv4 = <strong>{minor(totalMinor, price.currency)}/month</strong>; {price.taxBasis}.</p>
            <p>Source: <a className="underline break-all" href={price.sourceUrl} target="_blank" rel="noreferrer">{price.sourceUrl}</a> · {price.sourceVersion}</p>
            <p>Observed {formatTimestamp(price.observedAt)} · expires {formatTimestamp(price.expiresAt)} · A2 cap USD 7.09/month before tax.</p>
          </> : <p role="status">Price unknown. No exact fresh catalogue observation matches this offer and region; no amount is shown as zero.</p>}
          {admissionReasons.length > 0 && <div role="status"><strong>Request unavailable:</strong><ul className="list-disc pl-5">{admissionReasons.map((reason) => <li key={reason}>{reason}</li>)}</ul></div>}
          {decentralizedSource && <p>Decentralized source remains unavailable unless the catalogue reports an open effect gate and fresh sourced price facts.</p>}
          <p className="text-xs text-muted">A request is re-priced and admitted against the server cap before any provider effect; this read-only catalogue display is not the stored admission receipt.</p>
        </section>}
        <p data-testid="cost-estimate" role="status" className="text-muted">{t(strings.pages.request.estimate)}</p>
        <div className="flex flex-wrap gap-2">
          <Button data-testid={selectors.pages.requestSubmit} type="submit" disabled={!canRequest || state === "success"} pending={state === "submitting"} pendingLabel={t(strings.pages.request.submitting)}>{t(strings.pages.request.submit)}</Button>
          {(state === "success" || (state === "error" && !!receipt && receiptIsTerminal)) && <Button type="button" onClick={startAnother}>{t(strings.pages.request.newRequest)}</Button>}
          {state === "error" && requestKey && !receipt && <Button type="button" onClick={() => void loadReceipt(requestKey)}>{t(strings.pages.request.retryReceipt)}</Button>}
        </div>
        <div data-testid={selectors.pages.requestStatus} aria-live="polite">
          {state === "success" && <p role="status">{t(strings.pages.request.success)} <a className="underline" href="/">{t(strings.pages.dashboard.inventoryTitle)}</a></p>}
          {state === "error" && <p role="alert">{t(strings.pages.request.error)}</p>}
          {state === "validation" && <p role="alert">{t(strings.pages.request.validation)}</p>}
          {receiptMissing && requestKey && <p className="text-sm">{t(strings.pages.request.receiptMissing)}</p>}
        </div>
      </form>
      {catalogue && !catalogue.providers.some((provider) => provider.billingFacts.network || provider.billingFacts.effect_mode) && <p role="status">No decentralized provider is registered in the current setup catalogue, so there is no price or effect-gate evidence to display.</p>}
      </>}
      {receipt && <article data-testid="spend-receipt" className="grid max-w-2xl gap-2 rounded-card border border-border bg-surface p-space-md" aria-labelledby="receipt-heading">
        <h2 id="receipt-heading" className="text-lg font-semibold">{t(strings.pages.request.receiptHeading)}</h2>
        <p>{t(strings.pages.request.receiptKey)}: <code>{receipt.idempotencyKey}</code></p>
        <p>{t(strings.pages.request.decision)}: <strong>{receipt.admissionDecision}</strong> · {t(strings.pages.request.quoteState)}: {receipt.quoteState}</p>
        <p>{t(strings.pages.request.requestState)}: {receipt.requestState || "unknown"}</p>
        {receipt.instanceId && <p>{t(strings.pages.request.instance)}: <code>{receipt.instanceId}</code></p>}
        <p>{t(strings.pages.request.providerOffer)}: {receipt.provider} / {receipt.offerId}</p>
        <p>{t(strings.pages.request.selection)}: {receipt.size} · {receipt.region} · {receipt.term} · IPv4 {receipt.ipv4 ? "included" : "not selected"}</p>
        {!hasPriceObservation && <p role="status">{t(strings.pages.request.priceUnavailable)}</p>}
        <dl className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-4 gap-y-1 text-sm">
          {hasPriceObservation && <>
          <dt>{t(strings.pages.request.serverCost)}</dt><dd>{minor(receipt.serverMinorPerMonth, receipt.currency)}</dd>
          <dt>{t(strings.pages.request.ipv4Cost)}</dt><dd>{minor(receipt.ipv4MinorPerMonth, receipt.currency)}</dd>
          <dt className="font-semibold">{t(strings.pages.request.totalCost)}</dt><dd className="font-semibold">{minor(receipt.totalMinorPerMonth, receipt.currency)}</dd>
          </>}
          <dt>{t(strings.pages.request.cap)}</dt><dd>{minor(receipt.capMonthlyMinor, receipt.capCurrency)}</dd>
          <dt>{t(strings.pages.request.taxBasis)}</dt><dd>{receipt.taxBasis} · cap {receipt.capTaxBasis}</dd>
        </dl>
        {(receipt.quoteReason || receipt.admissionReason) && <p><strong>{t(strings.pages.request.reason)}:</strong> {receipt.admissionReason || receipt.quoteReason}</p>}
        {hasPriceObservation && <>
          <p>{t(strings.pages.request.source)}: <a className="underline" href={receipt.sourceUrl} target="_blank" rel="noreferrer">{receipt.sourceUrl}</a></p>
          <p>{t(strings.pages.request.sourceVersion)}: <code>{receipt.sourceVersion}</code></p>
        </>}
        <p>{t(strings.pages.request.observed)}: {formatTimestamp(receipt.observedAt)} · {t(strings.pages.request.expires)}: {formatTimestamp(receipt.expiresAt)} · {t(hasPriceObservation ? (fresh ? strings.pages.request.fresh : strings.pages.request.expired) : strings.pages.request.unknown)}</p>
        <p className="text-sm text-muted">{t(receipt.requestState === "open" || receipt.requestState === "reserving" ? strings.pages.request.receiptPending : strings.pages.request.receiptNote)}</p>
      </article>}
    </section>
  );
}
