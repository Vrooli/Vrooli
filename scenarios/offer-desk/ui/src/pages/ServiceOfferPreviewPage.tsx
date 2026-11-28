import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1.3.1";
import { fetchServiceOfferPreview, serviceOfferSampleUrl } from "../api/serviceoffer";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const money = (minor: number, currency: string) => new Intl.NumberFormat(undefined, { style: "currency", currency }).format(minor / 100);

export function ServiceOfferPreviewPage() {
  const { t } = useTranslation();
  const query = useQuery({ queryKey: ["service-offer-proposal-preview"], queryFn: fetchServiceOfferPreview, retry: false });
  if (query.isLoading) return <main className="mx-auto grid w-full max-w-3xl gap-4 p-4 sm:p-6" aria-busy="true"><p role="status">{t(strings.pages.serviceOfferPreview.loading)}</p></main>;
  if (query.isError || !query.data) return <main className="mx-auto grid w-full max-w-3xl gap-4 p-4 sm:p-6"><h1 className="text-2xl font-bold">{t(strings.pages.serviceOfferPreview.previewTitle)}</h1><p role="alert">{t(strings.pages.serviceOfferPreview.unavailable)}</p></main>;

  const { offer, surfaces } = query.data;
  return <main className="mx-auto grid w-full max-w-3xl gap-4 p-4 pb-8 sm:gap-6 sm:p-6">
    <header className="grid gap-3 rounded-2xl border border-app-border bg-app-surface p-5 sm:p-7">
      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-app-muted-foreground">{t(strings.pages.serviceOfferPreview.eyebrow)}</p>
      <h1 className="text-3xl font-bold leading-tight sm:text-4xl">{offer.name}</h1>
      <p className="max-w-2xl text-base leading-relaxed text-app-muted-foreground">{offer.summary}</p>
      <div className="flex flex-wrap items-end gap-x-6 gap-y-2 border-t border-app-border pt-4">
        <div><p className="text-xs uppercase tracking-wide text-app-muted-foreground">{t(strings.pages.serviceOfferPreview.proposedPrice)}</p><p className="text-2xl font-bold">{money(offer.proposed_price.amount_minor, offer.proposed_price.currency)} <span className="text-sm font-normal text-app-muted-foreground">{t(strings.pages.serviceOfferPreview.recommendationOnly)}</span></p></div>
        <div><p className="text-xs uppercase tracking-wide text-app-muted-foreground">{t(strings.pages.serviceOfferPreview.deliveryTarget)}</p><p className="text-lg font-semibold">{offer.delivery.target_minutes} {t(strings.pages.serviceOfferPreview.minutes)}</p></div>
      </div>
      <p className="rounded-lg bg-app-muted p-3 text-sm">{t(strings.pages.serviceOfferPreview.pendingNote)}</p>
    </header>

    <Card>
      <CardHeader><CardTitle>{t(strings.pages.serviceOfferPreview.freeSample)}</CardTitle><CardDescription>{t(strings.pages.serviceOfferPreview.freeSampleDescription)}</CardDescription></CardHeader>
      <CardContent><a className="inline-flex min-h-11 items-center rounded-lg border border-app-border px-4 font-medium underline underline-offset-4" href={serviceOfferSampleUrl()} target="_blank" rel="noreferrer">{t(strings.pages.serviceOfferPreview.sampleLink)}</a></CardContent>
    </Card>

    <Card>
      <CardHeader><CardTitle>{t(strings.pages.serviceOfferPreview.purchaseTitle)}</CardTitle><CardDescription>{offer.verification_promise}</CardDescription></CardHeader>
      <CardContent className="grid gap-3 text-sm leading-relaxed">
        <p>{t(strings.pages.serviceOfferPreview.checkoutMode, { mode: surfaces.checkout.mode, paymentMode: surfaces.checkout.payment_mode, captureWhen: surfaces.checkout.capture_when })}</p>
        <p><strong>{t(strings.pages.serviceOfferPreview.refundRule)}:</strong> {offer.refund_rule}</p>
        <button type="button" disabled aria-disabled="true" className="min-h-11 w-full rounded-lg bg-app-muted px-4 font-semibold text-app-muted-foreground">{t(strings.pages.serviceOfferPreview.purchaseDisabled)}</button>
      </CardContent>
    </Card>

    <Card>
      <CardHeader><CardTitle>{t(strings.pages.serviceOfferPreview.agentDoors)}</CardTitle><CardDescription>{t(strings.pages.serviceOfferPreview.agentDoorsDescription)}</CardDescription></CardHeader>
      <CardContent className="grid gap-3 sm:grid-cols-2">
        <div className="rounded-lg border border-app-border p-3"><p className="font-semibold">{t(strings.pages.serviceOfferPreview.mcpTool)}</p><p className="break-words text-sm text-app-muted-foreground">{surfaces.mcp.name} · {surfaces.mcp.payment_path}</p></div>
        <div className="rounded-lg border border-app-border p-3"><p className="font-semibold">{t(strings.pages.serviceOfferPreview.mppX402)}</p><p className="break-words text-sm text-app-muted-foreground">{surfaces.mpp.mode} · {surfaces.mpp.path}</p></div>
        <div className="rounded-lg border border-app-border p-3"><p className="font-semibold">{t(strings.pages.serviceOfferPreview.plugin)}</p><p className="text-sm text-app-muted-foreground">{t(strings.pages.serviceOfferPreview.pluginDescription, { disposition: surfaces.plugin.disposition })}</p></div>
        <div className="rounded-lg border border-app-border p-3"><p className="font-semibold">{t(strings.pages.serviceOfferPreview.pageAndApis)}</p><p className="text-sm text-app-muted-foreground">{t(strings.pages.serviceOfferPreview.pageDescription, { state: surfaces.page.publication_state })}</p></div>
      </CardContent>
    </Card>

    <Card>
      <CardHeader><CardTitle>{t(strings.pages.serviceOfferPreview.reviewGates)}</CardTitle><CardDescription>{t(strings.pages.serviceOfferPreview.reviewDescription)}</CardDescription></CardHeader>
      <CardContent className="grid gap-2 text-sm">
        <p><strong>{t(strings.pages.serviceOfferPreview.legalStatus)}:</strong> {offer.legal.status} · {t(strings.pages.serviceOfferPreview.reviewed)} {offer.legal.reviewed_at}</p>
        {offer.legal.notes.map((note) => <p className="text-app-muted-foreground" key={note}>{note}</p>)}
        <p><strong>{t(strings.pages.serviceOfferPreview.operatorApproval)}:</strong> {offer.operator_approval}</p>
        <p className="text-app-muted-foreground">{t(strings.pages.serviceOfferPreview.publicEvidence, { count: offer.legal.evidence.length })} {t(strings.pages.serviceOfferPreview.disclaimer)}</p>
      </CardContent>
    </Card>
  </main>;
}
