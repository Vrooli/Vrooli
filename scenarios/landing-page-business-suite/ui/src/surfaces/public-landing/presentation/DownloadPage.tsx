import { useEffect, useMemo, useRef, useState, type CSSProperties } from 'react';
import { useHref, useLocation } from 'react-router-dom';
import { Code, ConnectError } from '@connectrpc/connect';
import { useLandingVariant } from '../../../app/providers/useLandingVariant';
import { useUserAuth } from '../../../app/providers/useUserAuth';
import { requestDownload } from '../../../shared/api/downloads';
import type { DownloadAsset } from '../../../shared/api/types';
import { updateMetaTags } from '../../../shared/lib/seo';
import { actionsOf, canonicalPresentationHref, publicHref, resolvePublicPresentation, safeOwnerHref, scopedPresentationHref } from './publicIntegration';
import { PublicPresentationState } from './PublicPresentationState';
import { DownloadChooser, type DownloadState } from './DownloadChooser';
import { downloadSystemUi as ui } from './systemUi';
import { actionKey, type Presentation, type ResolvedActions } from './types';
import './presentation.css';

function detectedDownloadPlatform(): string {
  if (typeof navigator === 'undefined') return '';
  const value = `${navigator.userAgent} ${navigator.platform}`.toLowerCase();
  if (value.includes('win')) return 'windows';
  if (value.includes('mac')) return 'mac';
  if (value.includes('linux') || value.includes('x11')) return 'linux';
  return '';
}

function defaultDownloadSelection(options: DownloadAsset[]): string {
  const platform = detectedDownloadPlatform();
  const index = options.findIndex(option => option.platform === platform);
  return index >= 0 ? String(index) : options.length > 0 ? '0' : '';
}

export function DownloadPage() {
  const { config, request, loading, notFound, canonicalBaseUrl } = useLandingVariant();
  const base = useHref('/'); const location = useLocation();
  const resolved = useMemo(() => {
    if (!config?.presentation || !request || !/^\/apps\/[a-z0-9][a-z0-9-]*\/download$/.test(location.pathname) || request.route !== location.pathname.slice(0, -9)) return undefined;
    try { return resolvePublicPresentation(config.presentation, request, base, config.downloads); } catch { return undefined; }
  }, [config, request, base, location.pathname]);
  if (loading) return <PublicPresentationState state="loading" />;
  if (notFound) return <PublicPresentationState state="not-found" />;
  if (!resolved?.presentation.app_key || !request) return <PublicPresentationState state="unavailable" />;
  const appKey = resolved.presentation.app_key;
  const bundle = config?.presentation.diagnostics?.bundleKey;
  const apps = config?.downloads.filter(app => app.app_key === appKey && app.bundle_key === bundle) ?? [];
  const options = apps.length === 1 ? (apps[0]?.platforms ?? []).filter(asset => asset.app_key === appKey && asset.bundle_key === bundle && asset.platform && asset.release_version) : [];
  return <DownloadSession key={`${location.pathname}:${resolved.presentation.diagnostics.resolved_revision}`} presentation={resolved.presentation} resolvedActions={resolved.resolvedActions} options={options} appMetadata={apps[0]?.metadata} base={base} detailRoute={request.route} canonicalBaseUrl={canonicalBaseUrl} />;
}

