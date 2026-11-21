import { createClient } from '@connectrpc/connect';
import { AccountService } from '@vrooli/proto-types/landing-page-react-vite/v1/account_pb';
import type {
  GetMyCreditsResponse,
  GetEntitlementsResponse,
} from '@vrooli/proto-types/landing-page-react-vite/v1/account_pb';
import {
  SubscriptionState,
} from '@vrooli/proto-types/landing-page-react-vite/v1/billing_pb';
import type {
  VerifySubscriptionResponse,
  SubscriptionStatus,
  CreditsBalance,
} from '@vrooli/proto-types/landing-page-react-vite/v1/billing_pb';

import { transport } from './client';

const accountClient = createClient(AccountService, transport);

/** Fetches the session user's current subscription status. */
export async function getSubscriptionInfo(): Promise<SubscriptionStatus | undefined> {
  const resp: VerifySubscriptionResponse = await accountClient.getMySubscription({});
  return resp.status;
}

/** Fetches the session user's credit wallet summary. */
export function getCreditInfo(): Promise<GetMyCreditsResponse> {
  return accountClient.getMyCredits({});
}

/** Fetches the session user's entitlements (features, plan, credits). */
export function getEntitlements(): Promise<GetEntitlementsResponse> {
  return accountClient.getEntitlements({});
}

export { SubscriptionState };
export type {
  VerifySubscriptionResponse,
  SubscriptionStatus,
  CreditsBalance,
  GetMyCreditsResponse,
  GetEntitlementsResponse,
};
