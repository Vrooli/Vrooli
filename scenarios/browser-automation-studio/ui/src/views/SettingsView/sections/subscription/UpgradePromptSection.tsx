import * as MonetizationAccount from '@components/MonetizationAccount';
import { useEntitlementStore } from '@stores/entitlementStore';

const landingPageEnv = (import.meta.env as { VITE_LANDING_PAGE_URL?: unknown }).VITE_LANDING_PAGE_URL;
const LANDING_PAGE_URL = typeof landingPageEnv === 'string' && landingPageEnv.length > 0 ? landingPageEnv : 'https://vrooli.com';

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
