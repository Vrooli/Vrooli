import { Layers, Settings, Wrench } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';

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
    <AdminLayout>
      <div className="max-w-4xl mx-auto space-y-6">
        <PageHeader
          variant="icon-title"
          title="Subscription Tiers"
          description="Configure subscription plans and tier features"
          icon={Layers}
          iconBgClass="bg-purple-500/10"
          iconColorClass="text-purple-400"
          testId="tiers-management-header"
        />

        <Card className="bg-white/5 border-white/10">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wrench className="h-5 w-5 text-amber-400" />
              Coming Soon
            </CardTitle>
            <CardDescription className="text-slate-400">
              This feature is under development
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-slate-300">
              The Tiers Management page will allow you to:
            </p>
            <ul className="list-disc list-inside space-y-2 text-slate-400">
              <li>Create and edit subscription tier definitions</li>
              <li>Configure features and benefits per tier</li>
              <li>Set pricing and billing cycles for each tier</li>
              <li>Define upgrade/downgrade paths between tiers</li>
              <li>Manage trial periods and promotional offers</li>
            </ul>

            <div className="mt-6 p-4 bg-slate-900/50 rounded-lg border border-white/10">
              <div className="flex items-center gap-2 text-sm text-slate-400">
                <Settings className="h-4 w-4" />
                <span>
                  In the meantime, use <strong>Tier Limits</strong> to configure credit limits per tier,
                  and <strong>Billing</strong> to manage Stripe price configurations.
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </AdminLayout>
  );
}
