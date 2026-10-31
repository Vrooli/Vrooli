import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from './test-utils';
import App from './App';

const proxyInfo = vi.fn();
vi.mock('@vrooli/api-base', () => ({ getProxyInfo: () => proxyInfo() }));

// The providers make network calls on mount; passthrough keeps App routing pure.
vi.mock('./app/providers/AdminAuthProvider', () => ({
  AdminAuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));
vi.mock('./app/providers/LandingVariantProvider', () => ({
  LandingVariantProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));
vi.mock('./surfaces/admin-portal/components/ProtectedRoute', () => ({
  ProtectedRoute: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock('./surfaces/admin-portal/routes/AdminLogin', () => ({ AdminLogin: () => <div>AdminLogin</div> }));
vi.mock('./surfaces/admin-portal/routes/AdminHome', () => ({ AdminHome: () => <div>AdminHome</div> }));
vi.mock('./surfaces/admin-portal/routes/AdminAnalytics', () => ({ AdminAnalytics: () => <div>AdminAnalytics</div> }));
vi.mock('./surfaces/admin-portal/routes/Customization', () => ({ Customization: () => <div>Customization</div> }));
vi.mock('./surfaces/admin-portal/routes/VariantEditor', () => ({ VariantEditor: () => <div>VariantEditor</div> }));
vi.mock('./surfaces/admin-portal/routes/SectionEditor', () => ({ SectionEditor: () => <div>SectionEditor</div> }));
vi.mock('./surfaces/admin-portal/routes/AgentCustomization', () => ({ AgentCustomization: () => <div>AgentCustomization</div> }));
vi.mock('./surfaces/admin-portal/routes/BillingSettings', () => ({ BillingSettings: () => <div>BillingSettings</div> }));
vi.mock('./surfaces/admin-portal/routes/DownloadSettings', () => ({ DownloadSettings: () => <div>DownloadSettings</div> }));
vi.mock('./surfaces/admin-portal/routes/BrandingSettings', () => ({ BrandingSettings: () => <div>BrandingSettings</div> }));
vi.mock('./surfaces/admin-portal/routes/DocsViewer', () => ({ DocsViewer: () => <div>DocsViewer</div> }));
vi.mock('./surfaces/public-landing/routes/PublicLanding', () => ({ PublicLanding: () => <div>PublicLanding</div> }));

const renderAt = (path: string) => {
  window.history.replaceState({}, '', path);
  return renderWithProviders(<App />, { withoutRouter: true });
};

beforeEach(() => {
  vi.clearAllMocks();
  proxyInfo.mockReturnValue(undefined);
});

describe('App routing', () => {
  it('renders the public landing at the root path', () => {
    renderAt('/');
    expect(screen.getByText('PublicLanding')).toBeInTheDocument();
  });

  it('renders the admin login route', () => {
    renderAt('/admin/login');
    expect(screen.getByText('AdminLogin')).toBeInTheDocument();
  });

  it('renders a protected admin route', () => {
    renderAt('/admin/downloads');
    expect(screen.getByText('DownloadSettings')).toBeInTheDocument();
  });

  it('renders the variant editor route with a slug param', () => {
    renderAt('/admin/customization/variants/hero');
    expect(screen.getByText('VariantEditor')).toBeInTheDocument();
  });

  it('redirects unknown routes back to the public landing', () => {
    renderAt('/totally/unknown');
    expect(screen.getByText('PublicLanding')).toBeInTheDocument();
  });

  it('honors a reverse-proxy basename from the proxy info', () => {
    proxyInfo.mockReturnValue({ primary: { path: '/app/' } });
    renderAt('/app/admin/login');
    expect(screen.getByText('AdminLogin')).toBeInTheDocument();
  });
});
