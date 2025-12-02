import { apiCall } from './common';
import type { SiteBranding, SiteBrandingUpdate, PublicBranding } from './types';

// Admin endpoints (require authentication)

export function getBranding() {
  return apiCall<SiteBranding>('/admin/branding');
}

export function updateBranding(data: SiteBrandingUpdate) {
  return apiCall<SiteBranding>('/admin/branding', {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export function clearBrandingField(field: string) {
  return apiCall<SiteBranding>('/admin/branding/clear-field', {
    method: 'POST',
    body: JSON.stringify({ field }),
  });
}

// Public endpoints (no auth required)

export function getPublicBranding() {
  return apiCall<PublicBranding>('/public/branding');
}
