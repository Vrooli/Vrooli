import type { ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';
import { Wrench, Settings } from 'lucide-react';
import { AdminLayout } from './AdminLayout';
import { PageHeader } from './PageHeader';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { LAYOUT } from '../config/layout.constants';

export interface ComingSoonPageProps {
  /** Page title */
  title: string;
  /** Page description */
  description: string;
  /** Lucide icon component for the page header */
  icon: LucideIcon;
  /** Tailwind background class for icon container */
  iconBgClass: string;
  /** Tailwind text color class for icon */
  iconColorClass: string;
  /** Test ID prefix for the page */
  testId: string;
  /** Introduction text for what the page will offer */
  intro: string;
  /** List of features that will be available */
  features: string[];
  /** Alternative actions/pages to use in the meantime */
  alternatives: ReactNode;
}

/**
 * ComingSoonPage - A reusable "Coming Soon" stub page template.
 *
 * Replaces the duplicated stub pattern in UserAccounts, TiersManagement, and AppsManagement.
 *
 * @example
 * ```tsx
 * <ComingSoonPage
 *   title="User Accounts"
 *   description="View and manage user accounts"
 *   icon={Users}
 *   iconBgClass="bg-emerald-500/10"
 *   iconColorClass="text-emerald-400"
 *   testId="user-accounts"
 *   intro="The User Accounts page will allow you to:"
 *   features={[
 *     'Search and browse all registered user accounts',
 *     'View user profile details and activity history',
 *     'Manage user roles and permissions',
 *   ]}
 *   alternatives={<>In the meantime, use <strong>Feedback</strong> to view user submissions...</>}
 * />
 * ```
 */
export function ComingSoonPage({
  title,
  description,
  icon,
  iconBgClass,
  iconColorClass,
  testId,
  intro,
  features,
  alternatives,
}: ComingSoonPageProps) {
  return (
    <AdminLayout maxWidth="narrow">
      <div className={LAYOUT.sectionSpacing}>
        <PageHeader
          title={title}
          description={description}
          icon={icon}
          iconBgClass={iconBgClass}
          iconColorClass={iconColorClass}
          testId={`${testId}-header`}
        />

        <Card className={LAYOUT.card.base}>
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
            <p className="text-slate-300">{intro}</p>
            <ul className="list-disc list-inside space-y-2 text-slate-400">
              {features.map((feature, index) => (
                <li key={index}>{feature}</li>
              ))}
            </ul>

            <div className="mt-6 p-4 bg-slate-900/50 rounded-lg border border-white/10">
              <div className="flex items-center gap-2 text-sm text-slate-400">
                <Settings className="h-4 w-4" />
                <span>{alternatives}</span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </AdminLayout>
  );
}
