import { useEffect } from 'react';
import { updateMetaTags } from '../../../shared/lib/seo';
import { presentationSystemUi as copy } from './systemUi';

export function PublicPresentationState({ state, retry }: { state: 'loading' | 'unavailable' | 'not-found'; retry?: () => void }) {
  const title = state === 'loading' ? copy.loading : state === 'not-found' ? copy.notFound : copy.unavailable;
  useEffect(() => {
    updateMetaTags({ title, noindex: true });
    document.querySelector('link[rel="canonical"]')?.remove();
    document.querySelector('meta[name="presentation-revision"]')?.remove();
    document.querySelector('meta[property="og:url"]')?.remove();
    document.documentElement.dataset.experienceState = state;
    document.documentElement.dataset.captureReady = 'false';
    for (const key of ['presentationRevision', 'presentationMode', 'presentationDigest', 'variantSlug', 'presentationRequestedRoute', 'presentationResolvedRoute', 'presentationRequestedVariant', 'presentationLocale']) Reflect.deleteProperty(document.documentElement.dataset, key);
    return () => { delete document.documentElement.dataset.experienceState; delete document.documentElement.dataset.captureReady; };
  }, [state, title]);
  return <main data-testid="landing-experience-surface" data-experience-surface="public-landing" data-experience-state={state} data-http-status={state === 'not-found' ? '404' : undefined} aria-busy={state === 'loading'} className="min-h-full bg-slate-950 px-6 py-24 text-center text-slate-100">
    <h1 className="text-2xl font-semibold">{title}</h1>
    {state !== 'loading' && <p role="status" className="mt-4">{state === 'not-found' ? copy.notFoundDetail : copy.unavailableDetail}</p>}
    {state === 'unavailable' && retry && <button className="mt-6 rounded border border-white/30 px-4 py-2" onClick={retry}>{copy.retry}</button>}
  </main>;
}
