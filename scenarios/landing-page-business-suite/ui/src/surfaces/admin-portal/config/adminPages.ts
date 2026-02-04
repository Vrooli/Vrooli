import { matchPath } from 'react-router-dom';

// DOC: docs/guides/ADMIN_GUIDE.md#admin-navigation-map
// DOC: docs/guides/ADMIN_GUIDE.md#docs-viewer
// Canonical admin page catalog for documentation coverage and header doc links.

export interface AdminPageDocumentation {
  path: string;
  anchor?: string;
}

export interface AdminPageDefinition {
  id: string;
  name: string;
  description: string;
  route: string;
  routePattern: string;
  documentation?: AdminPageDocumentation;
}

const ADMIN_GUIDE_DOC = { path: 'guides/ADMIN_GUIDE.md' } satisfies AdminPageDocumentation;

export const ADMIN_PAGE_DEFINITIONS: AdminPageDefinition[] = [
  {
    id: 'admin-login',
    name: 'Admin Login',
    description: 'Authenticate to access the admin portal.',
    route: '/admin/login',
    routePattern: '/admin/login',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'accessing-the-admin-portal' },
  },
  {
    id: 'admin-home',
    name: 'Landing Manager Admin',
    description: 'Admin overview with quick flows, stats, and reset controls.',
    route: '/admin',
    routePattern: '/admin',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'admin-home' },
  },
  {
    id: 'landing-dashboard',
    name: 'Landing Dashboard',
    description: 'Landing page health, quick flows, and resume actions.',
    route: '/admin/landing',
    routePattern: '/admin/landing',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'landing-dashboard' },
  },
  {
    id: 'analytics-variant',
    name: 'Analytics Dashboard',
    description: 'Conversion metrics and variant performance for a specific variant.',
    route: '/admin/analytics/:variantSlug',
    routePattern: '/admin/analytics/:variantSlug',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'analytics' },
  },
  {
    id: 'analytics-dashboard',
    name: 'Analytics Dashboard',
    description: 'View conversion metrics and performance data.',
    route: '/admin/analytics',
    routePattern: '/admin/analytics',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'analytics' },
  },
  {
    id: 'customization',
    name: 'Customization',
    description: 'Manage landing page variants and A/B testing.',
    route: '/admin/customization',
    routePattern: '/admin/customization',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'customization-variants-and-sections' },
  },
  {
    id: 'customization-agent',
    name: 'Agent Customization',
    description: 'Trigger AI-powered landing page improvements.',
    route: '/admin/customization/agent',
    routePattern: '/admin/customization/agent',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'agent-customization' },
  },
  {
    id: 'section-editor',
    name: 'Section Editor',
    description: 'Edit a single landing page section with live preview.',
    route: '/admin/customization/variants/:variantSlug/sections/:sectionId',
    routePattern: '/admin/customization/variants/:variantSlug/sections/:sectionId',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'section-editor' },
  },
  {
    id: 'variant-editor',
    name: 'Variant Editor',
    description: 'Edit variant metadata, sections, and JSON payload.',
    route: '/admin/customization/variants/:slug',
    routePattern: '/admin/customization/variants/:slug',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'variant-editor' },
  },
  {
    id: 'branding',
    name: 'Branding',
    description: 'Site identity, SEO defaults, and coming soon settings.',
    route: '/admin/branding',
    routePattern: '/admin/branding',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'branding' },
  },
  {
    id: 'billing-dashboard',
    name: 'Billing Dashboard',
    description: 'Stripe readiness, plans status, and quick flows.',
    route: '/admin/billing-home',
    routePattern: '/admin/billing-home',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'billing-dashboard' },
  },
  {
    id: 'billing-settings',
    name: 'Stripe',
    description: 'Configure Stripe API keys and webhook settings.',
    route: '/admin/billing',
    routePattern: '/admin/billing',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'stripe-settings' },
  },
  {
    id: 'tiers',
    name: 'Plans',
    description: 'Subscription tier management (coming soon).',
    route: '/admin/tiers',
    routePattern: '/admin/tiers',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'plans-coming-soon' },
  },
  {
    id: 'api-keys',
    name: 'AI Keys',
    description: 'Manage API keys for AI providers.',
    route: '/admin/api-keys',
    routePattern: '/admin/api-keys',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'ai-keys' },
  },
  {
    id: 'apps-dashboard',
    name: 'Apps Dashboard',
    description: 'App distribution health and quick flows.',
    route: '/admin/apps',
    routePattern: '/admin/apps',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'apps-dashboard' },
  },
  {
    id: 'downloads',
    name: 'Downloads',
    description: 'App registry, installers, storage, and artifacts.',
    route: '/admin/downloads',
    routePattern: '/admin/downloads',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'downloads' },
  },
  {
    id: 'remote-profiles',
    name: 'Remote Profiles',
    description: 'Manage remote admin sessions for deployed suites.',
    route: '/admin/remote-profiles',
    routePattern: '/admin/remote-profiles',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'remote-profiles' },
  },
  {
    id: 'usage',
    name: 'Usage',
    description: 'Monthly AI credit usage reporting.',
    route: '/admin/usage',
    routePattern: '/admin/usage',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'usage' },
  },
  {
    id: 'tier-limits',
    name: 'Tier Limits',
    description: 'Credit limits per subscription tier.',
    route: '/admin/tier-limits',
    routePattern: '/admin/tier-limits',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'tier-limits' },
  },
  {
    id: 'app-limits',
    name: 'App Limits',
    description: 'Per-app usage limits.',
    route: '/admin/app-limits',
    routePattern: '/admin/app-limits',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'app-limits' },
  },
  {
    id: 'users-dashboard',
    name: 'Users Dashboard',
    description: 'User activity, feedback, and waitlist stats.',
    route: '/admin/users',
    routePattern: '/admin/users',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'users-dashboard' },
  },
  {
    id: 'accounts',
    name: 'Accounts',
    description: 'User accounts, subscriptions, and sessions (coming soon).',
    route: '/admin/accounts',
    routePattern: '/admin/accounts',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'accounts-in-progress' },
  },
  {
    id: 'feedback',
    name: 'Feedback',
    description: 'Triage user feedback and requests.',
    route: '/admin/feedback',
    routePattern: '/admin/feedback',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'feedback' },
  },
  {
    id: 'waitlist',
    name: 'Waitlist',
    description: 'Coming soon signups and toggle.',
    route: '/admin/waitlist',
    routePattern: '/admin/waitlist',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'waitlist' },
  },
  {
    id: 'profile',
    name: 'Profile',
    description: 'Update admin email and password.',
    route: '/admin/profile',
    routePattern: '/admin/profile',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'profile' },
  },
  {
    id: 'docs',
    name: 'Documentation',
    description: 'Browse the scenario documentation.',
    route: '/admin/docs',
    routePattern: '/admin/docs',
    documentation: { ...ADMIN_GUIDE_DOC, anchor: 'docs-viewer' },
  },
];

export interface AdminPageDocLink {
  url: string;
  label: string;
  pageId: string;
}

export function getAdminPageByPath(pathname: string): AdminPageDefinition | null {
  return (
    ADMIN_PAGE_DEFINITIONS.find((page) => matchPath({ path: page.routePattern, end: true }, pathname)) ?? null
  );
}

export function buildAdminDocUrl(doc: AdminPageDocumentation): string {
  const params = new URLSearchParams({ doc: doc.path });
  const anchor = doc.anchor ? `#${doc.anchor}` : '';
  return `/admin/docs?${params.toString()}${anchor}`;
}

export function getAdminPageDocLink(pathname: string): AdminPageDocLink | null {
  const page = getAdminPageByPath(pathname);
  if (!page?.documentation) {
    return null;
  }

  return {
    url: buildAdminDocUrl(page.documentation),
    label: `Open documentation for ${page.name}`,
    pageId: page.id,
  };
}
