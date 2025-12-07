import { fromJson } from '@bufbuild/protobuf';
import { SubscriptionState, VerifySubscriptionResponseSchema } from '@proto-lprv/billing_pb';
import { apiCall } from './common';
import type { CreditInfo, EntitlementPayload, SubscriptionInfo } from './types';

export function getSubscriptionInfo() {
  return apiCall('/me/subscription').then((resp) => {
    const message = fromJson(VerifySubscriptionResponseSchema, resp, {
      ignoreUnknownFields: true,
      protoFieldName: true,
    });
    const status = message.status;
    const mapState = (state?: SubscriptionState) => {
      switch (state) {
        case SubscriptionState.SUBSCRIPTION_STATE_ACTIVE:
          return 'active';
        case SubscriptionState.SUBSCRIPTION_STATE_TRIALING:
          return 'trialing';
        case SubscriptionState.SUBSCRIPTION_STATE_PAST_DUE:
          return 'past_due';
        case SubscriptionState.SUBSCRIPTION_STATE_CANCELED:
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
      updated_at: status?.cachedAt?.toJsonString(),
    };
    return subscription;
  });
}

type CreditsEnvelope = {
  balance?: {
    customer_email?: string;
    balance_credits?: number;
    bundle_key?: string;
    updated_at?: string;
  };
  display_credits_label?: string;
  display_credits_multiplier?: number;
};

export function getCreditInfo() {
  return apiCall<CreditsEnvelope>('/me/credits').then((resp) => {
    const balance = resp?.balance ?? {};
    const credits: CreditInfo = {
      customer_email: balance.customer_email ?? '',
      balance_credits: balance.balance_credits ?? 0,
      bonus_credits: 0,
      display_credits_label: resp?.display_credits_label ?? 'credits',
      display_credits_multiplier: resp?.display_credits_multiplier ?? 1,
    };
    return credits;
  });
}

export function getEntitlements(userEmail?: string) {
  const params = new URLSearchParams();
  if (userEmail?.trim()) {
    params.set('user', userEmail.trim());
  }
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<EntitlementPayload>(`/entitlements${query}`);
}
