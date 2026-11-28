import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertTriangle, CheckCircle2, RefreshCw, ShieldCheck, WifiOff } from 'lucide-react';
import { SiteShell } from '../site/SiteShell';
import { beginServiceOrder, getServiceOfferQuote, reconcileServiceOrder, serviceOrderResultURL, verifyCaptureServiceOrder, type ServiceOfferQuote, type ServiceOrder, type ServiceOrderDoor } from '../services/serviceOrders.service';

const META = { title: 'Service checkout', description: 'Review the service offer and submit a test mode order.', noindex: true };

function money(amountMinor: number, currency: string) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: currency.toUpperCase() }).format(amountMinor / 100);
}

function newIdempotencyKey() {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `checkout-${String(Date.now())}-${Math.random().toString(36).slice(2)}`;
}

export function ServiceCheckoutPage() {
  const [params] = useSearchParams();
  const offerID = params.get('offer_id')?.trim() ?? '';
  const offerVersion = Number(params.get('offer_version'));
  const doorParam = params.get('door')?.trim() || 'browser';
  const validDoor = doorParam === 'browser' || doorParam === 'browsing_agent';
  const door: ServiceOrderDoor = validDoor ? doorParam : 'browser';
  const validIdentity = Boolean(offerID) && Number.isInteger(offerVersion) && offerVersion > 0;
  const validCheckout = validIdentity && validDoor;
  const [quote, setQuote] = useState<ServiceOfferQuote | null>(null);
  const [quoteError, setQuoteError] = useState('');
  const [loading, setLoading] = useState(true);
  const [request, setRequest] = useState('');
  const [authorize, setAuthorize] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [actionBusy, setActionBusy] = useState(false);
  const [submitError, setSubmitError] = useState('');
  const [order, setOrder] = useState<ServiceOrder | null>(null);
  const [retryQuote, setRetryQuote] = useState(0);
  const idempotencyRef = useRef<{ input: string; key: string } | null>(null);
  const requestRef = useRef(0);
  const orderAction = order?.state === 'ledger_pending'
    ? 'reconcile'
    : order && order.state !== 'reconciled' && order.state !== 'voided' && order.state !== 'declined'
      ? 'verify'
      : null;

  useEffect(() => {
    const id = ++requestRef.current;
    if (!validCheckout) {
      setLoading(false);
      setQuoteError(validIdentity
        ? 'This checkout channel is not supported. Return to the service page and choose a supported test channel.'
        : 'This checkout link is missing a valid offer. Return to the service page and open checkout again.');
      return;
    }
    setLoading(true);
    setQuoteError('');
    setQuote(null);
    getServiceOfferQuote(offerID, offerVersion)
      .then((result) => {
        if (requestRef.current !== id) return;
        if (result.offer_id !== offerID || result.offer_version !== offerVersion
          || !Number.isSafeInteger(result.amount_minor) || result.amount_minor < 1
          || !/^[A-Z]{3}$/i.test(result.currency)) {
          throw new Error('The offer terms could not be verified. Please reopen checkout from the service page.');
        }
        setQuote(result);
      })
      .catch((error: unknown) => {
        if (requestRef.current === id) setQuoteError(error instanceof Error ? error.message : 'The offer is unavailable. Please try again.');
      })
      .finally(() => { if (requestRef.current === id) setLoading(false); });
    return () => { if (requestRef.current === id) requestRef.current += 1; };
  }, [offerID, offerVersion, retryQuote, validCheckout, validIdentity]);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!quote || !request.trim() || submitting) return;
    setSubmitting(true);
    setSubmitError('');
    setOrder(null);
    const trimmedInput = request.trim();
    const idempotencyInput = `${door}\n${trimmedInput}`;
    if (!idempotencyRef.current || idempotencyRef.current.input !== idempotencyInput) {
      idempotencyRef.current = { input: idempotencyInput, key: newIdempotencyKey() };
    }
    try {
      const result = await beginServiceOrder({
        offer_id: quote.offer_id,
        offer_version: quote.offer_version,
        door,
        idempotency_key: idempotencyRef.current.key,
        job_input: { sector: trimmedInput },
      });
      setOrder(result);
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'We could not start the order. You can safely retry.');
    } finally {
      setSubmitting(false);
    }
  }

  async function advanceOrder() {
    if (!order || !orderAction || actionBusy) return;
    setActionBusy(true);
    setSubmitError('');
    try {
      const updated = orderAction === 'verify'
        ? await verifyCaptureServiceOrder(order.order_id)
        : await reconcileServiceOrder(order.order_id);
      if (updated.order_id !== order.order_id || updated.job_id !== order.job_id) {
        throw new Error('The order response did not match this request. Check order status before retrying.');
      }
      setOrder(updated);
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'The test order step did not finish. You can safely retry.');
    } finally {
      setActionBusy(false);
    }
  }

  return (
    <SiteShell meta={META} width="narrow" chrome="minimal">
      <main className="site-service-checkout" aria-labelledby="service-checkout-title">
        <div className="site-service-checkout-banner" role="note"><ShieldCheck aria-hidden="true" />Test mode · no live payment</div>
        <header className="site-hero">
          <p className="eyebrow">Service order</p>
          <h1 id="service-checkout-title">Review and continue</h1>
          <p className="site-lede">Your request is reviewed before any test capture. The final amount comes from the current service offer.</p>
        </header>

        <section className="site-service-checkout-channel" aria-label="Selected test checkout channel">
          <span>Selected test channel</span>
          <strong>{!validDoor ? 'Unsupported channel' : door === 'browsing_agent' ? 'Browsing agent' : 'Person in browser'}</strong>
          <small>This records the requested checkout door only. It does not verify or assert an agent identity.</small>
        </section>

        {loading ? <section className="site-card" aria-busy="true"><p role="status">Loading current offer terms…</p></section> : null}
        {!loading && quoteError ? <section className="site-card site-service-checkout-error" role="alert">
          <AlertTriangle aria-hidden="true" /><h2>Checkout unavailable</h2><p>{quoteError}</p>
          {validCheckout && <button className="button button-primary" type="button" onClick={() => { setRetryQuote((value) => value + 1); }}><RefreshCw aria-hidden="true" />Try again</button>}
        </section> : null}

        {!loading && quote && <>
          <section className="site-card site-service-checkout-summary" aria-label="Offer summary">
            <p className="site-checkout-kicker">Offer {quote.offer_id} · v{quote.offer_version}</p>
            <p className="site-service-checkout-price">{money(quote.amount_minor, quote.currency)}</p>
            <p className="site-service-checkout-subtle">One-time test authorization. Capture happens only after delivery verification.</p>
          </section>

          {order ? <section className="site-card site-service-checkout-success" aria-live="polite" role="status">
            <CheckCircle2 aria-hidden="true" /><h2>{order.state === 'reconciled' ? 'Test order complete' : 'Request received'}</h2>
            <p>Order <strong>{order.order_id}</strong> for job <strong>{order.job_id}</strong> is in <strong>{order.state}</strong> state. No live payment occurred.</p>
            {order.state === 'reconciled' ? <p className="site-service-checkout-receipt">Accepted test receipt <strong>{order.ledger_receipt_id || 'pending ID'}</strong> · posting <strong>{order.ledger_posting_id || 'pending ID'}</strong></p> : <p className="site-service-checkout-subtle">Continue through delivery verification and test-ledger reconciliation.</p>}
            {order.state === 'reconciled' && order.offer_id === 'company-market-scan' && order.offer_version === 1 && <p className="site-service-checkout-receipt">This Markdown test result is bound to this order and job. It contains a prepared sample fixture, not custom research: <a href={serviceOrderResultURL(order.order_id)} download>Download your order result</a></p>}
            {submitError && <p className="site-service-checkout-inline-error" role="alert"><WifiOff aria-hidden="true" />{submitError}</p>}
          {orderAction && <button className="button button-primary site-service-checkout-submit" type="button" disabled={actionBusy} onClick={() => { void advanceOrder(); }}>
              {actionBusy ? <><span className="site-spinner" aria-hidden="true" />Updating test order…</> : orderAction === 'verify' ? 'Verify delivery and capture test authorization' : 'Reconcile accepted Money Ledger receipt'}
            </button>}
            <p className="site-service-checkout-subtle">Keep this order ID for status checks. Test receipts record no movement of funds.</p>
          </section> : <form className="site-card site-service-checkout-form" onSubmit={(event) => { void submit(event); }}>
            <label htmlFor="service-request">Market or vendor category</label>
            <textarea id="service-request" name="sector" rows={3} maxLength={500} required value={request} onChange={(event) => { setRequest(event.target.value); }} aria-describedby="service-request-help" />
            <p id="service-request-help" className="site-service-checkout-subtle">Share only information you are authorized to provide. Do not include passwords or payment details.</p>
            {submitError && <p className="site-service-checkout-inline-error" role="alert"><WifiOff aria-hidden="true" />{submitError}</p>}
            <label className="site-service-checkout-consent">
              <input type="checkbox" checked={authorize} onChange={(event) => { setAuthorize(event.target.checked); }} />
              <span>I understand this creates a test-only authorization record for {money(quote.amount_minor, quote.currency)}. No funds move; a test capture is recorded only after delivery verification.</span>
            </label>
            <button className="button button-primary site-service-checkout-submit" type="submit" disabled={submitting || !request.trim() || !authorize}>
              {submitting ? <><span className="site-spinner" aria-hidden="true" />Starting test order…</> : `Create ${money(quote.amount_minor, quote.currency)} test authorization`}
            </button>
            <p className="site-service-checkout-subtle">No live payment is collected. Test checkout is available only when the service-order route is running in test mode.</p>
          </form>}
        </>}
      </main>
    </SiteShell>
  );
}
