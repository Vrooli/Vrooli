import { createContext } from 'react';
import type { LandingConfigResponse, Variant as LandingVariant } from '../../shared/api';

export type VariantResolution = 'unknown' | 'url_param' | 'local_storage' | 'api_select' | 'fallback';

export interface LandingVariantContextType {
  request?: { route: string; locale: string; variant: string };
  notFound?: boolean;
  canonicalBaseUrl?: string;
  refreshable?: boolean;
  visitorId?: string;
  /** Public projection exposes a resolved slug, not private variant metadata. */
  variant: (Pick<LandingVariant, 'slug'> & Partial<LandingVariant>) | null;
  config: LandingConfigResponse | null;
  loading: boolean;
  error: string | null;
  resolution: VariantResolution;
  statusNote: string | null;
  lastUpdated: number | null;
  refresh: () => Promise<void>;
}

export const LandingVariantContext = createContext<LandingVariantContextType | undefined>(undefined);
