import type { MessageInitShape } from '@bufbuild/protobuf';
import { createClient } from '@connectrpc/connect';
import {
  SeoService,
  UpdateVariantSEORequestSchema,
} from '@vrooli/proto-types/landing-page-react-vite/v1/seo_pb';
import type {
  SEOResponse,
  UpdateVariantSEOResponse,
} from '@vrooli/proto-types/landing-page-react-vite/v1/seo_pb';
import type { VariantSEOConfig } from '@vrooli/proto-types/landing-page-react-vite/v1/variant_pb';

import { transport } from './client';

const seoClient = createClient(SeoService, transport);

export type VariantSEOConfigInput = NonNullable<
  MessageInitShape<typeof UpdateVariantSEORequestSchema>['config']
>;

/** Fetches the resolved SEO payload for a variant slug (public). */
export function getVariantSEO(slug: string): Promise<SEOResponse> {
  return seoClient.getVariantSEO({ slug });
}

/** Updates a variant's SEO overrides (admin). */
export function updateVariantSEO(
  slug: string,
  config: VariantSEOConfigInput,
): Promise<UpdateVariantSEOResponse> {
  return seoClient.updateVariantSEO({ slug, config });
}

export type { SEOResponse, UpdateVariantSEOResponse, VariantSEOConfig };
