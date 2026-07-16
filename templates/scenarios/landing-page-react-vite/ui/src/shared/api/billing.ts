import { create, type MessageInitShape } from '@bufbuild/protobuf';
import { createClient } from '@connectrpc/connect';
import { LandingPagePaymentsService } from '@vrooli/proto-types/landing-page-react-vite/v1/billing_pb';
import {
  ConfigSource,
  GetStripeSettingsResponseSchema,
  UpdateStripeSettingsRequestSchema,
} from '@vrooli/proto-types/landing-page-react-vite/v1/settings_pb';
import type {
  GetStripeSettingsResponse,
  StripeSettings,
  StripeConfigSnapshot,
} from '@vrooli/proto-types/landing-page-react-vite/v1/settings_pb';
import { BundleAdminService } from '@vrooli/proto-types/landing-page-react-vite/v1/bundles_pb';
import type { BundleCatalogEntry } from '@vrooli/proto-types/landing-page-react-vite/v1/bundles_pb';
import type { PlanOption } from '@vrooli/proto-types/landing-page-react-vite/v1/pricing_pb';

import { transport } from './client';

const paymentsClient = createClient(LandingPagePaymentsService, transport);
const bundleClient = createClient(BundleAdminService, transport);

export type StripeSettingsUpdate = MessageInitShape<typeof UpdateStripeSettingsRequestSchema>;
export interface BundlePriceUpdate {
  planName?: string;
  displayWeight?: number;
  displayEnabled?: boolean;
  subtitle?: string;
  badge?: string;
  ctaLabel?: string;
  highlight?: boolean;
  features?: string[];
}

/** Fetches the current Stripe settings and redacted key snapshot (admin). */
export function getStripeSettings(): Promise<GetStripeSettingsResponse> {
  return paymentsClient.getStripeSettings({});
}

/** Updates Stripe settings; only provided fields change (admin). */
export async function updateStripeSettings(
  data: StripeSettingsUpdate,
): Promise<GetStripeSettingsResponse> {
  const resp = await paymentsClient.updateStripeSettings(data);
  // UpdateStripeSettingsResponse mirrors GetStripeSettingsResponse; normalize to
  // the single shape the settings UI consumes.
  return create(GetStripeSettingsResponseSchema, {
    settings: resp.settings,
    snapshot: resp.snapshot,
  });
}

/** Lists the bundle catalog with per-bundle prices (admin). */
export async function getBundleCatalog(): Promise<BundleCatalogEntry[]> {
  const resp = await bundleClient.listBundleCatalog({});
  return resp.bundles;
}

/** Updates display metadata for a single bundle price (admin). */
export async function updateBundlePrice(
  bundleKey: string,
  priceId: string,
  data: BundlePriceUpdate,
): Promise<PlanOption | undefined> {
  const resp = await bundleClient.updateBundlePrice({ ...data, bundleKey, priceId });
  return resp.price;
}

export { ConfigSource };
export type {
  GetStripeSettingsResponse,
  StripeSettings,
  StripeConfigSnapshot,
  BundleCatalogEntry,
};
