import { AppWindow } from 'lucide-react';
import { ComingSoonPage } from '../components/ComingSoonPage';

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
    <ComingSoonPage
      title="Apps Management"
      description="Manage applications and their configurations"
      icon={AppWindow}
      iconBgClass="bg-blue-500/10"
      iconColorClass="text-blue-400"
      testId="apps-management"
      intro="The Apps Management page will allow you to:"
      features={[
        'View all connected applications and their status',
        'Configure application-specific settings and permissions',
        'Monitor application health and performance metrics',
        'Manage API access and rate limits per application',
        'View usage statistics broken down by application',
      ]}
      alternatives="In the meantime, use <strong>App Limits</strong> to configure credit limits per application."
    />
  );
}
