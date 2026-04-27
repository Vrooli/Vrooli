import { ReactNode } from 'react';
import { Route } from 'react-router-dom';
import { ErrorBoundary } from '../../shared/ui/ErrorBoundary';
import { ProtectedRoute } from '../../surfaces/admin-portal/components/ProtectedRoute';
import { AdminHome } from '../../surfaces/admin-portal/routes/AdminHome';
import { AdminAnalytics } from '../../surfaces/admin-portal/routes/AdminAnalytics';
import { Customization } from '../../surfaces/admin-portal/routes/Customization';
import { VariantEditor } from '../../surfaces/admin-portal/routes/VariantEditor';
import { SectionEditor } from '../../surfaces/admin-portal/routes/SectionEditor';
import { AgentCustomization } from '../../surfaces/admin-portal/routes/AgentCustomization';
import { BillingSettings } from '../../surfaces/admin-portal/routes/BillingSettings';
import { DownloadSettings } from '../../surfaces/admin-portal/routes/DownloadSettings';
import { RemoteProfiles } from '../../surfaces/admin-portal/routes/RemoteProfiles';
import { BrandingSettings } from '../../surfaces/admin-portal/routes/BrandingSettings';
import { DocsViewer } from '../../surfaces/admin-portal/routes/DocsViewer';
import { FeedbackManagement } from '../../surfaces/admin-portal/routes/FeedbackManagement';
import { WaitlistManagement } from '../../surfaces/admin-portal/routes/WaitlistManagement';
import { ProfileSettings } from '../../surfaces/admin-portal/routes/ProfileSettings';
import { APIKeysSettings } from '../../surfaces/admin-portal/routes/APIKeysSettings';
import { TierLimitsSettings } from '../../surfaces/admin-portal/routes/TierLimitsSettings';
import { AppLimitsSettings } from '../../surfaces/admin-portal/routes/AppLimitsSettings';
import { UsageDashboard } from '../../surfaces/admin-portal/routes/UsageDashboard';
import { AppsManagement } from '../../surfaces/admin-portal/routes/AppsManagement';
import { TiersManagement } from '../../surfaces/admin-portal/routes/TiersManagement';
import { CouponsManagement } from '../../surfaces/admin-portal/routes/CouponsManagement';
import { UserAccounts } from '../../surfaces/admin-portal/routes/UserAccounts';
import { LandingDashboard } from '../../surfaces/admin-portal/routes/LandingDashboard';
import { BillingDashboard } from '../../surfaces/admin-portal/routes/BillingDashboard';
import { UsersDashboard } from '../../surfaces/admin-portal/routes/UsersDashboard';

function AdminRoute({ name, children }: { name: string; children: ReactNode }) {
  return (
    <ProtectedRoute>
      <ErrorBoundary level="route" name={name}>
        {children}
      </ErrorBoundary>
    </ProtectedRoute>
  );
}

export const adminRoutes = (
  <>
    <Route path="/admin" element={<AdminRoute name="AdminHome"><AdminHome /></AdminRoute>} />

    {/* Section dashboards */}
    <Route path="/admin/landing" element={<AdminRoute name="LandingDashboard"><LandingDashboard /></AdminRoute>} />
    <Route path="/admin/billing-home" element={<AdminRoute name="BillingDashboard"><BillingDashboard /></AdminRoute>} />
    <Route path="/admin/users" element={<AdminRoute name="UsersDashboard"><UsersDashboard /></AdminRoute>} />

    {/* Analytics */}
    <Route path="/admin/analytics" element={<AdminRoute name="AdminAnalytics"><AdminAnalytics /></AdminRoute>} />
    <Route path="/admin/analytics/:variantSlug" element={<AdminRoute name="AdminAnalyticsVariant"><AdminAnalytics /></AdminRoute>} />

    {/* Customization + variants */}
    <Route path="/admin/customization" element={<AdminRoute name="Customization"><Customization /></AdminRoute>} />
    <Route path="/admin/customization/agent" element={<AdminRoute name="AgentCustomization"><AgentCustomization /></AdminRoute>} />
    <Route path="/admin/customization/variants/:slug" element={<AdminRoute name="VariantEditor"><VariantEditor /></AdminRoute>} />
    <Route path="/admin/customization/variants/:variantSlug/sections/:sectionId" element={<AdminRoute name="SectionEditor"><SectionEditor /></AdminRoute>} />

    {/* Settings */}
    <Route path="/admin/billing" element={<AdminRoute name="BillingSettings"><BillingSettings /></AdminRoute>} />
    <Route path="/admin/downloads" element={<AdminRoute name="DownloadSettings"><DownloadSettings /></AdminRoute>} />
    <Route path="/admin/remote-profiles" element={<AdminRoute name="RemoteProfiles"><RemoteProfiles /></AdminRoute>} />
    <Route path="/admin/branding" element={<AdminRoute name="BrandingSettings"><BrandingSettings /></AdminRoute>} />
    <Route path="/admin/profile" element={<AdminRoute name="ProfileSettings"><ProfileSettings /></AdminRoute>} />
    <Route path="/admin/docs" element={<AdminRoute name="DocsViewer"><DocsViewer /></AdminRoute>} />
    <Route path="/admin/feedback" element={<AdminRoute name="FeedbackManagement"><FeedbackManagement /></AdminRoute>} />
    <Route path="/admin/waitlist" element={<AdminRoute name="WaitlistManagement"><WaitlistManagement /></AdminRoute>} />

    {/* Credit system */}
    <Route path="/admin/api-keys" element={<AdminRoute name="APIKeysSettings"><APIKeysSettings /></AdminRoute>} />
    <Route path="/admin/tier-limits" element={<AdminRoute name="TierLimitsSettings"><TierLimitsSettings /></AdminRoute>} />
    <Route path="/admin/app-limits" element={<AdminRoute name="AppLimitsSettings"><AppLimitsSettings /></AdminRoute>} />
    <Route path="/admin/usage" element={<AdminRoute name="UsageDashboard"><UsageDashboard /></AdminRoute>} />

    {/* Catalog management */}
    <Route path="/admin/apps" element={<AdminRoute name="AppsManagement"><AppsManagement /></AdminRoute>} />
    <Route path="/admin/tiers" element={<AdminRoute name="TiersManagement"><TiersManagement /></AdminRoute>} />
    <Route path="/admin/coupons" element={<AdminRoute name="CouponsManagement"><CouponsManagement /></AdminRoute>} />
    <Route path="/admin/accounts" element={<AdminRoute name="UserAccounts"><UserAccounts /></AdminRoute>} />
  </>
);
