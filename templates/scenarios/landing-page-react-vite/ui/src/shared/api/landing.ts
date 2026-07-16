import { createClient } from '@connectrpc/connect';
import { LandingConfigService } from '@vrooli/proto-types/landing-page-react-vite/v1/config_pb';
import type {
  LandingConfigResponse,
  LandingVariantSummary,
  LandingSection,
  LandingBranding,
} from '@vrooli/proto-types/landing-page-react-vite/v1/config_pb';
import { LandingPagePaymentsService } from '@vrooli/proto-types/landing-page-react-vite/v1/billing_pb';
import {
  BillingInterval,
  IntroPricingType,
  PlanKind,
} from '@vrooli/proto-types/landing-page-react-vite/v1/pricing_pb';
import type {
  PricingOverview,
  PlanOption,
  Bundle,
} from '@vrooli/proto-types/landing-page-react-vite/v1/pricing_pb';

import { transport } from './client';

const configClient = createClient(LandingConfigService, transport);
const paymentsClient = createClient(LandingPagePaymentsService, transport);

/** Fetches the aggregated public landing payload for a variant (or a selected one). */
export function getLandingConfig(variantSlug?: string): Promise<LandingConfigResponse> {
  return configClient.getLandingConfig({ variantSlug: variantSlug ?? '' });
}

/** Fetches the pricing overview (bundle + monthly/yearly plans) for the pricing surface. */
export async function getPlans(): Promise<PricingOverview | undefined> {
  const resp = await paymentsClient.getPricing({});
  return resp.pricing;
}

export { BillingInterval, IntroPricingType, PlanKind };
export type {
  LandingConfigResponse,
  LandingVariantSummary,
  LandingSection,
  LandingBranding,
  PricingOverview,
  PlanOption,
  Bundle,
};
