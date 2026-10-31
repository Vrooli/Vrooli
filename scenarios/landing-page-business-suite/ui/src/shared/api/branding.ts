import { createClient } from '@connectrpc/connect';
import { fromJsonString, toJson, type JsonValue } from '@bufbuild/protobuf';
import { z } from 'zod';
import { BrandingService, BrandingResponseSchema, PublicBrandingResponseSchema, UpdateBrandingRequestSchema, type BrandingResponse, type PublicBrandingResponse } from '@vrooli/proto-types/landing-page-business-suite/v1/branding_pb';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { CONNECT_API_BASE } from './common';
import type { SiteBranding, SiteBrandingUpdate, PublicBranding } from './types';
import { parseOrThrow } from './safeParse';
import { withAdminReauthentication } from './adminReauthentication';

const brandingClient = createClient(BrandingService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

const NullableStringSchema = z.string().nullable().optional();
const SiteBrandingSchema: z.ZodType<SiteBranding> = z.object({
  id: z.number().int().safe(),
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
  legal_name: NullableStringSchema,
  contact_address: NullableStringSchema,
  privacy_policy_markdown: NullableStringSchema,
  terms_markdown: NullableStringSchema,
  privacy_effective_date: NullableStringSchema,
  terms_effective_date: NullableStringSchema,
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
  canonical_base_url: NullableStringSchema,
  support_chat_url: NullableStringSchema,
  coming_soon_enabled: z.boolean().nullable().optional(),
  coming_soon_message: NullableStringSchema,
  support_email: NullableStringSchema,
  legal_name: NullableStringSchema,
  contact_address: NullableStringSchema,
  privacy_policy_markdown: NullableStringSchema,
  terms_markdown: NullableStringSchema,
  privacy_effective_date: NullableStringSchema,
  terms_effective_date: NullableStringSchema,
});

/** Decode the complete generated envelope; v2 messages have no instance codec. */
function normalizeBranding(response: JsonValue): Record<string, unknown> {
  if (!response || typeof response !== 'object' || Array.isArray(response)) throw new Error('Invalid branding response');
  const branding = response.branding;
  if (!branding || typeof branding !== 'object' || Array.isArray(branding)) throw new Error('Missing branding response');
  return { ...branding, ...('id' in branding ? { id: Number(branding.id) } : {}) };
}

function decodePrivateBranding(response: BrandingResponse): SiteBranding {
  return parseOrThrow(SiteBrandingSchema, normalizeBranding(toJson(BrandingResponseSchema, response, { useProtoFieldName: true, alwaysEmitImplicit: true })), 'SiteBranding');
}

// Admin endpoints (require authentication)

export function getBranding() {
  return brandingClient.getBranding({}).then(decodePrivateBranding);
}

export function updateBranding(data: SiteBrandingUpdate) {
  const request = fromJsonString(UpdateBrandingRequestSchema, JSON.stringify(data), { ignoreUnknownFields: false });
  return withAdminReauthentication(() => brandingClient.updateBranding(request)).then(decodePrivateBranding);
}

export function clearBrandingField(field: string) {
  return withAdminReauthentication(() => brandingClient.clearBrandingField({ field })).then(decodePrivateBranding);
}

// Public endpoints (no auth required)

export function getPublicBranding() {
  return brandingClient.getPublicBranding({}).then((response: PublicBrandingResponse) => parseOrThrow(PublicBrandingSchema,
    normalizeBranding(toJson(PublicBrandingResponseSchema, response, { useProtoFieldName: true, alwaysEmitImplicit: true })), 'PublicBranding'));
}
