import { useEffect } from 'react';
import { updateMetaTags } from '../../../shared/lib/seo';
import { NotFoundPage, UnavailablePage } from '../site/SitePages';
import { presentationSystemUi as copy } from './systemUi';
import './presentation.css';

export function PublicPresentationState({ state, retry }: { state: 'loading' | 'unavailable' | 'not-found'; retry?: () => void }) {
  const title = state === 'loading' ? copy.loading : state === 'not-found' ? copy.notFound : copy.unavailable;
  useEffect(() => {
    // Not-found and unavailable pages own their branded, noindex metadata;
    // updateMetaTags clears absent tags, so it must not run over theirs.
    if (state === 'loading') updateMetaTags({ title, noindex: true });
    document.querySelector('link[rel="canonical"]')?.remove();
    document.querySelector('meta[name="presentation-revision"]')?.remove();
    document.querySelector('meta[property="og:url"]')?.remove();
    document.documentElement.dataset.experienceState = state;
    document.documentElement.dataset.captureReady = 'false';
    for (const key of ['presentationRevision', 'presentationMode', 'presentationDigest', 'variantSlug', 'presentationRequestedRoute', 'presentationResolvedRoute', 'presentationRequestedVariant', 'presentationLocale']) Reflect.deleteProperty(document.documentElement.dataset, key);
    return () => { delete document.documentElement.dataset.experienceState; delete document.documentElement.dataset.captureReady; };
  }, [state, title]);
  if (state === 'not-found') return <NotFoundPage />;
  if (state === 'unavailable') return <UnavailablePage retry={retry} />;
  // Loading keeps the night canvas the page is about to paint, so the first
  // frame never flashes a different surface before the presentation arrives.
  return <main data-testid="landing-experience-surface" data-experience-surface="public-landing" data-experience-state={state} aria-busy="true" className="presentation-page theme-signal site-loading">
    <div className="site-loading-mark" role="status"><span className="site-spinner" aria-hidden="true" /><span className="sr-only">{title}</span></div>
  </main>;
}
