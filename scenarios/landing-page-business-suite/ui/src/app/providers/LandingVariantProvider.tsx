import { createContext, useContext, useEffect, useState, ReactNode, useCallback } from 'react';
import { getLandingConfig, type LandingConfigResponse } from '../../shared/api';
import { getFallbackLandingConfig } from '../../shared/lib/fallbackLandingConfig';

export interface Variant {
  id?: number;
  slug: string;
  name: string;
  description?: string;
  weight?: number;
  status?: string;
  created_at?: string;
  updated_at?: string;
}

export type VariantResolution = 'unknown' | 'url_param' | 'local_storage' | 'api_select' | 'fallback';

interface LandingVariantContextType {
  variant: Variant | null;
  config: LandingConfigResponse | null;
  loading: boolean;
  error: string | null;
  resolution: VariantResolution;
  statusNote: string | null;
  lastUpdated: number | null;
  refresh: () => Promise<void>;
}

const LandingVariantContext = createContext<LandingVariantContextType | undefined>(undefined);

const VARIANT_STORAGE_KEY = 'landing_manager_variant_slug';

function getVariantSlugFromUrl(): string | null {
  const params = new URLSearchParams(window.location.search);
  return params.get('variant') || params.get('variant_slug');
}

function getStoredVariantSlug(): string | null {
  try {
    return localStorage.getItem(VARIANT_STORAGE_KEY);
  } catch (err) {
    console.warn('Failed to read stored variant slug:', err);
    return null;
  }
}

function storeVariantSlug(slug: string): void {
  try {
    localStorage.setItem(VARIANT_STORAGE_KEY, slug);
  } catch (err) {
    console.error('Failed to store variant slug:', err);
  }
}

export function LandingVariantProvider({ children }: { children: ReactNode }) {
  const [variant, setVariant] = useState<Variant | null>(null);
  const [config, setConfig] = useState<LandingConfigResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [resolution, setResolution] = useState<VariantResolution>('unknown');
  const [statusNote, setStatusNote] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<number | null>(null);

  const applyConfig = useCallback(
    (nextConfig: LandingConfigResponse | null, nextResolution: VariantResolution, note?: string | null) => {
      setConfig(nextConfig);

      if (nextConfig?.variant) {
        const status = nextConfig.fallback ? 'fallback' : 'active';
        setVariant({
          id: nextConfig.variant.id,
          slug: nextConfig.variant.slug,
          name: nextConfig.variant.name,
          description: nextConfig.variant.description,
          status,
        });

        if (!nextConfig.fallback && nextConfig.variant.slug) {
          storeVariantSlug(nextConfig.variant.slug);
        }
      } else {
        setVariant(null);
      }

      setResolution(nextResolution);
      setStatusNote(note ?? null);
      setLastUpdated(Date.now());
    },
    []
  );

  const loadVariant = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const urlSlug = getVariantSlugFromUrl();
      const storedSlug = getStoredVariantSlug();
      const slugToUse = urlSlug || storedSlug || undefined;

      const landingConfig = await getLandingConfig(slugToUse);
      if (!landingConfig?.variant?.slug) {
        throw new Error('Landing config missing variant');
      }

      const nextResolution: VariantResolution = urlSlug
        ? 'url_param'
        : storedSlug
          ? 'local_storage'
          : 'api_select';

      const note =
        nextResolution === 'url_param'
          ? 'Variant pinned via URL parameter'
          : nextResolution === 'local_storage'
            ? 'Reusing visitor assignment from localStorage'
            : 'Variant selected via weighted API';

      applyConfig(landingConfig, landingConfig.fallback ? 'fallback' : nextResolution, note);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      console.error('Failed to load variant:', message);

      try {
        const fallbackConfig = getFallbackLandingConfig();
        applyConfig(fallbackConfig, 'fallback', `API unavailable: ${message}`);
      } catch (fallbackErr) {
        console.error('Failed to load fallback landing config:', fallbackErr);
        setError(message);
        setResolution('unknown');
        setStatusNote(message);
      }
    } finally {
      setLoading(false);
    }
  }, [applyConfig]);

  useEffect(() => {
    loadVariant();
  }, [loadVariant]);

  const refresh = useCallback(async () => {
    await loadVariant();
  }, [loadVariant]);

  return (
    <LandingVariantContext.Provider
      value={{ variant, config, loading, error, resolution, statusNote, lastUpdated, refresh }}
    >
      {children}
    </LandingVariantContext.Provider>
  );
}

export function useLandingVariant() {
  const context = useContext(LandingVariantContext);
  if (context === undefined) {
    throw new Error('useLandingVariant must be used within a LandingVariantProvider');
  }
  return context;
}
