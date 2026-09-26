import { createClient } from '@connectrpc/connect';
import {
  VariantService,
} from '@vrooli/proto-types/landing-page-react-vite/v1/variant_pb';
import type {
  Variant,
  VariantSnapshot,
  LandingHeaderConfig,
  VariantSEOConfig,
  HeaderBrandingConfig,
  HeaderNavConfig,
  HeaderNavLink,
  HeaderVisibilityConfig,
  HeaderCTAGroup,
  HeaderCTAConfig,
  HeaderBehaviorConfig,
} from '@vrooli/proto-types/landing-page-react-vite/v1/variant_pb';
import { VariantSpaceService } from '@vrooli/proto-types/landing-page-react-vite/v1/variant_space_pb';

import { transport } from './client';

const variantClient = createClient(VariantService, transport);
const variantSpaceClient = createClient(VariantSpaceService, transport);

export interface VariantCreatePayload {
  name: string;
  slug: string;
  description?: string;
  weight?: number;
  axes: Record<string, string>;
}

export interface VariantUpdatePayload {
  name?: string;
  description?: string;
  weight?: number;
  axes?: Record<string, string>;
  headerConfig?: LandingHeaderConfig;
}

/** Returns a weighted-random active variant (public). */
export async function selectVariant(): Promise<Variant | undefined> {
  const resp = await variantClient.selectVariant({});
  return resp.variant;
}

/** Fetches an active variant by slug (public). */
export async function getPublicVariant(slug: string): Promise<Variant | undefined> {
  const resp = await variantClient.getPublicVariant({ slug });
  return resp.variant;
}

/** Fetches a variant by slug including SEO overrides (admin). */
export async function getVariant(slug: string): Promise<Variant | undefined> {
  const resp = await variantClient.getVariant({ slug });
  return resp.variant;
}

/** Lists variants, optionally filtered by status (admin). */
export async function listVariants(statusFilter?: string): Promise<Variant[]> {
  const resp = await variantClient.listVariants({ statusFilter: statusFilter ?? '' });
  return resp.variants;
}

/** Creates a new active variant (admin). */
export async function createVariant(data: VariantCreatePayload): Promise<Variant | undefined> {
  const resp = await variantClient.createVariant({
    slug: data.slug,
    name: data.name,
    description: data.description ?? '',
    weight: data.weight ?? 0,
    axes: data.axes,
  });
  return resp.variant;
}

/** Partially updates a variant (admin). Axes replace only when provided. */
export async function updateVariant(
  slug: string,
  data: VariantUpdatePayload,
): Promise<Variant | undefined> {
  const resp = await variantClient.updateVariant({
    slug,
    name: data.name,
    description: data.description,
    weight: data.weight,
    axes: data.axes ? { values: data.axes } : undefined,
    headerConfig: data.headerConfig,
  });
  return resp.variant;
}

/** Exports a variant snapshot by slug (admin). */
export async function exportVariantSnapshot(slug: string): Promise<VariantSnapshot | undefined> {
  const resp = await variantClient.exportVariantSnapshot({ slug });
  return resp.snapshot;
}

/** Replaces a variant from a snapshot (admin). */
export async function importVariantSnapshot(
  slug: string,
  snapshot: VariantSnapshot,
): Promise<Variant | undefined> {
  const resp = await variantClient.importVariantSnapshot({ slug, snapshot });
  return resp.variant;
}

/** Archives a variant by slug (admin). */
export async function archiveVariant(slug: string): Promise<Variant | undefined> {
  const resp = await variantClient.archiveVariant({ slug });
  return resp.variant;
}

/** Soft-deletes a variant by slug (admin). */
export async function deleteVariant(slug: string): Promise<boolean> {
  const resp = await variantClient.deleteVariant({ slug });
  return resp.deleted;
}

/**
 * VariantSpace is the schemaless axis catalog authored as JSON on the server
 * and returned as raw bytes by GetVariantSpace. It is intentionally not a proto
 * message — the axis shape is author-defined — so it is parsed here.
 */
export interface VariantSpaceAxisVariant {
  id: string;
  label: string;
  description?: string;
  examples?: Record<string, string>;
  defaultWeight?: number;
  tags?: string[];
  status?: 'active' | 'experimental' | 'deprecated';
  agentHints?: string[];
}

export interface VariantSpaceAxis {
  _note?: string;
  variants: VariantSpaceAxisVariant[];
}

export interface VariantSpace {
  _name: string;
  _schemaVersion: number;
  _note?: string;
  _agentGuidelines?: string[];
  axes: Record<string, VariantSpaceAxis>;
  constraints?: {
    _note?: string;
    disallowedCombinations?: Record<string, string>[];
  };
}

/** Fetches the variant space (axis catalog) and decodes its raw JSON payload. */
export async function getVariantSpace(): Promise<VariantSpace> {
  const resp = await variantSpaceClient.getVariantSpace({});
  const text = new TextDecoder().decode(resp.rawJson);
  return JSON.parse(text) as VariantSpace;
}

/** Axis id -> selected variant value. Matches the proto `map<string,string>` axes shape. */
export type VariantAxes = Record<string, string>;

export type { Variant, VariantSnapshot, LandingHeaderConfig, VariantSEOConfig };
export type {
  HeaderBrandingConfig,
  HeaderNavConfig,
  HeaderVisibilityConfig,
  HeaderCTAGroup,
  HeaderCTAConfig,
  HeaderBehaviorConfig,
};
// The UI historically named header nav links `LandingHeaderNavLink`; the proto
// message is `HeaderNavLink`. Alias for source compatibility.
export type { HeaderNavLink as LandingHeaderNavLink };
export type { HeaderNavLink };

