import { apiCall } from './common';
import type { Variant, VariantAxes, VariantSpace } from './types';

export interface VariantCreatePayload {
  name: string;
  slug: string;
  description?: string;
  weight?: number;
  axes: VariantAxes;
}

export interface VariantUpdatePayload {
  name?: string;
  description?: string;
  weight?: number;
  axes?: VariantAxes;
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
