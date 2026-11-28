import { useEffect, useRef } from 'react';
import { PresentationAssignmentSource } from '@vrooli/proto-types/landing-page-business-suite/v1/config_pb';
import type { PresentationDiagnostics } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import { recordPresentationExposure } from '../../../shared/api/landing';
import { isValidVisitorId } from '../../../shared/lib/visitorIdentity';

/** Public route effect only: the renderer and authorized previews stay pure. */
export function usePresentationExposure(d: PresentationDiagnostics, visitorId: string | undefined, ready: boolean, pathname: string) {
  const attempted = useRef(new Set<string>());
  useEffect(() => {
    if (!ready || !isValidVisitorId(visitorId) || d.preview || d.requestedVariant || d.assignmentSource !== 'weighted_visitor') return;
    if (pathname !== d.resolvedRoute || (pathname !== '/' && !/^\/apps\/[a-z0-9][a-z0-9-]*$/.test(pathname))) return;
    if (!d.resolvedVariant || !d.resolvedRevision || !d.locale || !d.blockDigest || !d.weightFingerprint) return;
    const proof = {
      visitorId, variantSlug: d.resolvedVariant, revision: d.resolvedRevision, route: d.resolvedRoute,
      locale: d.locale, blockDigest: d.blockDigest, weightFingerprint: d.weightFingerprint,
      source: PresentationAssignmentSource.WEIGHTED_VISITOR,
    };
    const key = JSON.stringify(proof);
    const record = () => {
      if (document.visibilityState !== 'visible' || attempted.current.has(key)) return;
      attempted.current.add(key);
      // Owner revalidates and deduplicates. Metrics failure never hides the page.
      void recordPresentationExposure(proof).catch(() => {});
    };
    record();
    document.addEventListener('visibilitychange', record);
    return () => { document.removeEventListener('visibilitychange', record); };
  }, [d, visitorId, ready, pathname]);
}
