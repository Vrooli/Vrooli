import { apiCall } from './common';
import type { CreditInfo, EntitlementPayload, SubscriptionInfo } from './types';

export function getSubscriptionInfo() {
  return apiCall<SubscriptionInfo>('/me/subscription');
}

export function getCreditInfo() {
  return apiCall<CreditInfo>('/me/credits');
}

export function getEntitlements(userEmail?: string) {
  const params = new URLSearchParams();
  if (userEmail?.trim()) {
    params.set('user', userEmail.trim());
  }
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<EntitlementPayload>(`/entitlements${query}`);
}
