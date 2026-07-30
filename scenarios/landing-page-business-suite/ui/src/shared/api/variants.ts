import { createClient } from '@connectrpc/connect';
import { create, toJson, type JsonObject, type JsonValue } from '@bufbuild/protobuf';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { SeoService, type SEOResponse, type UpdateVariantSEOResponse } from '@vrooli/proto-types/landing-page-business-suite/seo_pb';
import {
  VariantService,
  VariantSchema as VariantMessageSchema,
  VariantSnapshotSchema as VariantSnapshotMessageSchema,
  type Variant as VariantMessage,
  type VariantSnapshot as VariantSnapshotMessage,
} from '@vrooli/proto-types/landing-page-business-suite/variant_pb';
import { VariantSectionService } from '@vrooli/proto-types/landing-page-business-suite/variant_section_pb';
import type { ContentSection as VariantSectionMessage } from '@vrooli/proto-types/landing-page-business-suite/shared/content_pb';
import { VariantSpaceService } from '@vrooli/proto-types/landing-page-business-suite/variant_space_pb';
import { CONNECT_API_BASE } from './common';
import { parseOrNull } from './safeParse';
import { isRecord } from '../lib/utils';
import {
  VariantSchema,
  VariantSnapshotSchema,
  VariantSpaceSchema,
  VariantSEOResponseSchema,
} from './schemas/variants.schema';
import { SuccessResponseSchema } from './schemas/common.schema';
import type {
  LandingHeaderConfig,
  Variant,
  VariantAxes,
  VariantSnapshot,
  VariantSEOConfig,
  VariantSEOResponse,
  ContentSection,
} from './types';

const variantSpaceClient = createClient(
  VariantSpaceService,
  createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }),
);
const seoClient = createClient(
  SeoService,
  createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }),
);
const variantClient = createClient(
  VariantService,
  createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }),
);
const variantSectionClient = createClient(
  VariantSectionService,
  createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }),
);

export interface VariantCreatePayload {
  name: string;
  slug: string;
  description?: string;
  weight?: number;
  axes: VariantAxes;
  header_config?: LandingHeaderConfig;
}

export interface VariantUpdatePayload {
  name?: string;
  description?: string;
  weight?: number;
  axes?: VariantAxes;
  header_config?: LandingHeaderConfig;
}

export function getPublicVariant(slug: string) {
  return variantClient.getPublicVariant({ slug }).then((response) => parseVariant(response.variant));
}

export function listVariants() {
  return variantClient.listVariants({}).then((response) => ({ variants: response.variants.map(parseVariant) }));
}

export function getVariant(slug: string) {
  return variantClient.getVariant({ slug }).then((response) => parseVariant(response.variant));
}

export function createVariant(data: VariantCreatePayload) {
  return variantClient.createVariant({
    slug: data.slug, name: data.name, description: data.description ?? '', weight: data.weight ?? 0, axes: data.axes,
  }).then((response) => parseVariant(response.variant));
}

export function updateVariant(slug: string, data: VariantUpdatePayload) {
  return variantClient.updateVariant({
    slug, name: data.name, description: data.description, weight: data.weight,
    axes: data.axes ? { values: data.axes } : undefined,
    headerConfig: data.header_config ? headerConfigFromLegacy(data.header_config) : undefined,
  }).then((response) => parseVariant(response.variant));
}

export function exportVariantSnapshot(slug: string) {
  return variantClient.exportVariantSnapshot({ slug }).then((response) => parseSnapshot(response.snapshot));
}

export function importVariantSnapshot(slug: string, snapshot: VariantSnapshot) {
  return variantClient.importVariantSnapshot({ slug, snapshot: snapshotToProto(snapshot) }).then((response) => parseSnapshot(response.snapshot));
}

export function archiveVariant(slug: string) {
  return variantClient.archiveVariant({ slug }).then((response) => parseVariant(response.variant));
}

export function deleteVariant(slug: string) {
  return variantClient.deleteVariant({ slug }).then((response) => ({ success: response.deleted }));
}

