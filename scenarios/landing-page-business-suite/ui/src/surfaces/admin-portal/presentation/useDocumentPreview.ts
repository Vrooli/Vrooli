import { useEffect, useMemo, useState } from 'react';
import { Code, ConnectError } from '@connectrpc/connect';
import type { ProductPresentationDocument, ResolvedProductPresentation } from '../../../shared/api/productPresentation';

export type DocumentPreviewRequest = (input: { variantSlug: string; document: ProductPresentationDocument; route: string; locale: string }, signal: AbortSignal) => Promise<ResolvedProductPresentation | undefined>;
interface PreviewInput { variantSlug: string; document?: ProductPresentationDocument; route: string; locale: string; enabled: boolean }
type Result = { identity: object; status: 'loading' | 'ready' | 'error'; presentation?: ResolvedProductPresentation; error?: string; unauthorized?: boolean };

/** Read-only preview state is separate from save/publish CAS state and never locks typing. */
export function useDocumentPreview(input: PreviewInput, request: DocumentPreviewRequest) {
  const { variantSlug, document, route, locale, enabled } = input;
  const identity = useMemo(() => ({ variantSlug, document, route, locale, enabled }), [variantSlug, document, route, locale, enabled]);
  const validRoute = route === '/' || /^\/apps\/[a-z0-9][a-z0-9-]*$/.test(route);
  const eligible = enabled && !!document && validRoute;
  const [result, setResult] = useState<Result>();
  useEffect(() => {
    if (!eligible) return;
    const controller = new AbortController();
    const timer = setTimeout(() => {
      setResult({ identity, status: 'loading' });
      void request({ variantSlug, document, route, locale }, controller.signal).then(presentation => {
        if (controller.signal.aborted) return;
        const d = presentation?.diagnostics;
        if (!presentation?.page || !d?.preview || !d.noindex || !d.noStore || d.fallback || !/^[a-f0-9]{64}$/.test(d.resolvedRevision) || d.requestedRevision !== d.resolvedRevision || !d.blockDigest || d.requestedRoute !== route || d.resolvedRoute !== route || d.requestedVariant !== variantSlug || d.resolvedVariant !== variantSlug || (locale && d.locale !== locale) || presentation.page.locale !== d.locale) throw new Error('Preview privacy or request identity mismatch');
        setResult({ identity, status: 'ready', presentation });
      }).catch((cause: unknown) => {
        if (controller.signal.aborted) return;
        const unauthorized = cause instanceof ConnectError && (cause.code === Code.Unauthenticated || cause.code === Code.PermissionDenied);
        const validation = cause instanceof ConnectError && (cause.code === Code.InvalidArgument || cause.code === Code.FailedPrecondition);
        setResult({ identity, status: 'error', unauthorized, error: unauthorized ? 'Administrator access is required. Sign in again to preview.' : validation ? cause.rawMessage : 'The private preview could not be resolved. Your local edits have not been saved.' });
      });
    }, 300);
    return () => { clearTimeout(timer); controller.abort(); };
  }, [identity, eligible, document, request, variantSlug, route, locale]);
  // Hide an obsolete response during render, before debounce/effect cleanup runs.
  if (!eligible) return { status: 'inactive' as const, presentation: undefined, error: enabled && !validRoute ? 'Use / or an exact /apps/:slug route.' : undefined, unauthorized: false };
  if (result?.identity !== identity) return { status: 'debouncing' as const, presentation: undefined, error: undefined, unauthorized: false };
  return result;
}
