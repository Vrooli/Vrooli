import * as MonetizationAccount from '@components/MonetizationAccount';
import { useEntitlementStore } from '@stores/entitlementStore';
import { LANDING_PAGE_URL } from '@shared/upgradeDestination';

/** BAS adapter for the RCL upgrade primitive; plan decisions stay in the signed lease. */
export function UpgradePromptSection() {
  const { status, userEmail } = useEntitlementStore();
  if (status?.tier === 'studio' || status?.tier === 'business') return null;

  const checkoutUrl = new URL('/checkout', LANDING_PAGE_URL);
  checkoutUrl.searchParams.set('plan', 'pro');
  if (userEmail) checkoutUrl.searchParams.set('email', userEmail);

  return (
    <MonetizationAccount.UpgradePrompt
      feature="BAS premium features"
      requiredPlan="pro"
      href={checkoutUrl.toString()}
    />
  );
}

export default UpgradePromptSection;
