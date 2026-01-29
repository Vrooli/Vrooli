import { createContext } from 'react';
import type { LandingConfigResponse, Variant as LandingVariant } from '../../shared/api';

export type VariantResolution = 'unknown' | 'url_param' | 'local_storage' | 'api_select' | 'fallback';

export interface LandingVariantContextType {
  variant: LandingVariant | null;
  config: LandingConfigResponse | null;
  loading: boolean;
  error: string | null;
  resolution: VariantResolution;
  statusNote: string | null;
  lastUpdated: number | null;
  refresh: () => Promise<void>;
}

export const LandingVariantContext = createContext<LandingVariantContextType | undefined>(undefined);
