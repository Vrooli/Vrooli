import { useEffect, useMemo, useRef, useState, type MouseEvent } from 'react';
import { useHref, useLocation } from 'react-router-dom';
import { useLandingVariant } from '../../../app/providers/useLandingVariant';
import { useMetrics } from '../../../shared/hooks/useMetricsHook';
import { updateMetaTags, updateThemeColor } from '../../../shared/lib/seo';
import { PresentationPage, type PresentationPageProps } from '../presentation';
import { canonicalPresentationHref, resolvePublicPresentation } from '../presentation/publicIntegration';
import { PublicPresentationState } from '../presentation/PublicPresentationState';
import { resolvePricing } from '../presentation/commerce';
import { usePresentationExposure } from '../presentation/usePresentationExposure';
import type { PresentationDiagnostics } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';

export const PRESENTATION_READY_TIMEOUT_MS = 15000;

// These are the locally shipped families declared by presentation.css, not the
// application's global/admin fonts. Inspect rendered styles to select used faces.
const PRESENTATION_FONT_FAMILIES = new Set(['PresentationArchivo', 'PresentationSans', 'Presentation Mono']);
function presentationFontRequests(root: HTMLElement | null) {
  const requests = new Map<string, { family: string; text: string }>();
  for (const node of root?.querySelectorAll<HTMLElement>('.presentation-page, .presentation-page *') ?? []) {
    const style = getComputedStyle(node);
    const family = style.fontFamily.split(',')[0]?.trim().replace(/^['"]|['"]$/g, '');
    if (!family || !PRESENTATION_FONT_FAMILIES.has(family)) continue;
    const request = `${style.fontStyle || 'normal'} ${style.fontWeight || '400'} 16px "${family}"`;
    requests.set(request, { family, text: node.textContent || 'BESbswy' });
  }
  return requests;
}

export function PublicLanding() {
  const { config, loading, notFound, refresh, refreshable, request, canonicalBaseUrl } = useLandingVariant();
  const location = useLocation();
  const base = useHref('/');
  const resolved = useMemo(() => {
    if (!config?.presentation || !request) return undefined;
    try { return resolvePublicPresentation(config.presentation, request, base, config.downloads); }
    catch (error) {
      // Keep the public response generic while exposing contract failures to
      // production browser evidence and operators.
      console.error('[landing-page-business-suite] public presentation binding failed', error);
      return undefined;
    }
  }, [config, request, base]);
  if (location.pathname !== '/' && !/^\/apps\/[a-z0-9][a-z0-9-]*$/.test(location.pathname)) return <PublicPresentationState state="not-found" />;
  if (loading) return <PublicPresentationState state="loading" />;
  if (notFound) return <PublicPresentationState state="not-found" />;
  const diagnostics = config?.presentation.diagnostics;
  if (!resolved || !diagnostics) return <PublicPresentationState state="unavailable" retry={refreshable === false ? undefined : () => { void refresh(); }} />;
  return <ReadyPresentation key={`${location.pathname}:${diagnostics.resolvedVariant}:${diagnostics.resolvedRevision}:${diagnostics.locale}`} {...resolved} resolvedPricing={resolvePricing(resolved.presentation, config.pricing)} diagnostics={diagnostics} canonicalBaseUrl={canonicalBaseUrl} />;
}

function ReadyPresentation({ diagnostics: d, canonicalBaseUrl, ...props }: PresentationPageProps & { diagnostics: PresentationDiagnostics; canonicalBaseUrl?: string }) {
  const root = useRef<HTMLDivElement>(null);
  const [pixels, setPixels] = useState<'loading' | 'ready' | 'unavailable'>('loading');
  const failed = useRef(false);
  const { trackCTAClick } = useMetrics();
  const { visitorId } = useLandingVariant();
  const { pathname } = useLocation();
  usePresentationExposure(d, visitorId, pixels === 'ready', pathname);
  const page = props.presentation.page;
  useEffect(() => {
    const element = document.documentElement;
    const authority = canonicalBaseUrl ?? document.querySelector<HTMLMetaElement>('meta[name="presentation-canonical-base"]')?.content;
    const canonical = canonicalPresentationHref(authority, d.resolvedRoute);
    if (!canonical) document.querySelector('link[rel="canonical"]')?.remove();
    updateMetaTags({ title: page.title, description: page.description, canonical, noindex: d.noindex || Boolean(d.requestedVariant), twitterCard: 'summary' });
    let ogUrl = document.querySelector<HTMLMetaElement>('meta[property="og:url"]');
    if (canonical) {
      if (!ogUrl) { ogUrl = document.createElement('meta'); ogUrl.setAttribute('property', 'og:url'); document.head.append(ogUrl); }
      ogUrl.content = canonical;
    } else ogUrl?.remove();
    updateThemeColor(page.theme.background);
    element.lang = page.locale;
    Object.assign(element.dataset, {
      presentationRevision: d.resolvedRevision, presentationMode: props.presentation.mode,
      presentationDigest: d.blockDigest, variantSlug: d.resolvedVariant,
      presentationRequestedRoute: d.requestedRoute, presentationResolvedRoute: d.resolvedRoute,
      presentationRequestedVariant: d.requestedVariant, presentationLocale: d.locale,
      experienceState: 'loading', captureReady: 'false',
    });
    let live = true;
    let timeout: ReturnType<typeof setTimeout> | undefined;
    const cleanups: (() => void)[] = [];
    const fonts = document.fonts;
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- Test DOMs may omit the Font Loading API.
    const fontRequests = fonts ? presentationFontRequests(root.current) : new Map<string, { family: string; text: string }>();
    const usedFont = (face: FontFace) => [...fontRequests.values()].some(({ family }) => family === face.family.replace(/^['"]|['"]$/g, ''));
    const requiredFontFailed = () => {
      let failed = false;
      fonts.forEach(face => { if (usedFont(face) && face.status === 'error') failed = true; });
      return failed;
    };
    const unavailable = () => {
      if (!live) return;
      failed.current = true;
      element.dataset.experienceState = 'unavailable'; element.dataset.captureReady = 'false'; setPixels('unavailable');
    };
    // Keep this listener after initial readiness: later font loads can fail too.
    const fontError = () => { if (requiredFontFailed()) unavailable(); };
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- Test DOMs may omit the Font Loading API.
    fonts?.addEventListener('loadingerror', fontError);
    const images = [...(root.current?.querySelectorAll('img') ?? [])].filter(img => img.loading !== 'lazy');
    // Readiness describes actual pixels/fonts, not merely an accepted response.
    const ready = async () => {
      try {
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- Test DOMs may omit the Font Loading API.
        const fontReady = fonts ? Promise.all([...fontRequests].map(async ([request, { text }]) => {
          const faces = await fonts.load(request, text);
          if (!faces.length || faces.some(face => face.status !== 'loaded')) throw new Error('Presentation font failed');
        })) : undefined;
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- Test DOMs may omit the Font Loading API.
        if (fonts && !fontRequests.size) throw new Error('Presentation typography unavailable');
        const imageReady = Promise.all(images.map(img => {
          if (img.complete && img.naturalWidth > 0) return Promise.resolve();
          if (typeof img.decode === 'function') return img.decode();
          return new Promise<void>((resolve, reject) => {
            const loaded = () => { resolve(); }; const error = () => { reject(new Error('Presentation image failed')); };
            img.addEventListener('load', loaded); img.addEventListener('error', error);
            cleanups.push(() => { img.removeEventListener('load', loaded); img.removeEventListener('error', error); });
          });
        }));
        await Promise.race([
          Promise.all([fontReady, imageReady]),
          new Promise<never>((_, reject) => { timeout = setTimeout(() => { reject(new Error('Presentation readiness timed out')); }, PRESENTATION_READY_TIMEOUT_MS); }),
        ]);
        // FontFaceSet.ready fulfills even when individual faces failed to load.
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- Test DOMs may omit the Font Loading API.
        if (fonts && requiredFontFailed()) throw new Error('Presentation font failed');
        if (live && !failed.current) { element.dataset.experienceState = 'ready'; element.dataset.captureReady = 'true'; setPixels('ready'); }
      } catch { unavailable(); }
      finally { clearTimeout(timeout); cleanups.forEach(cleanup => { cleanup(); }); }
    };
    void ready();
    return () => {
      live = false;
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- Test DOMs may omit the Font Loading API.
      fonts?.removeEventListener('loadingerror', fontError);
      clearTimeout(timeout); cleanups.forEach(cleanup => { cleanup(); });
      updateMetaTags({ noindex: true });
      document.querySelector('link[rel="canonical"]')?.remove();
      document.querySelector('meta[property="og:url"]')?.remove();
      for (const key of ['presentationRevision', 'presentationMode', 'presentationDigest', 'variantSlug', 'presentationRequestedRoute', 'presentationResolvedRoute', 'presentationRequestedVariant', 'presentationLocale', 'experienceState', 'captureReady']) Reflect.deleteProperty(element.dataset, key);
    };
  }, [d, page, canonicalBaseUrl, props.presentation.mode]);
  const click = (event: MouseEvent<HTMLDivElement>) => {
    if (!(event.target instanceof Element)) return;
    const target = event.target.closest('a,button');
    if (target?.closest('.unavailable-action')) return;
    if (target) trackCTAClick(target.closest('[data-block]')?.id || 'presentation-navigation', { route: d.resolvedRoute, revision: d.resolvedRevision });
  };
  if (pixels === 'unavailable') return <PublicPresentationState state="unavailable" />;
  return <div ref={root} onClick={click} onErrorCapture={event => { if (event.target instanceof HTMLImageElement) { failed.current = true; document.documentElement.dataset.experienceState = 'unavailable'; document.documentElement.dataset.captureReady = 'false'; setPixels('unavailable'); } }} data-testid="landing-experience-surface" data-experience-surface="public-landing" data-experience-state={pixels}
    data-presentation-revision={d.resolvedRevision} data-presentation-mode={props.presentation.mode}
    data-presentation-digest={d.blockDigest} data-variant-slug={d.resolvedVariant}
    data-presentation-route={d.resolvedRoute} data-presentation-locale={d.locale}
    data-capture-landmark={props.presentation.mode === 'app_detail' ? 'app-detail' : undefined}>
    <PresentationPage {...props} />
  </div>;
}
