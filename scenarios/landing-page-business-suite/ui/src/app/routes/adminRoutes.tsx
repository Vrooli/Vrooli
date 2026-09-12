/* eslint-disable react-refresh/only-export-components */
import React, { ReactNode, lazy } from 'react';
import { Route } from 'react-router-dom';
import { ErrorBoundary } from '../../shared/ui/ErrorBoundary';
import { ProtectedRoute } from '../../surfaces/admin-portal/components/ProtectedRoute';
import { onProfilerRender } from '../../lib/profiler';

const lazyRoute = <T extends Record<K, React.ComponentType>, K extends keyof T>(
  loader: () => Promise<T>,
  exportName: K,
) =>
  lazy(() =>
    loader().then((module) => ({
      default: module[exportName],
    }))
  );

const AdminHome = lazyRoute(() => import('../../surfaces/admin-portal/routes/AdminHome'), 'AdminHome');
const AdminAnalytics = lazyRoute(() => import('../../surfaces/admin-portal/routes/AdminAnalytics'), 'AdminAnalytics');
const Customization = lazyRoute(() => import('../../surfaces/admin-portal/routes/Customization'), 'Customization');
const VariantEditor = lazyRoute(() => import('../../surfaces/admin-portal/routes/VariantEditor'), 'VariantEditor');
const SectionEditor = lazyRoute(() => import('../../surfaces/admin-portal/routes/SectionEditor'), 'SectionEditor');
const AgentCustomization = lazyRoute(() => import('../../surfaces/admin-portal/routes/AgentCustomization'), 'AgentCustomization');
const BillingSettings = lazyRoute(() => import('../../surfaces/admin-portal/routes/BillingSettings'), 'BillingSettings');
const DownloadSettings = lazyRoute(() => import('../../surfaces/admin-portal/routes/DownloadSettings'), 'DownloadSettings');
const RemoteProfiles = lazyRoute(() => import('../../surfaces/admin-portal/routes/RemoteProfiles'), 'RemoteProfiles');
const BrandingSettings = lazyRoute(() => import('../../surfaces/admin-portal/routes/BrandingSettings'), 'BrandingSettings');
const DocsViewer = lazyRoute(() => import('../../surfaces/admin-portal/routes/DocsViewer'), 'DocsViewer');
const FeedbackManagement = lazyRoute(() => import('../../surfaces/admin-portal/routes/FeedbackManagement'), 'FeedbackManagement');
const WaitlistManagement = lazyRoute(() => import('../../surfaces/admin-portal/routes/WaitlistManagement'), 'WaitlistManagement');
const ProfileSettings = lazyRoute(() => import('../../surfaces/admin-portal/routes/ProfileSettings'), 'ProfileSettings');
const APIKeysSettings = lazyRoute(() => import('../../surfaces/admin-portal/routes/APIKeysSettings'), 'APIKeysSettings');
const TierLimitsSettings = lazyRoute(() => import('../../surfaces/admin-portal/routes/TierLimitsSettings'), 'TierLimitsSettings');
const AppLimitsSettings = lazyRoute(() => import('../../surfaces/admin-portal/routes/AppLimitsSettings'), 'AppLimitsSettings');
const UsageDashboard = lazyRoute(() => import('../../surfaces/admin-portal/routes/UsageDashboard'), 'UsageDashboard');
const AppsManagement = lazyRoute(() => import('../../surfaces/admin-portal/routes/AppsManagement'), 'AppsManagement');
const TiersManagement = lazyRoute(() => import('../../surfaces/admin-portal/routes/TiersManagement'), 'TiersManagement');
const CouponsManagement = lazyRoute(() => import('../../surfaces/admin-portal/routes/CouponsManagement'), 'CouponsManagement');
const UserAccounts = lazyRoute(() => import('../../surfaces/admin-portal/routes/UserAccounts'), 'UserAccounts');
const LandingDashboard = lazyRoute(() => import('../../surfaces/admin-portal/routes/LandingDashboard'), 'LandingDashboard');
const BillingDashboard = lazyRoute(() => import('../../surfaces/admin-portal/routes/BillingDashboard'), 'BillingDashboard');
const UsersDashboard = lazyRoute(() => import('../../surfaces/admin-portal/routes/UsersDashboard'), 'UsersDashboard');

function AdminRoute({ name, children }: { name: string; children: ReactNode }) {
  return (
    <ProtectedRoute>
      <ErrorBoundary level="route" name={name}>
        <React.Profiler id={name} onRender={onProfilerRender}>
          {children}
        </React.Profiler>
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
