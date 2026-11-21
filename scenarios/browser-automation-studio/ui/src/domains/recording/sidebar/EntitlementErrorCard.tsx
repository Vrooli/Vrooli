import { useCallback } from 'react';
import * as MonetizationAccount from '@components/MonetizationAccount';
import type { SubscriptionTier } from '@stores/entitlementStore';
import type { EntitlementErrorCode, EntitlementErrorDetails } from './types';

const SUBSCRIPTION_SETTINGS_URL = '/settings?tab=subscription';

export interface EntitlementErrorCardProps {
  errorCode: EntitlementErrorCode;
  details?: EntitlementErrorDetails;
}

export function EntitlementErrorCard({ errorCode, details }: EntitlementErrorCardProps) {
  const navigateToSubscription = useCallback(() => {
    window.location.assign(SUBSCRIPTION_SETTINGS_URL);
  }, []);
  const tier = (details?.tier as SubscriptionTier) || 'free';
  const isTierError = errorCode === 'AI_NOT_AVAILABLE';
  const isCreditError = errorCode === 'INSUFFICIENT_CREDITS';
  if (!isTierError && !isCreditError) return null;

  return (
    <MonetizationAccount.EntitlementErrorCard
      errorType={errorCode}
      title={isTierError ? 'AI features not available' : 'AI credits exhausted'}
      message={isTierError ? "Your current plan doesn't include AI-powered browser automation." : "You've used all your AI credits for this month."}
      plan={tier}
      creditsUsed={details?.creditsUsed}
      creditsLimit={details?.creditsLimit}
      resetDate={details?.resetDate}
      onManage={navigateToSubscription}
    />
  );
}

export default EntitlementErrorCard;
