import { apiCall } from './common';
import type { DownloadAsset, LandingConfigResponse, PricingOverview } from './types';

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

export function requestDownload(platform: string, user?: string) {
  const params = new URLSearchParams({ platform });
  if (user) {
    params.set('user', user);
  }
  return apiCall<DownloadAsset>(`/downloads?${params.toString()}`);
}
