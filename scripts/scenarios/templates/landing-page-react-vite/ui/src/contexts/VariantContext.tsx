import { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { getLandingConfig, type LandingConfigResponse } from '../lib/api';
import { getFallbackLandingConfig } from '../lib/fallbackLandingConfig';

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

interface VariantContextType {
  variant: Variant | null;
  config: LandingConfigResponse | null;
  loading: boolean;
  error: string | null;
}

const VariantContext = createContext<VariantContextType | undefined>(undefined);

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

export function VariantProvider({ children }: { children: ReactNode }) {
  const [variant, setVariant] = useState<Variant | null>(null);
  const [config, setConfig] = useState<LandingConfigResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    function applyConfig(nextConfig: LandingConfigResponse | null) {
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
    }

    async function loadVariant() {
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

        applyConfig(landingConfig);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        console.error('Failed to load variant:', message);

        try {
          const fallbackConfig = getFallbackLandingConfig();
          applyConfig(fallbackConfig);
        } catch (fallbackErr) {
          console.error('Failed to load fallback landing config:', fallbackErr);
          setError(message);
        }
      } finally {
        setLoading(false);
      }
    }

    loadVariant();
  }, []);

  return (
    <VariantContext.Provider value={{ variant, config, loading, error }}>
      {children}
    </VariantContext.Provider>
  );
}

export function useVariant() {
  const context = useContext(VariantContext);
  if (context === undefined) {
    throw new Error('useVariant must be used within a VariantProvider');
  }
  return context;
}
