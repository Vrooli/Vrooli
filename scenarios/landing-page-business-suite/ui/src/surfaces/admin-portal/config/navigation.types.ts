import type { LucideIcon } from 'lucide-react';

/**
 * Navigation item definition for admin portal
 */
export interface NavItem {
  id: string;
  name: string;
  description: string;
  icon: LucideIcon;
  path: string;
  testId?: string;
  badge?: string | null;
  /** Marks items as placeholder stubs not yet implemented */
  isStub?: boolean;
}

/**
 * Navigation dropdown group containing multiple items
 */
export interface NavGroup {
  id: string;
  label: string;
  icon: LucideIcon;
  items: NavItem[];
  /** When true, renders this group on the right side of the nav bar */
  rightAligned?: boolean;
}

/**
 * Direct link in the header navigation (not in a dropdown)
 */
export interface DirectNavLink {
  id: string;
  name: string;
  icon: LucideIcon;
  path: string;
  testId?: string;
}

/**
 * Complete navigation configuration for the admin portal
 */
export interface NavigationConfig {
  /** Direct links shown in header (not in dropdowns) */
  directLinks: DirectNavLink[];
  /** Dropdown groups with nested items */
  groups: NavGroup[];
}

/**
 * Breadcrumb segment for navigation trail
 */
export interface BreadcrumbSegment {
  label: string;
  path?: string;
}
