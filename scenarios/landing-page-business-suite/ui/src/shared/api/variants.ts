import { apiCall } from './common';
import { parseOrNull } from './safeParse';
import {
  VariantSchema,
  VariantListResponseSchema,
  VariantSnapshotSchema,
  VariantSpaceSchema,
  VariantSEOResponseSchema,
} from './schemas/variants.schema';
import { SuccessResponseSchema } from './schemas/common.schema';
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
  return apiCall<Variant>(`/public/variants/${slug}`).then((resp) => {
    const validated = parseOrNull(VariantSchema, resp, 'Variant');
    if (!validated) {
      throw new Error('Invalid variant response from API');
    }
    return validated;
  });
}

export function listVariants() {
  return apiCall<{ variants: Variant[] }>('/variants').then((resp) => {
    const validated = parseOrNull(VariantListResponseSchema, resp, 'VariantListResponse');
    if (!validated) {
      return { variants: [] };
    }
    return validated;
  });
}

export function getVariant(slug: string) {
  return apiCall<Variant>(`/variants/${slug}`).then((resp) => {
    const validated = parseOrNull(VariantSchema, resp, 'Variant');
    if (!validated) {
      throw new Error('Invalid variant response from API');
    }
    return validated;
  });
}

export function createVariant(data: VariantCreatePayload) {
  return apiCall<Variant>('/variants', {
    method: 'POST',
    body: JSON.stringify(data),
  }).then((resp) => {
    const validated = parseOrNull(VariantSchema, resp, 'Variant');
    if (!validated) {
      throw new Error('Invalid variant response from API');
    }
    return validated;
  });
}

export function updateVariant(slug: string, data: VariantUpdatePayload) {
  return apiCall<Variant>(`/variants/${slug}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  }).then((resp) => {
    const validated = parseOrNull(VariantSchema, resp, 'Variant');
    if (!validated) {
      throw new Error('Invalid variant response from API');
    }
    return validated;
  });
}

export function exportVariantSnapshot(slug: string) {
  return apiCall<VariantSnapshot>(`/admin/variants/${slug}/export`).then((resp) => {
    const validated = parseOrNull(VariantSnapshotSchema, resp, 'VariantSnapshot');
    if (!validated) {
      throw new Error('Invalid variant snapshot response from API');
    }
    return validated;
  });
}

export function importVariantSnapshot(slug: string, snapshot: VariantSnapshot) {
  return apiCall<VariantSnapshot>(`/admin/variants/${slug}/import`, {
    method: 'PUT',
    body: JSON.stringify(snapshot),
  }).then((resp) => {
    const validated = parseOrNull(VariantSnapshotSchema, resp, 'VariantSnapshot');
    if (!validated) {
      throw new Error('Invalid variant snapshot response from API');
    }
    return validated;
  });
}

export function archiveVariant(slug: string) {
  return apiCall<{ success: boolean }>(`/variants/${slug}/archive`, {
    method: 'POST',
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'ArchiveVariantResponse');
    if (!validated) {
      throw new Error('Invalid archive variant response from API');
    }
    return validated;
  });
}

export function deleteVariant(slug: string) {
  return apiCall<{ success: boolean }>(`/variants/${slug}`, {
    method: 'DELETE',
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'DeleteVariantResponse');
    if (!validated) {
      throw new Error('Invalid delete variant response from API');
    }
    return validated;
  });
}

export function selectVariant(variantSlug?: string) {
  const query = variantSlug ? `?variant_slug=${variantSlug}` : '';
  return apiCall<Variant>(`/variants/select${query}`).then((resp) => {
    const validated = parseOrNull(VariantSchema, resp, 'Variant');
    if (!validated) {
      throw new Error('Invalid variant selection response from API');
    }
    return validated;
  });
}

export function getVariantSpace() {
  return apiCall<VariantSpace>('/variant-space').then((resp) => {
    const validated = parseOrNull(VariantSpaceSchema, resp, 'VariantSpace');
    if (!validated) {
      throw new Error('Invalid variant space response from API');
    }
    return validated;
  });
}

export function getVariantSEO(slug: string) {
  return apiCall<VariantSEOResponse>(`/seo/${slug}`).then((resp) => {
    const validated = parseOrNull(VariantSEOResponseSchema, resp, 'VariantSEOResponse');
    if (!validated) {
      throw new Error('Invalid variant SEO response from API');
    }
    return validated;
  });
}

export function updateVariantSEO(slug: string, config: VariantSEOConfig) {
  return apiCall<{ success?: boolean; updated_at?: string }>(`/admin/variants/${slug}/seo`, {
    method: 'PUT',
    body: JSON.stringify(config),
    credentials: 'include',
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'UpdateVariantSEOResponse');
    if (!validated) {
      throw new Error('Invalid update variant SEO response from API');
    }
    return validated;
  });
}
