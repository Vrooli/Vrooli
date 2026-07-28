import { createClient } from '@connectrpc/connect';
import { BrandingService, type BrandingResponse, type PublicBrandingResponse } from '@vrooli/proto-types/landing-page-business-suite/branding_pb';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { CONNECT_API_BASE } from './common';
import type { SiteBranding, SiteBrandingUpdate, PublicBranding } from './types';

const brandingClient = createClient(BrandingService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

function snakeCase(value: string): string {
  return value.replace(/[A-Z]/gu, (letter) => `_${letter.toLowerCase()}`);
}

function normalizeBranding<T>(branding: { toJson(): unknown } | undefined): T {
  if (!branding) throw new Error('Missing branding response');
  const raw = branding.toJson();
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new Error('Invalid branding response');
  return Object.fromEntries(Object.entries(raw).map(([key, value]) => [snakeCase(key), key === 'id' ? Number(value) : value])) as T;
}

// Admin endpoints (require authentication)

export function getBranding() {
  return brandingClient.getBranding({}).then((response: BrandingResponse) => normalizeBranding<SiteBranding>(response.branding));
}

export function updateBranding(data: SiteBrandingUpdate) {
  const request = Object.fromEntries(Object.entries(data).map(([key, value]) => [key.replace(/_([a-z])/gu, (_, letter: string) => letter.toUpperCase()), value]));
  return brandingClient.updateBranding(request).then((response: BrandingResponse) => normalizeBranding<SiteBranding>(response.branding));
}

export function clearBrandingField(field: string) {
  return brandingClient.clearBrandingField({ field }).then((response: BrandingResponse) => normalizeBranding<SiteBranding>(response.branding));
}

// Public endpoints (no auth required)

export function getPublicBranding() {
  return brandingClient.getPublicBranding({}).then((response: PublicBrandingResponse) => normalizeBranding<PublicBranding>(response.branding));
}
