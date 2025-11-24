import { createContext, useContext, useEffect, useState, ReactNode } from 'react';

export interface Variant {
  id: string;
  slug: string;
  name: string;
  description: string;
  weight: number;
  status: string;
  created_at: string;
  updated_at: string;
}

interface VariantContextType {
  variant: Variant | null;
  loading: boolean;
  error: string | null;
}

const VariantContext = createContext<VariantContextType | undefined>(undefined);

const VARIANT_STORAGE_KEY = 'landing_manager_variant';

/**
 * Fetches a variant by slug from the API
 * [REQ:AB-URL]
 */
async function fetchVariantBySlug(slug: string): Promise<Variant> {
  const res = await fetch(`/api/v1/variants/${slug}`, {
    headers: { 'Content-Type': 'application/json' },
    cache: 'no-store',
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch variant ${slug}: ${res.status}`);
  }

  return res.json();
}

/**
 * Selects a variant using weight-based random selection
 * [REQ:AB-API]
 */
async function selectVariant(): Promise<Variant> {
  const res = await fetch('/api/v1/variants/select', {
    headers: { 'Content-Type': 'application/json' },
    cache: 'no-store',
  });

  if (!res.ok) {
    throw new Error(`Failed to select variant: ${res.status}`);
  }

  return res.json();
}

/**
 * Gets the variant slug from URL query parameters
 * [REQ:AB-URL]
 */
function getVariantSlugFromUrl(): string | null {
  const params = new URLSearchParams(window.location.search);
  return params.get('variant') || params.get('variant_slug');
}

/**
 * Gets the stored variant from localStorage
 * [REQ:AB-STORAGE]
 */
function getStoredVariant(): Variant | null {
  try {
    const stored = localStorage.getItem(VARIANT_STORAGE_KEY);
    if (stored) {
      return JSON.parse(stored) as Variant;
    }
  } catch (err) {
    console.warn('Failed to parse stored variant:', err);
  }
  return null;
}

/**
 * Stores the variant in localStorage
 * [REQ:AB-STORAGE]
 */
function storeVariant(variant: Variant): void {
  try {
    localStorage.setItem(VARIANT_STORAGE_KEY, JSON.stringify(variant));
  } catch (err) {
    console.error('Failed to store variant:', err);
  }
}

export function VariantProvider({ children }: { children: ReactNode }) {
  const [variant, setVariant] = useState<Variant | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadVariant() {
      try {
        setLoading(true);
        setError(null);

        // Priority 1: Check URL for variant_slug [REQ:AB-URL]
        const urlSlug = getVariantSlugFromUrl();
        if (urlSlug) {
          const fetchedVariant = await fetchVariantBySlug(urlSlug);
          setVariant(fetchedVariant);
          storeVariant(fetchedVariant);
          return;
        }

        // Priority 2: Check localStorage [REQ:AB-STORAGE]
        const storedVariant = getStoredVariant();
        if (storedVariant) {
          setVariant(storedVariant);
          return;
        }

        // Priority 3: API weight-based selection [REQ:AB-API]
        const selectedVariant = await selectVariant();
        setVariant(selectedVariant);
        storeVariant(selectedVariant);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        console.error('Failed to load variant:', message);
        setError(message);
      } finally {
        setLoading(false);
      }
    }

    loadVariant();
  }, []);

  return (
    <VariantContext.Provider value={{ variant, loading, error }}>
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
