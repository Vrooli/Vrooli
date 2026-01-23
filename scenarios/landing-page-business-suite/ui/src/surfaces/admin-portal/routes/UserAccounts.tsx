import { Users, Settings, Wrench } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';

/**
 * User Accounts - Stub page for managing user accounts
 *
 * This is a placeholder page that will be implemented to allow admins to:
 * - View and search user accounts
 * - Manage user permissions and roles
 * - Handle account-related administrative tasks
 */
export function UserAccounts() {
  return (
    <AdminLayout>
      <div className="max-w-4xl mx-auto space-y-6">
        <PageHeader
          variant="icon-title"
          title="User Accounts"
          description="View and manage user accounts"
          icon={Users}
          iconBgClass="bg-emerald-500/10"
          iconColorClass="text-emerald-400"
          testId="user-accounts-header"
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
              The User Accounts page will allow you to:
            </p>
            <ul className="list-disc list-inside space-y-2 text-slate-400">
              <li>Search and browse all registered user accounts</li>
              <li>View user profile details and activity history</li>
              <li>Manage user roles and permissions</li>
              <li>Handle account suspensions and reactivations</li>
              <li>Export user data for reporting and analysis</li>
            </ul>

            <div className="mt-6 p-4 bg-slate-900/50 rounded-lg border border-white/10">
              <div className="flex items-center gap-2 text-sm text-slate-400">
                <Settings className="h-4 w-4" />
                <span>
                  In the meantime, use <strong>Feedback</strong> to view user submissions
                  and <strong>Waitlist</strong> to manage pending signups.
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </AdminLayout>
  );
}