function parseVariantSection(message: VariantSectionMessage | undefined): ContentSection {
  if (!message?.key || !message.sectionType) {
    throw new Error('Variant section response did not include a stable key and section type');
  }
  return {
    id: 0,
    variant_id: 0,
    key: message.key,
    section_type: message.sectionType as ContentSection['section_type'],
    content: message.content ?? {},
    order: message.order,
    enabled: message.enabled,
    created_at: '',
    updated_at: '',
  };
}

function isJsonValue(value: unknown): value is JsonValue {
  if (value === null || typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return true;
  }
  if (Array.isArray(value)) {
    return value.every(isJsonValue);
  }
  return isRecord(value) && Object.values(value).every(isJsonValue);
}

function jsonObjectFromContent(content: Record<string, unknown>): JsonObject {
  if (!isJsonValue(content) || Array.isArray(content) || content === null) {
    throw new Error('Section content must be a JSON object');
  }
  return content;
}

export function getVariantSections(slug: string) {
  return variantSectionClient.getVariantSections({ slug }).then((response) => ({ sections: response.sections.map(parseVariantSection) }));
}

export function getVariantSection(slug: string, sectionKey: string) {
  return variantSectionClient.getVariantSection({ slug, sectionKey }).then((response) => parseVariantSection(response.section));
}

export function createVariantSection(slug: string, section: Pick<ContentSection, 'key' | 'section_type' | 'content' | 'order' | 'enabled'>) {
  return variantSectionClient.createVariantSection({
    slug,
    section: { key: section.key ?? '', sectionType: section.section_type, content: jsonObjectFromContent(section.content), order: section.order, enabled: section.enabled },
  }).then((response) => parseVariantSection(response.section));
}

export function updateVariantSection(slug: string, sectionKey: string, patch: Partial<Pick<ContentSection, 'section_type' | 'content' | 'order' | 'enabled'>>) {
  return variantSectionClient.updateVariantSection({
    slug,
    sectionKey,
    sectionType: patch.section_type,
    content: patch.content ? jsonObjectFromContent(patch.content) : undefined,
    order: patch.order,
    enabled: patch.enabled,
  }).then((response) => parseVariantSection(response.section));
}

export function deleteVariantSection(slug: string, sectionKey: string) {
  return variantSectionClient.deleteVariantSection({ slug, sectionKey }).then((response) => ({ success: response.deleted }));
}

export function selectVariant(_variantSlug?: string) {
  return variantClient.selectVariant({}).then((response) => parseVariant(response.variant));
}

function parseVariant(message: VariantMessage | undefined): Variant {
  if (!message) throw new Error('Variant response did not include a variant');
	const raw = toJson(VariantMessageSchema, message, { useProtoFieldName: true });
	if (!isRecord(raw)) throw new Error('Invalid variant response from API');
  if (raw.id === '0') delete raw.id;
  const validated = parseOrNull(VariantSchema, raw, 'Variant');
  if (!validated) throw new Error('Invalid variant response from API');
  return validated;
}

function parseSnapshot(message: VariantSnapshotMessage | undefined): VariantSnapshot {
  if (!message) throw new Error('Variant snapshot response did not include a snapshot');
	const raw = toJson(VariantSnapshotMessageSchema, message, { useProtoFieldName: true });
	if (!isRecord(raw)) throw new Error('Invalid variant snapshot response from API');
  const snapshot = {
    variant: {
      slug: raw.slug, name: raw.name, description: raw.description, weight: raw.weight, status: raw.status, axes: raw.axes ?? {},
      header_config: raw.header_config, seo_config: raw.seo_config,
    },
    sections: raw.sections ?? [],
  };
  const validated = parseOrNull(VariantSnapshotSchema, snapshot, 'VariantSnapshot');
  if (!validated) throw new Error('Invalid variant snapshot response from API');
  return validated;
}

