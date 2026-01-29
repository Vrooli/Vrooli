import { useContext } from 'react';
import { LandingVariantContext } from './LandingVariantContext';

// Re-export the type for convenience
export type { VariantResolution } from './LandingVariantContext';

export function useLandingVariant() {
  const context = useContext(LandingVariantContext);
  if (context === undefined) {
    throw new Error('useLandingVariant must be used within a LandingVariantProvider');
  }
  return context;
}
