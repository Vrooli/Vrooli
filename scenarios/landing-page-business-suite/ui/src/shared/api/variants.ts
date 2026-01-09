import { apiCall } from './common';
import type {
  LandingHeaderConfig,
  Variant,
  VariantAxes,
  VariantSpace,
  VariantSnapshot,
  VariantSEOConfig,
  VariantSEOResponse,
} from './types';

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
  return apiCall<Variant>(`/public/variants/${slug}`);
}

export function listVariants() {
  return apiCall<{ variants: Variant[] }>('/variants');
}

export function getVariant(slug: string) {
  return apiCall<Variant>(`/variants/${slug}`);
}

export function createVariant(data: VariantCreatePayload) {
  return apiCall<Variant>('/variants', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export function updateVariant(slug: string, data: VariantUpdatePayload) {
  return apiCall<Variant>(`/variants/${slug}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export function exportVariantSnapshot(slug: string) {
  return apiCall<VariantSnapshot>(`/admin/variants/${slug}/export`);
}

export function importVariantSnapshot(slug: string, snapshot: VariantSnapshot) {
  return apiCall<VariantSnapshot>(`/admin/variants/${slug}/import`, {
    method: 'PUT',
    body: JSON.stringify(snapshot),
  });
}

export function archiveVariant(slug: string) {
  return apiCall<{ success: boolean }>(`/variants/${slug}/archive`, {
    method: 'POST',
  });
}

export function deleteVariant(slug: string) {
  return apiCall<{ success: boolean }>(`/variants/${slug}`, {
    method: 'DELETE',
  });
}

export function selectVariant(variantSlug?: string) {
  const query = variantSlug ? `?variant_slug=${variantSlug}` : '';
  return apiCall<Variant>(`/variants/select${query}`);
}

export function getVariantSpace() {
  return apiCall<VariantSpace>('/variant-space');
}

export function getVariantSEO(slug: string) {
  return apiCall<VariantSEOResponse>(`/seo/${slug}`);
}

export function updateVariantSEO(slug: string, config: VariantSEOConfig) {
  return apiCall<{ success?: boolean; updated_at?: string }>(`/admin/variants/${slug}/seo`, {
    method: 'PUT',
    body: JSON.stringify(config),
    credentials: 'include',
  });
}
