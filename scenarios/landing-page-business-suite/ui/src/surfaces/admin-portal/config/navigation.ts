import {
  Home,
  ShieldCheck,
  Book,
  AppWindow,
  Gauge,
  Download,
  Activity,
  Palette,
  Settings2,
  BarChart3,
  Layers,
  CreditCard,
  Key,
  Users,
  MessageSquare,
  ClipboardList,
  LayoutDashboard,
  Bot,
} from 'lucide-react';
import type { NavigationConfig } from './navigation.types';

/**
 * Admin portal navigation configuration
 *
 * Navigation order follows user journey:
 * - Direct Links: Home, Docs
 * - Landing: Customize how to advertise the bundle
 * - Billing: Set up payments to make money
 * - Apps: Configure the apps being sold
 * - Users: Manage customers
 * - Account: Admin's own settings (right-aligned)
 */
export const NAVIGATION_CONFIG: NavigationConfig = {
  directLinks: [
    {
      id: 'home',
      name: 'Home',
      icon: Home,
      path: '/admin',
      testId: 'nav-home',
    },
    {
      id: 'docs',
      name: 'Docs',
      icon: Book,
      path: '/admin/docs',
      testId: 'nav-docs',
    },
  ],

  groups: [
    {
      id: 'landing',
      label: 'Landing',
      icon: Palette,
      items: [
        {
          id: 'landing-dashboard',
          name: 'Dashboard',
          description: 'Landing page flows, stats, and navigation',
          icon: LayoutDashboard,
          path: '/admin/landing',
          testId: 'nav-landing-dashboard',
        },
        {
          id: 'customization',
          name: 'Customization',
          description: 'Manage landing page variants and A/B testing',
          icon: Palette,
          path: '/admin/customization',
          testId: 'nav-customization',
        },
        {
          id: 'analytics',
          name: 'Analytics',
          description: 'View conversion metrics and performance data',
          icon: BarChart3,
          path: '/admin/analytics',
          testId: 'nav-analytics',
        },
        {
          id: 'branding',
          name: 'Branding',
          description: 'Site identity, colors, SEO, and support settings',
          icon: Settings2,
          path: '/admin/branding',
          testId: 'nav-branding',
        },
        {
          id: 'agent',
          name: 'Agent',
          description: 'AI-powered landing page improvements',
          icon: Bot,
          path: '/admin/customization/agent',
          testId: 'nav-agent',
        },
      ],
    },
    {
      id: 'billing',
      label: 'Billing',
      icon: CreditCard,
      items: [
        {
          id: 'billing-dashboard',
          name: 'Dashboard',
          description: 'Stripe status, flows, and payment navigation',
          icon: LayoutDashboard,
          path: '/admin/billing-home',
          testId: 'nav-billing-dashboard',
        },
        {
          id: 'billing-settings',
          name: 'Stripe',
          description: 'Configure Stripe API keys',
          icon: CreditCard,
          path: '/admin/billing',
          testId: 'nav-billing',
        },
        {
          id: 'tiers',
          name: 'Plans',
          description: 'Manage bundle pricing tiers',
          icon: Layers,
          path: '/admin/tiers',
          testId: 'nav-tiers',
          isStub: true,
        },
        {
          id: 'api-keys',
          name: 'AI Keys',
          description: 'Manage API keys for AI providers',
          icon: Key,
          path: '/admin/api-keys',
          testId: 'nav-api-keys',
        },
      ],
    },
    {
      id: 'apps',
      label: 'Apps',
      icon: AppWindow,
      items: [
        {
          id: 'apps-dashboard',
          name: 'Dashboard',
          description: 'App health, usage overview, and navigation',
          icon: LayoutDashboard,
          path: '/admin/apps',
          testId: 'nav-apps-dashboard',
        },
        {
          id: 'downloads',
          name: 'Downloads',
          description: 'App registry, platforms, and installers',
          icon: Download,
          path: '/admin/downloads',
          testId: 'nav-downloads',
        },
        {
          id: 'usage',
          name: 'Usage',
          description: 'Monitor credit consumption',
          icon: Activity,
          path: '/admin/usage',
          testId: 'nav-usage',
        },
        {
          id: 'tier-limits',
          name: 'Tier Limits',
          description: 'Credit limits per subscription tier',
          icon: Gauge,
          path: '/admin/tier-limits',
          testId: 'nav-tier-limits',
        },
        {
          id: 'app-limits',
          name: 'App Limits',
          description: 'Per-app usage limits',
          icon: Gauge,
          path: '/admin/app-limits',
          testId: 'nav-app-limits',
        },
      ],
    },
    {
      id: 'users',
      label: 'Users',
      icon: Users,
      items: [
        {
          id: 'users-dashboard',
          name: 'Dashboard',
          description: 'User stats, feedback overview, and navigation',
          icon: LayoutDashboard,
          path: '/admin/users',
          testId: 'nav-users-dashboard',
        },
        {
          id: 'accounts',
          name: 'Accounts',
          description: 'User accounts, subscriptions, and sessions',
          icon: Users,
          path: '/admin/accounts',
          testId: 'nav-accounts',
          isStub: true,
        },
        {
          id: 'feedback',
          name: 'Feedback',
          description: 'Triage user feedback',
          icon: MessageSquare,
          path: '/admin/feedback',
          testId: 'nav-feedback',
        },
        {
          id: 'waitlist',
          name: 'Waitlist',
          description: 'Coming soon signups',
          icon: ClipboardList,
          path: '/admin/waitlist',
          testId: 'nav-waitlist',
        },
      ],
    },
    {
      id: 'account',
      label: 'Account',
      icon: ShieldCheck,
      rightAligned: true,
      items: [
        {
          id: 'profile',
          name: 'Profile',
          description: 'Email, password, and credentials',
          icon: ShieldCheck,
          path: '/admin/profile',
          testId: 'nav-profile',
        },
      ],
    },
  ],
};

/**
 * Map of route paths to breadcrumb labels
 * Used for generating breadcrumb navigation
 */
export const ROUTE_LABELS: Record<string, string> = {
  '/admin': 'Admin',
  '/admin/landing': 'Landing Dashboard',
  '/admin/profile': 'Profile',
  '/admin/docs': 'Documentation',
  '/admin/apps': 'Apps Dashboard',
  '/admin/app-limits': 'App Limits',
  '/admin/downloads': 'Downloads',
  '/admin/usage': 'Usage',
  '/admin/branding': 'Branding',
  '/admin/customization': 'Customization',
  '/admin/customization/agent': 'Agent',
  '/admin/analytics': 'Analytics',
  '/admin/tiers': 'Plans',
  '/admin/tier-limits': 'Tier Limits',
  '/admin/api-keys': 'AI Keys',
  '/admin/billing': 'Stripe',
  '/admin/billing-home': 'Billing Dashboard',
  '/admin/accounts': 'Accounts',
  '/admin/feedback': 'Feedback',
  '/admin/waitlist': 'Waitlist',
  '/admin/users': 'Users Dashboard',
};
