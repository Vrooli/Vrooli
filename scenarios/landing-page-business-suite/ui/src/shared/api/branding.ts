import { createClient } from '@connectrpc/connect';
import { z } from 'zod';
import { BrandingService, type BrandingResponse, type PublicBrandingResponse } from '@vrooli/proto-types/landing-page-business-suite/v1/branding_pb';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { CONNECT_API_BASE } from './common';
import type { SiteBranding, SiteBrandingUpdate, PublicBranding } from './types';
import { parseOrThrow } from './safeParse';

const brandingClient = createClient(BrandingService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

const NullableStringSchema = z.string().nullable().optional();
const SiteBrandingSchema: z.ZodType<SiteBranding> = z.object({
  id: z.number(),
  site_name: z.string(),
  tagline: NullableStringSchema,
  logo_url: NullableStringSchema,
  logo_icon_url: NullableStringSchema,
  favicon_url: NullableStringSchema,
  apple_touch_icon_url: NullableStringSchema,
  default_title: NullableStringSchema,
  default_description: NullableStringSchema,
  default_og_image_url: NullableStringSchema,
  theme_primary_color: NullableStringSchema,
  theme_background_color: NullableStringSchema,
  canonical_base_url: NullableStringSchema,
  google_site_verification: NullableStringSchema,
  robots_txt: NullableStringSchema,
  support_chat_url: NullableStringSchema,
  support_email: NullableStringSchema,
  smtp_host: NullableStringSchema,
  smtp_port: z.number().nullable().optional(),
  smtp_username: NullableStringSchema,
  smtp_password: NullableStringSchema,
  smtp_from: NullableStringSchema,
  coming_soon_enabled: z.boolean().nullable().optional(),
  coming_soon_message: NullableStringSchema,
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
});
const PublicBrandingSchema: z.ZodType<PublicBranding> = z.object({
  site_name: z.string(),
  tagline: NullableStringSchema,
  logo_url: NullableStringSchema,
  logo_icon_url: NullableStringSchema,
  favicon_url: NullableStringSchema,
  theme_primary_color: NullableStringSchema,
  theme_background_color: NullableStringSchema,
  support_chat_url: NullableStringSchema,
  coming_soon_enabled: z.boolean().nullable().optional(),
  coming_soon_message: NullableStringSchema,
});

function snakeCase(value: string): string {
  return value.replace(/[A-Z]/gu, (letter) => `_${letter.toLowerCase()}`);
}

function normalizeBranding(branding: { toJson(): unknown } | undefined): Record<string, unknown> {
  if (!branding) throw new Error('Missing branding response');
  const raw = branding.toJson();
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new Error('Invalid branding response');
  return Object.fromEntries(Object.entries(raw).map(([key, value]) => [snakeCase(key), key === 'id' ? Number(value) : value]));
}

// Admin endpoints (require authentication)

export function getBranding() {
  return brandingClient.getBranding({}).then((response: BrandingResponse) => parseOrThrow(SiteBrandingSchema, normalizeBranding(response.branding), 'SiteBranding'));
}

export function updateBranding(data: SiteBrandingUpdate) {
  const request = Object.fromEntries(Object.entries(data).map(([key, value]) => [key.replace(/_([a-z])/gu, (_, letter: string) => letter.toUpperCase()), value]));
  return brandingClient.updateBranding(request).then((response: BrandingResponse) => parseOrThrow(SiteBrandingSchema, normalizeBranding(response.branding), 'SiteBranding'));
}

export function clearBrandingField(field: string) {
  return brandingClient.clearBrandingField({ field }).then((response: BrandingResponse) => parseOrThrow(SiteBrandingSchema, normalizeBranding(response.branding), 'SiteBranding'));
}

// Public endpoints (no auth required)

export function getPublicBranding() {
  return brandingClient.getPublicBranding({}).then((response: PublicBrandingResponse) => parseOrThrow(PublicBrandingSchema, normalizeBranding(response.branding), 'PublicBranding'));
}
