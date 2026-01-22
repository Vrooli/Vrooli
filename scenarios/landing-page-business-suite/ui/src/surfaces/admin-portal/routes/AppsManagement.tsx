import { AppWindow, Settings, Wrench } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';

/**
 * Apps Management - Stub page for managing applications
 *
 * This is a placeholder page that will be implemented to allow admins to:
 * - View and manage connected applications
 * - Configure app-specific settings
 * - Monitor app health and status
 */
export function AppsManagement() {
  return (
    <AdminLayout>
      <div className="max-w-4xl mx-auto space-y-6">
        <div className="flex items-center gap-4 mb-8">
          <div className="p-3 bg-blue-500/10 rounded-xl">
            <AppWindow className="h-8 w-8 text-blue-400" />
          </div>
          <div>
            <h1 className="text-3xl font-semibold">Apps Management</h1>
            <p className="text-slate-400 mt-1">
              Manage applications and their configurations
            </p>
          </div>
        </div>

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
              The Apps Management page will allow you to:
            </p>
            <ul className="list-disc list-inside space-y-2 text-slate-400">
              <li>View all connected applications and their status</li>
              <li>Configure application-specific settings and permissions</li>
              <li>Monitor application health and performance metrics</li>
              <li>Manage API access and rate limits per application</li>
              <li>View usage statistics broken down by application</li>
            </ul>

            <div className="mt-6 p-4 bg-slate-900/50 rounded-lg border border-white/10">
              <div className="flex items-center gap-2 text-sm text-slate-400">
                <Settings className="h-4 w-4" />
                <span>
                  In the meantime, use <strong>App Limits</strong> to configure credit limits per application.
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </AdminLayout>
  );
}
