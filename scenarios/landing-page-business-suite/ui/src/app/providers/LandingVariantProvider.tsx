import { createContext, useContext, useEffect, useState, ReactNode, useCallback } from 'react';
import { getLandingConfig, isApiError, type LandingConfigResponse, type Variant as LandingVariant } from '../../shared/api';
import { getFallbackLandingConfig } from '../../shared/lib/fallbackLandingConfig';

export type VariantResolution = 'unknown' | 'url_param' | 'local_storage' | 'api_select' | 'fallback';

/**
 * Structured logging for variant loading failures.
 * Provides context for debugging while avoiding sensitive data exposure.
 */
function logVariantError(context: string, error: unknown, metadata?: Record<string, unknown>) {
  const errorInfo = {
    context,
    timestamp: new Date().toISOString(),
    errorType: isApiError(error) ? error.type : 'unknown',
    errorMessage: error instanceof Error ? error.message : String(error),
    retryable: isApiError(error) ? error.retryable : false,
    ...metadata,
  };

  // Log structured error for debugging (visible in browser devtools)
  console.error('[LandingVariantProvider]', JSON.stringify(errorInfo, null, 2));

  // Expose a global hook for external monitoring tools
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('landing:variant:error', { detail: errorInfo }));
  }
}
// Note: local_storage resolution is retained for backward compatibility in consumers,
// but we no longer persist/read variant assignments locally. Variants are resolved server-side
// unless explicitly pinned via URL param.

interface LandingVariantContextType {
  variant: LandingVariant | null;
  config: LandingConfigResponse | null;
  loading: boolean;
  error: string | null;
  resolution: VariantResolution;
  statusNote: string | null;
  lastUpdated: number | null;
  refresh: () => Promise<void>;
}

const LandingVariantContext = createContext<LandingVariantContextType | undefined>(undefined);

function getVariantSlugFromUrl(): string | null {
  const params = new URLSearchParams(window.location.search);
  return params.get('variant') || params.get('variant_slug');
}

export function LandingVariantProvider({ children }: { children: ReactNode }) {
  const [variant, setVariant] = useState<LandingVariant | null>(null);
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
      const slugToUse = urlSlug || undefined;

      const landingConfig = await getLandingConfig(slugToUse);
      if (!landingConfig?.variant?.slug) {
        throw new Error('Landing config missing variant');
      }

      const nextResolution: VariantResolution = urlSlug ? 'url_param' : 'api_select';

      const note =
        nextResolution === 'url_param'
          ? 'Variant pinned via URL parameter'
          : 'Variant selected via weighted API';

      applyConfig(landingConfig, landingConfig.fallback ? 'fallback' : nextResolution, note);
    } catch (err) {
      // Structured logging for better observability
      const urlSlug = getVariantSlugFromUrl();
      logVariantError('loadVariant', err, {
        requestedSlug: urlSlug,
        userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : 'unknown',
      });

      const message = isApiError(err) ? err.userMessage : (err instanceof Error ? err.message : 'Unknown error');

      try {
        const fallbackConfig = getFallbackLandingConfig();
        applyConfig(fallbackConfig, 'fallback', `API unavailable: ${message}`);

        // Log successful fallback activation
        console.info('[LandingVariantProvider] Fallback activated', {
          reason: message,
          fallbackVariant: fallbackConfig.variant?.slug,
        });
      } catch (fallbackErr) {
        logVariantError('loadFallback', fallbackErr, {
          originalError: message,
        });
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
