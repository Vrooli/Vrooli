import { getPublicBranding } from '../../../shared/api/branding';
import type { PublicBranding } from '../../../shared/api/types';

let brandingRequest: Promise<PublicBranding | null> | undefined;

/** One public branding read per page load, shared by every site surface. */
export function loadSiteBranding(): Promise<PublicBranding | null> {
  brandingRequest ??= getPublicBranding().catch((error: unknown) => {
    console.error('[landing-page-business-suite] public branding unavailable', error);
    brandingRequest = undefined;
    return null;
  });
  return brandingRequest;
}

/** Test seam: forget the cached branding read. */
export function resetSiteBrandingCache() {
  brandingRequest = undefined;
}
