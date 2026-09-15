import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react';
import { useLocation } from 'react-router-dom';
import { Code, ConnectError } from '@connectrpc/connect';
import { getLandingConfig } from '../../shared/api/landing';
import type { LandingConfigResponse } from '../../shared/api/types';
import { LandingVariantContext, type LandingVariantContextType } from './LandingVariantContext';
import { waitForLandingWorkflowLoadingState } from './landingWorkflowLoading';
import { presentationSystemUi } from '../../surfaces/public-landing/presentation/systemUi';
import { readPresentationBootstrap } from './landingPresentationBootstrap';
import { getVisitorId } from '../../shared/lib/attribution';

export type { VariantResolution } from './LandingVariantContext';
interface State {
  key: string; config: LandingConfigResponse | null; loading: boolean;
  error: string | null; notFound: boolean; updated: number | null;
}
/** Globally composed for legacy consumers, but public assignment is route-scoped. */
export function LandingVariantProvider({ children }: { children: ReactNode }) {
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  const variant = params.get('variant') || params.get('variant_slug') || '';
  const locale = params.get('locale') || '';
  const download = /^\/apps\/[a-z0-9][a-z0-9-]*\/download$/.test(location.pathname);
  const detail = download || /^\/apps\/[a-z0-9][a-z0-9-]*$/.test(location.pathname);
  const active = location.pathname === '/' || detail || ['/checkout', '/feedback'].includes(location.pathname);
  // Download is a noindex workflow over the canonical detail document, not a new page mode.
  const route = download ? location.pathname.slice(0, -9) : detail ? location.pathname : '/';
  const key = JSON.stringify([location.pathname, variant, locale]);
  const [initial] = useState(() => active ? readPresentationBootstrap({ route, locale, variant }) : undefined);
  const visitor = useRef<string>();
  if (active && !visitor.current) visitor.current = getVisitorId(initial?.visitorId);
  const initialKey = useRef(key);
  const started = useRef(false);
  const [state, setState] = useState<State>(() => ({ key: initial ? key : '', config: initial?.config ?? null, loading: false, error: null, notFound: false, updated: initial ? Date.now() : null }));
  const pending = useRef<AbortController>();
  const currentKey = useRef(key);
  currentKey.current = key;
  const load = useCallback(async (force = false) => {
    pending.current?.abort();
    if (!active) return;
    // Bootstrap already supplies this document. Explicit refresh and navigation
    // are read-only resolutions using the same anonymous visitor identity.
    if (!force && initial && !started.current && initialKey.current === key) return;
    started.current = true;
    const controller = new AbortController(); pending.current = controller;
    setState({ key, config: null, loading: true, error: null, notFound: false, updated: null });
    const current = () => !controller.signal.aborted && currentKey.current === key;
    try {
      await waitForLandingWorkflowLoadingState();
      if (!current()) return;
      const config = await getLandingConfig(variant || undefined, visitor.current, { route, locale, signal: controller.signal });
      if (!current()) return;
      if (!config.presentation.diagnostics?.resolvedVariant || config.presentation.diagnostics.preview) throw new Error('Invalid public configuration');
      setState({ key, config, loading: false, error: null, notFound: false, updated: Date.now() });
    } catch (error) {
      if (!current()) return;
      const code = ConnectError.from(error).code;
      setState({ key, config: null, loading: false, error: presentationSystemUi.unavailable,
        notFound: code === Code.NotFound || (detail && code === Code.PermissionDenied), updated: Date.now() });
    }
  }, [active, detail, key, locale, route, variant, initial]);
  useEffect(() => { void load(); return () => { pending.current?.abort(); }; }, [load]);
  // Key the visible snapshot synchronously: no stale page/metrics frame before effects.
  const current = active && state.key === key ? state : undefined;
  const config = current?.config ?? null;
  const value: LandingVariantContextType = {
    config, variant: config?.presentation.diagnostics ? { slug: config.presentation.diagnostics.resolvedVariant } : null,
    loading: active && (!current || current.loading), error: current?.error ?? null,
    notFound: current?.notFound ?? false, request: { route, locale, variant },
    canonicalBaseUrl: initial?.canonicalBaseUrl,
    visitorId: active ? visitor.current : undefined, refreshable: true,
    resolution: config ? config.fallback ? 'fallback' : variant ? 'url_param' : 'api_select' : 'unknown',
    statusNote: null, lastUpdated: current?.updated ?? null, refresh: () => load(true),
  };
  return <LandingVariantContext.Provider value={value}>{children}</LandingVariantContext.Provider>;
}
