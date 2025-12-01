import { apiCall } from './common';
import type { LandingConfigResponse, PricingOverview } from './types';

export function getLandingConfig(variantSlug?: string) {
  const params = new URLSearchParams();
  if (variantSlug) {
    params.set('variant', variantSlug);
  }
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<LandingConfigResponse>(`/landing-config${query}`);
}

export function getPlans() {
  return apiCall<PricingOverview>('/plans');
}
