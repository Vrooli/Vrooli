import { createClient } from '@connectrpc/connect';
import { AccountService, type GetEntitlementsResponse, type GetMyCreditsResponse, type GetMySubscriptionResponse } from '@vrooli/proto-types/landing-page-business-suite/v1/account_pb';
import { SubscriptionState } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/commerce_pb';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { CONNECT_API_BASE } from './common';
import { parseOrNull } from './safeParse';
import {
  SubscriptionInfoSchema,
  CreditInfoSchema,
  EntitlementPayloadSchema,
} from './schemas/billing.schema';
import type { CreditInfo, SubscriptionInfo } from './types';

const accountClient = createClient(AccountService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

export function getSubscriptionInfo() {
  return accountClient.getMySubscription({}).then((message: GetMySubscriptionResponse) => {
    const status = message.status;
    const mapState = (state?: SubscriptionState) => {
      switch (state) {
        case SubscriptionState.ACTIVE:
          return 'active';
        case SubscriptionState.TRIALING:
          return 'trialing';
        case SubscriptionState.PAST_DUE:
          return 'past_due';
        case SubscriptionState.CANCELED:
          return 'canceled';
        default:
          return 'inactive';
      }
    };

    const subscription: SubscriptionInfo = {
      status: mapState(status?.state),
      subscription_id: status?.subscriptionId,
      customer_email: status?.userIdentity,
      plan_tier: status?.planTier,
      price_id: status?.stripePriceId,
      bundle_key: status?.bundleKey,
      updated_at: status?.cachedAt?.toJsonString?.(),
    };
    const validated = parseOrNull(SubscriptionInfoSchema, subscription, 'SubscriptionInfo');
    if (!validated) {
      throw new Error('Invalid subscription info response from API');
    }
    return validated;
  });
}

export function getCreditInfo() {
  return accountClient.getMyCredits({}).then((resp: GetMyCreditsResponse) => {
    const balance = resp.balance ?? {};
    const credits: CreditInfo = {
      customer_email: balance.customerEmail ?? '',
      balance_credits: Number(balance.balanceCredits ?? 0),
      bonus_credits: 0,
      display_credits_label: resp.displayCreditsLabel ?? 'credits',
      display_credits_multiplier: resp.displayCreditsMultiplier ?? 1,
    };
    const validated = parseOrNull(CreditInfoSchema, credits, 'CreditInfo');
    if (!validated) {
      throw new Error('Invalid credit info response from API');
    }
    return validated;
  });
}

export function getEntitlements() {
  return accountClient.getEntitlements({}).then((resp: GetEntitlementsResponse) => {
    const mapState = (state?: SubscriptionState) => {
      switch (state) {
        case SubscriptionState.ACTIVE: return 'active';
        case SubscriptionState.TRIALING: return 'trialing';
        case SubscriptionState.PAST_DUE: return 'past_due';
        case SubscriptionState.CANCELED: return 'canceled';
        default: return 'inactive';
      }
    };
    // The wire carries generated proto messages; project them onto the UI's
    // snake_case types before validation. Entitlement credits are the raw
    // internal balance without display facts, so display fields default and
    // balance displays belong to getCreditInfo().
    const subscription = resp.subscription ? {
      status: mapState(resp.subscription.state),
      subscription_id: resp.subscription.subscriptionId,
      customer_email: resp.subscription.userIdentity,
      plan_tier: resp.subscription.planTier,
      price_id: resp.subscription.stripePriceId,
      bundle_key: resp.subscription.bundleKey,
    } : undefined;
    const credits = resp.credits ? {
      customer_email: resp.credits.customerEmail ?? '',
      balance_credits: Number(resp.credits.balanceCredits ?? 0),
      bonus_credits: 0,
      display_credits_label: 'credits',
      display_credits_multiplier: 1,
    } : undefined;
    const validated = parseOrNull(EntitlementPayloadSchema, {
      status: resp.status,
      plan_tier: resp.planTier,
      price_id: resp.priceId,
      features: resp.features,
      credits,
      subscription,
      billing_cycle_start: resp.billingCycleStart,
    }, 'EntitlementPayload');
    if (!validated) {
      throw new Error('Invalid entitlement payload response from API');
    }
    return validated;
  });
}
