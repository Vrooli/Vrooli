import {
  getVariantSEO,
  updateVariantSEO,
  type SiteBranding,
  type VariantSEOConfig,
  type VariantSEOResponse,
} from '../../../shared/api';

function valueOrUndefined(value?: string | null, fallback?: string | null) {
  const trimmed = value?.trim() ?? '';
  if (!trimmed) return undefined;
  if (fallback && trimmed === fallback.trim()) return undefined;
  return trimmed;
}

export function buildEditableSEOConfig(
  response: VariantSEOResponse,
  branding?: SiteBranding | null
): VariantSEOConfig {
  return {
    title: valueOrUndefined(response.title, branding?.default_title),
    description: valueOrUndefined(response.description, branding?.default_description),
    og_title: valueOrUndefined(response.og_title),
    og_description: valueOrUndefined(response.og_description),
    og_image_url: valueOrUndefined(response.og_image_url, branding?.default_og_image_url),
    twitter_card: (response.twitter_card as VariantSEOConfig['twitter_card']) || undefined,
    canonical_path: undefined, // API does not expose path today; keep undefined instead of guessing
    noindex: response.noindex ?? false,
    structured_data: response.structured_data ?? undefined,
  };
}

export async function loadVariantSEOConfig(
  slug: string,
  branding?: SiteBranding | null
): Promise<VariantSEOConfig> {
  const response = await getVariantSEO(slug);
  return buildEditableSEOConfig(response, branding);
}

export async function saveVariantSEOConfig(slug: string, config: VariantSEOConfig) {
  await updateVariantSEO(slug, config);
}
