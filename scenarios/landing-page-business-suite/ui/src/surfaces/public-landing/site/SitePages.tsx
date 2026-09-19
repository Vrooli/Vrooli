import { useEffect, useId, useMemo } from 'react';
import { Link, useLocation, useSearchParams } from 'react-router-dom';
import { ArrowRight, Check, Compass, RotateCw } from 'lucide-react';
import { SiteShell } from './SiteShell';
import { useSiteIdentity } from './useSiteIdentity';
import { siteCopy } from './siteCopy';
import { Markdown } from './Markdown';
import { parseMarkdown, slugify } from './markdownParse';
import { defaultLegalMarkdown, renderLegalTokens, type LegalDocumentKind } from './legalTemplates';

type ThankYouKind = 'message' | 'refund' | 'checkout' | 'waitlist';

export function ThankYouPage() {
  const [params] = useSearchParams();
  const requested = params.get('type') ?? (params.get('checkout') === 'success' ? 'checkout' : 'message');
  const kind: ThankYouKind = (['message', 'refund', 'checkout', 'waitlist'] as const).find(value => value === requested) ?? 'message';
  const copy = siteCopy.thankYou;
  const identity = useSiteIdentity();
  const id = useId();
  return <SiteShell meta={{ title: copy.title, description: copy.description, noindex: true }} width="narrow" identity={identity}>
    <section className="site-card site-status-card" aria-labelledby={`${id}-heading`} data-thank-you={kind}>
      <span className="site-status-mark site-status-success" aria-hidden="true"><Check /></span>
      <p className="eyebrow">{copy.eyebrow}</p>
      <h1 id={`${id}-heading`}>{copy[kind].heading}</h1>
      <p className="site-lede">{copy[kind].body}</p>
      <div className="site-actions">
        {kind === 'checkout'
          ? <Link className="button button-primary" to="/auth/login">{copy.signIn}<ArrowRight aria-hidden="true" /></Link>
          : <Link className="button button-primary" to="/">{copy.home}<ArrowRight aria-hidden="true" /></Link>}
        {kind === 'checkout' && <Link className="button button-secondary" to="/">{copy.home}</Link>}
      </div>
      {identity.contactEmail && <p className="site-status-foot">{copy.questions} <a href={`mailto:${identity.contactEmail}`}>{identity.contactEmail}</a></p>}
    </section>
  </SiteShell>;
}

export function LegalPage({ kind }: { kind: LegalDocumentKind }) {
  const identity = useSiteIdentity();
  const copy = siteCopy.legal[kind];
  const id = useId();
  const { hash } = useLocation();
  const effectiveDate = kind === 'privacy' ? identity.privacyEffectiveDate : identity.termsEffectiveDate;
  const source = useMemo(() => renderLegalTokens((kind === 'privacy' ? identity.privacyMarkdown : identity.termsMarkdown) ?? defaultLegalMarkdown(kind), {
    businessName: identity.businessName, siteName: identity.brandName, contactEmail: identity.contactEmail,
    contactAddress: identity.addressLines.join('\n'), website: identity.website, effectiveDate,
  }), [kind, identity, effectiveDate]);
  const sections = useMemo(() => parseMarkdown(source).flatMap(block => block.kind === 'heading' && block.level === 2 ? [block.text] : []), [source]);
  const formattedDate = useMemo(() => {
    if (!effectiveDate) return undefined;
    const date = new Date(`${effectiveDate}T00:00:00`);
    return Number.isNaN(date.getTime()) ? effectiveDate : date.toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' });
  }, [effectiveDate]);
  useEffect(() => {
    if (!hash || identity.loading) return;
    document.getElementById(decodeURIComponent(hash.slice(1)))?.scrollIntoView();
  }, [hash, identity.loading]);
  return <SiteShell meta={{ title: copy.title, description: copy.description }} width="reading" identity={identity}>
    <header className="site-hero site-legal-hero">
      <p className="eyebrow">{siteCopy.legal.eyebrow}</p>
      <h1 id={`${id}-heading`}>{copy.title}</h1>
      {formattedDate && <p className="site-legal-meta">{siteCopy.legal.effective} <time dateTime={effectiveDate}>{formattedDate}</time></p>}
    </header>
    <div className="site-legal-layout">
      {sections.length > 2 && <nav className="site-toc" aria-label={siteCopy.legal.contents}>
        <p>{siteCopy.legal.contents}</p>
        <ol>{sections.map(section => <li key={section}><a href={`#${slugify(section)}`}>{section}</a></li>)}</ol>
      </nav>}
      <article className="site-prose" aria-labelledby={`${id}-heading`} aria-busy={identity.loading}>
        {identity.loading ? <div className="site-skeleton-stack" aria-hidden="true">{[88, 96, 72, 0, 40, 92, 84, 60].map((width, index) => width ? <span key={index} className="site-skeleton" style={{ width: `${String(width)}%` }} /> : <span key={index} className="site-skeleton-gap" />)}</div> : <Markdown source={source} />}
        <aside className="site-prose-foot">
          <p>{siteCopy.legal.questions}</p>
          <Link className="button button-secondary" to="/contact">{siteCopy.legal.contact}<ArrowRight aria-hidden="true" /></Link>
        </aside>
      </article>
    </div>
  </SiteShell>;
}

export function NotFoundPage() {
  const copy = siteCopy.notFound;
  const id = useId();
  useEffect(() => {
    const element = document.documentElement;
    element.dataset.experienceState = 'not-found';
    return () => { delete element.dataset.experienceState; };
  }, []);
  return <SiteShell meta={{ title: copy.title, description: copy.description, noindex: true }} width="narrow">
    <section className="site-status-card site-not-found" aria-labelledby={`${id}-heading`} data-testid="landing-experience-surface" data-experience-surface="public-landing" data-experience-state="not-found" data-http-status="404">
      <p className="site-status-code" aria-hidden="true">{copy.code}</p>
      <span className="site-status-mark" aria-hidden="true"><Compass /></span>
      <h1 id={`${id}-heading`}>{copy.heading}</h1>
      <p className="site-lede">{copy.body}</p>
      <div className="site-actions">
        <Link className="button button-primary" to="/">{copy.home}<ArrowRight aria-hidden="true" /></Link>
        <Link className="button button-secondary" to="/contact">{copy.contact}</Link>
      </div>
    </section>
  </SiteShell>;
}

export function UnavailablePage({ retry }: { retry?: () => void }) {
  const copy = siteCopy.unavailable;
  const id = useId();
  return <SiteShell meta={{ title: copy.title, description: copy.body, noindex: true }} width="narrow">
    <section className="site-status-card" aria-labelledby={`${id}-heading`} data-testid="landing-experience-surface" data-experience-surface="public-landing" data-experience-state="unavailable">
      <span className="site-status-mark" aria-hidden="true"><RotateCw /></span>
      <h1 id={`${id}-heading`}>{copy.heading}</h1>
      <p className="site-lede" role="status">{copy.body}</p>
      <div className="site-actions">
        {retry && <button type="button" className="button button-primary" onClick={retry}>{copy.retry}</button>}
        <Link className="button button-secondary" to="/contact">{copy.contact}</Link>
      </div>
    </section>
  </SiteShell>;
}
