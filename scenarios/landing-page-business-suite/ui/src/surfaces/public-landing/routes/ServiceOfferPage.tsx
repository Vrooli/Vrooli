import { useEffect } from 'react';
import { Link, useHref } from 'react-router-dom';
import { ArrowRight, FileText, ShieldCheck } from 'lucide-react';
import { SiteShell } from '../site/SiteShell';

const offer = {
  id: 'company-market-scan',
  version: 1,
  title: 'Company Market Scan',
  description: 'A cited snapshot of public company and market information.',
  price: '$49.00 USD',
  schemaPath: '/services/company-market-scan.json',
};

export function ServiceOfferPage() {
  const metadataHref = useHref(offer.schemaPath);
  const sampleHref = useHref('/samples/company-market-scan.md');
  useEffect(() => {
    const url = new URL(metadataHref, window.location.origin).href;
    let link = document.head.querySelector<HTMLLinkElement>('link[rel="alternate"][type="application/ld+json"]');
    if (!link) {
      link = document.createElement('link');
      link.rel = 'alternate';
      link.type = 'application/ld+json';
      document.head.append(link);
    }
    link.href = url;
    return () => link?.remove();
  }, [metadataHref]);

  const checkout = `/service-checkout?offer_id=${offer.id}&offer_version=${offer.version}`;
  return (
    <SiteShell meta={{ title: offer.title, description: offer.description }} width="reading" chrome="minimal">
      <main className="site-service-checkout" aria-labelledby="company-market-scan-title">
        <div className="site-service-checkout-banner" role="note"><ShieldCheck aria-hidden="true" />Test showcase · no live payment</div>
        <header className="site-hero">
          <p className="eyebrow">One-time research service · Offer v{offer.version}</p>
          <h1 id="company-market-scan-title">{offer.title}</h1>
          <p className="site-lede">{offer.description} For small teams deciding whether to evaluate a market or vendor.</p>
        </header>

        <section className="site-card site-service-checkout-summary" aria-label="Offer terms">
          <p className="site-service-checkout-price">{offer.price}</p>
          <p>One market or vendor category. Enter one request of up to 500 characters.</p>
          <p>Receive one Markdown report with scope, public source links, reported facts, and unchecked areas. Target delivery: 60 minutes.</p>
          <p className="site-service-checkout-subtle">The test showcase returns a deterministic local fixture; it does not perform custom market research or collect a real payment.</p>
        </section>

        <section className="site-card" aria-labelledby="scan-trust-title">
          <h2 id="scan-trust-title">Sources, privacy, and terms</h2>
          <ul>
            <li>Every material factual claim must have a resolving public citation, with source dates and unchecked areas reported.</li>
            <li>Public business sources only. No personal contact data or private records. This is not investment, legal, accounting, or purchasing advice.</li>
            <li>No account or contact details are required. Do not submit passwords, payment data, personal contact data, or confidential information.</li>
            <li>Authorization is captured only after delivery verification. Failed citation validation releases authorization; a captured order is refundable if its report materially misses the stated scope.</li>
          </ul>
          <p className="site-service-checkout-subtle">The offer is staged for local test checkout. Legal review is open and operator approval is pending; live payment is unavailable.</p>
        </section>

        <nav aria-label="Offer actions" className="site-service-checkout-actions">
          <a className="button button-secondary" href={sampleHref}><FileText aria-hidden="true" />Read the free sample</a>
          <Link className="button button-primary" to={checkout}><ArrowRight aria-hidden="true" />Continue without an account</Link>
        </nav>
        <p className="site-service-checkout-subtle">Machine-readable offer: <a href={metadataHref}>JSON metadata</a></p>
      </main>
    </SiteShell>
  );
}