function snapshotToProto(snapshot: VariantSnapshot): VariantSnapshotMessage {
	return create(VariantSnapshotMessageSchema, {
    slug: snapshot.variant.slug, name: snapshot.variant.name, description: snapshot.variant.description ?? '', weight: snapshot.variant.weight ?? 0,
	status: snapshot.variant.status ?? 'active', axes: snapshot.variant.axes, headerConfig: snapshot.variant.header_config ? headerConfigFromLegacy(snapshot.variant.header_config) : undefined,
    sections: snapshot.sections.map((section) => ({ id: 0n, variantId: 0n, sectionType: section.section_type, content: { fields: section.content }, order: section.order, enabled: section.enabled ?? true })),
	});
}

function headerConfigFromLegacy(config: LandingHeaderConfig) {
  return {
    branding: { mode: config.branding.mode, label: config.branding.label ?? '', subtitle: config.branding.subtitle ?? '', mobilePreference: config.branding.mobile_preference ?? '' },
    nav: { links: config.nav.links.map((link) => headerLinkFromLegacy(link)) },
    ctas: { primary: { mode: config.ctas.primary.mode, label: config.ctas.primary.label ?? '', href: config.ctas.primary.href ?? '', variant: config.ctas.primary.variant ?? '' }, secondary: { mode: config.ctas.secondary.mode, label: config.ctas.secondary.label ?? '', href: config.ctas.secondary.href ?? '', variant: config.ctas.secondary.variant ?? '' } },
    behavior: { sticky: config.behavior.sticky, hideOnScroll: config.behavior.hide_on_scroll },
  };
}

function headerLinkFromLegacy(link: NonNullable<LandingHeaderConfig['nav']['links']>[number]): Record<string, unknown> {
  return { id: link.id, type: link.type, label: link.label, sectionType: link.section_type ?? '', sectionId: link.section_id, anchor: link.anchor ?? '', href: link.href ?? '', visibleOn: { desktop: link.visible_on?.desktop ?? true, mobile: link.visible_on?.mobile ?? true }, children: link.children?.map(headerLinkFromLegacy) ?? [] };
}

export async function getVariantSpace() {
  const response = await variantSpaceClient.getVariantSpace({});
  let rawVariantSpace: unknown;
  try {
    rawVariantSpace = JSON.parse(new TextDecoder().decode(response.rawJson));
  } catch {
    throw new Error('Invalid variant space response from API');
  }
  const validated = parseOrNull(VariantSpaceSchema, rawVariantSpace, 'VariantSpace');
  if (!validated) {
    throw new Error('Invalid variant space response from API');
  }
  return validated;
}

function normalizeVariantSEOResponse(response: SEOResponse): VariantSEOResponse {
  return {
    site_name: response.siteName,
    title: response.title,
    description: response.description,
    og_title: response.ogTitle,
    og_description: response.ogDescription,
    og_image_url: response.ogImageUrl || undefined,
    twitter_card: response.twitterCard || undefined,
    canonical_url: response.canonicalUrl || undefined,
    favicon_url: response.faviconUrl || undefined,
    apple_touch_icon_url: response.appleTouchIconUrl || undefined,
    theme_primary_color: response.themePrimaryColor || undefined,
    noindex: response.noindex,
    structured_data: response.structuredData,
  };
}

export function getVariantSEO(slug: string) {
  return seoClient.getVariantSEO({ slug }).then((response: SEOResponse) => {
    const validated = parseOrNull(VariantSEOResponseSchema, normalizeVariantSEOResponse(response), 'VariantSEOResponse');
    if (!validated) {
      throw new Error('Invalid variant SEO response from API');
    }
    return validated;
  });
}

export function updateVariantSEO(slug: string, config: VariantSEOConfig) {
  return seoClient.updateVariantSEO({
    slug,
    config: {
      title: config.title,
      description: config.description,
      ogTitle: config.og_title,
      ogDescription: config.og_description,
      ogImageUrl: config.og_image_url,
      twitterCard: config.twitter_card,
      canonicalPath: config.canonical_path,
      noindex: config.noindex,
      structuredData: config.structured_data,
    },
  }).then((response: UpdateVariantSEOResponse) => {
    const validated = parseOrNull(SuccessResponseSchema, { success: response.success }, 'UpdateVariantSEOResponse');
    if (!validated) {
      throw new Error('Invalid update variant SEO response from API');
    }
    return validated;
  });
}
