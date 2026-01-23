import { Layers } from 'lucide-react';
import { ComingSoonPage } from '../components/ComingSoonPage';

/**
 * Tiers Management - Stub page for managing subscription tiers
 *
 * This is a placeholder page that will be implemented to allow admins to:
 * - Create and manage subscription tiers
 * - Configure tier features and pricing
 * - Set up tier-based access controls
 */
export function TiersManagement() {
  return (
    <ComingSoonPage
      title="Subscription Tiers"
      description="Configure subscription plans and tier features"
      icon={Layers}
      iconBgClass="bg-purple-500/10"
      iconColorClass="text-purple-400"
      testId="tiers-management"
      intro="The Tiers Management page will allow you to:"
      features={[
        'Create and edit subscription tier definitions',
        'Configure features and benefits per tier',
        'Set pricing and billing cycles for each tier',
        'Define upgrade/downgrade paths between tiers',
        'Manage trial periods and promotional offers',
      ]}
      alternatives="In the meantime, use <strong>Tier Limits</strong> to configure credit limits per tier, and <strong>Billing</strong> to manage Stripe price configurations."
    />
  );
}
