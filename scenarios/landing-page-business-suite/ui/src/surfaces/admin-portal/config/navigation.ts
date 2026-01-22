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
} from 'lucide-react';
import type { NavigationConfig } from './navigation.types';

/**
 * Admin portal navigation configuration
 *
 * Reorganized structure:
 * - Direct Links: Home, Profile, Docs
 * - Apps Dropdown: Apps, App Limits, Downloads, Usage
 * - Landing Page Dropdown: Branding, Customization, Analytics
 * - Billing Dropdown: Tiers, Tier Limits, API Keys, Billing
 * - Users Dropdown: Accounts, Feedback, Waitlist
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
      id: 'profile',
      name: 'Profile',
      icon: ShieldCheck,
      path: '/admin/profile',
      testId: 'nav-profile',
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
      id: 'apps',
      label: 'Apps',
      icon: AppWindow,
      items: [
        {
          id: 'apps-management',
          name: 'Apps',
          description: 'Manage applications and configurations',
          icon: AppWindow,
          path: '/admin/apps',
          testId: 'nav-apps',
          isStub: true,
        },
        {
          id: 'app-limits',
          name: 'App Limits',
          description: 'Configure per-app credit limits',
          icon: Gauge,
          path: '/admin/app-limits',
          testId: 'nav-app-limits',
        },
        {
          id: 'downloads',
          name: 'Downloads',
          description: 'Manage downloadable resources',
          icon: Download,
          path: '/admin/downloads',
          testId: 'nav-downloads',
        },
        {
          id: 'usage',
          name: 'Usage',
          description: 'Monitor credit usage across applications',
          icon: Activity,
          path: '/admin/usage',
          testId: 'nav-usage',
        },
      ],
    },
    {
      id: 'landing-page',
      label: 'Landing Page',
      icon: Palette,
      items: [
        {
          id: 'branding',
          name: 'Branding',
          description: 'Configure logo, colors, and brand identity',
          icon: Settings2,
          path: '/admin/branding',
          testId: 'nav-branding',
        },
        {
          id: 'customization',
          name: 'Customization',
          description: 'Manage landing page variants and sections',
          icon: Palette,
          path: '/admin/customization',
          testId: 'nav-customization',
        },
        {
          id: 'analytics',
          name: 'Analytics',
          description: 'View landing page performance metrics',
          icon: BarChart3,
          path: '/admin/analytics',
          testId: 'nav-analytics',
        },
      ],
    },
    {
      id: 'billing',
      label: 'Billing',
      icon: CreditCard,
      items: [
        {
          id: 'tiers',
          name: 'Tiers',
          description: 'Configure subscription tiers and plans',
          icon: Layers,
          path: '/admin/tiers',
          testId: 'nav-tiers',
          isStub: true,
        },
        {
          id: 'tier-limits',
          name: 'Tier Limits',
          description: 'Set credit limits per subscription tier',
          icon: Gauge,
          path: '/admin/tier-limits',
          testId: 'nav-tier-limits',
        },
        {
          id: 'api-keys',
          name: 'API Keys',
          description: 'Manage API keys and access tokens',
          icon: Key,
          path: '/admin/api-keys',
          testId: 'nav-api-keys',
        },
        {
          id: 'billing-settings',
          name: 'Billing',
          description: 'Configure Stripe and payment settings',
          icon: CreditCard,
          path: '/admin/billing',
          testId: 'nav-billing',
        },
      ],
    },
    {
      id: 'users',
      label: 'Users',
      icon: Users,
      items: [
        {
          id: 'accounts',
          name: 'Accounts',
          description: 'View and manage user accounts',
          icon: Users,
          path: '/admin/accounts',
          testId: 'nav-accounts',
          isStub: true,
        },
        {
          id: 'feedback',
          name: 'Feedback',
          description: 'Review user feedback and suggestions',
          icon: MessageSquare,
          path: '/admin/feedback',
          testId: 'nav-feedback',
        },
        {
          id: 'waitlist',
          name: 'Waitlist',
          description: 'Manage waitlist signups',
          icon: ClipboardList,
          path: '/admin/waitlist',
          testId: 'nav-waitlist',
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
  '/admin/profile': 'Profile',
  '/admin/docs': 'Documentation',
  '/admin/apps': 'Apps',
  '/admin/app-limits': 'App Limits',
  '/admin/downloads': 'Downloads',
  '/admin/usage': 'Usage',
  '/admin/branding': 'Branding',
  '/admin/customization': 'Customization',
  '/admin/analytics': 'Analytics',
  '/admin/tiers': 'Tiers',
  '/admin/tier-limits': 'Tier Limits',
  '/admin/api-keys': 'API Keys',
  '/admin/billing': 'Billing',
  '/admin/accounts': 'Accounts',
  '/admin/feedback': 'Feedback',
  '/admin/waitlist': 'Waitlist',
};