function DownloadSession({ presentation, resolvedActions, options, appMetadata, base, detailRoute, canonicalBaseUrl }: { presentation: Presentation; resolvedActions: ResolvedActions; options: DownloadAsset[]; appMetadata?: Record<string, unknown>; base: string; detailRoute: string; canonicalBaseUrl?: string }) {
  const auth = useUserAuth();
  const { request } = useLandingVariant();
  const { refreshSession } = auth;
  const [selected, setSelected] = useState(() => defaultDownloadSelection(options));
  const [state, setState] = useState<{ identity: object | null; value: DownloadState }>({ identity: null, value: { status: 'idle' } });
  const pending = useRef(false); const operation = useRef(0); const identity = useRef(auth.user); identity.current = auth.user;
  const stateValue = state.identity === auth.user ? state.value : { status: 'idle' as const };
  useEffect(() => {
    if (selected === '' && options.length > 0) setSelected(defaultDownloadSelection(options));
  }, [options, selected]);
  const asset = selected === '' ? undefined : options[Number(selected)];
  const invalidId = asset && (asset.id === undefined || !Number.isSafeInteger(asset.id) || asset.id <= 0);
  const duplicateId = asset && options.filter(option => option.id === asset.id).length !== 1;
  const disabledReason = invalidId ? ui.invalidId : duplicateId ? ui.duplicateId : auth.isSessionLoading ? ui.checking : asset?.requires_entitlement && !auth.isAuthenticated ? ui.signInRequired : undefined;
  useEffect(() => { if (options.length > 0) void refreshSession(); }, [refreshSession, options.length]);
  useEffect(() => () => { operation.current++; }, []);
  useEffect(() => {
    const authority = canonicalBaseUrl ?? document.querySelector<HTMLMetaElement>('meta[name="presentation-canonical-base"]')?.content;
    const canonical = canonicalPresentationHref(authority, detailRoute);
    updateMetaTags({ title: presentation.page.title, description: presentation.page.description, canonical, noindex: true, twitterCard: 'summary' });
    let ogUrl = document.querySelector<HTMLMetaElement>('meta[property="og:url"]');
    if (canonical) {
      if (!ogUrl) { ogUrl = document.createElement('meta'); ogUrl.setAttribute('property', 'og:url'); document.head.append(ogUrl); }
      ogUrl.content = canonical;
    } else { document.querySelector('link[rel="canonical"]')?.remove(); ogUrl?.remove(); }
    return () => { document.querySelector('link[rel="canonical"]')?.remove(); document.querySelector('meta[property="og:url"]')?.remove(); };
  }, [presentation.page, canonicalBaseUrl, detailRoute]);
  const choose = (value: string) => { operation.current++; pending.current = false; setSelected(value); setState({ identity: auth.user, value: { status: 'idle' } }); };
  const prepare = async () => {
    if (!asset || asset.id === undefined || disabledReason || pending.current) return;
    const op = ++operation.current; const owner = auth.user; pending.current = true;
    setState({ identity: owner, value: { status: 'preparing' } });
    const current = () => operation.current === op && identity.current === owner;
    try {
      // Reuse session-derived authorization. Never use the catalog artifact URL.
      const authorized = await requestDownload(asset.app_key, asset.platform, undefined, { assetId: asset.id });
      if (!current()) return;
      if (authorized.bundle_key !== asset.bundle_key || authorized.app_key !== asset.app_key || authorized.platform !== asset.platform || authorized.release_version !== asset.release_version ||
        authorized.id !== asset.id || (asset.checksum && authorized.checksum !== asset.checksum)) throw new Error('changed');
      const href = safeOwnerHref(authorized.artifact_url, base);
      if (!href || /\/apps\/[^/]+(?:\/download)?(?:[?#]|$)/.test(href)) throw new Error('changed');
      setState({ identity: owner, value: { status: 'ready', href } });
    } catch (error) {
      if (!current()) return;
      const code = ConnectError.from(error).code;
      setState({ identity: owner, value: { status: 'error', message: code === Code.Unauthenticated ? ui.signInRequired : code === Code.PermissionDenied ? ui.denied : code === Code.FailedPrecondition || code === Code.NotFound || (error instanceof Error && error.message === 'changed') ? ui.changed : ui.failed } });
    } finally { if (operation.current === op) pending.current = false; }
  };
  const launchActions = [...new Map(actionsOf(presentation).filter(action => action.kind === 'open' && action.app_key === presentation.app_key).map(action => [actionKey(action), action])).values()];
  const theme = { '--ink': presentation.page.theme.primary, '--paper': presentation.page.theme.background, '--accent': presentation.page.theme.accent } as CSSProperties;
  return <main className={`presentation-page download-page theme-${presentation.page.theme.variant}`} lang={presentation.page.locale} style={theme} data-presentation-revision={presentation.diagnostics.resolved_revision} data-document-route={detailRoute}>
    <div className="wrap"><header className="download-header"><a href={scopedPresentationHref(detailRoute, base, request)}>{ui.back}</a>{options.length > 0 && <div>
      <span role="status">{auth.isSessionLoading ? ui.checking : auth.isAuthenticated ? ui.signedIn : ui.signInRequired}</span>
      {!auth.isAuthenticated && <a href={publicHref('/auth/login', base)}>{ui.signIn}</a>}
      <button type="button" disabled={auth.isSessionLoading} onClick={() => { void auth.refreshSession(); }}>{ui.recheck}</button>
    </div>}</header>
    {options.length > 0 && !auth.isAuthenticated && <p className="commerce-note">{ui.returnNote}</p>}
    <DownloadChooser title={presentation.page.title} description={presentation.page.description} options={options} selected={selected} onSelect={choose} onPrepare={() => { void prepare(); }} state={stateValue} disabledReason={disabledReason} unavailableReason={presentation.page.display.shell.unavailable_reason} appMetadata={appMetadata} launchActions={launchActions} resolvedActions={resolvedActions} />
    </div>
  </main>;
}
