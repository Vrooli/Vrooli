import { describe, expect, it } from 'vitest';
import { NAVIGATION_CONFIG } from './navigation';
import {
  buildBreadcrumbs,
  findActiveGroup,
  findNavItemByPath,
  getAllNavItems,
  getRouteLabel,
  isGroupActive,
  isStubRoute,
} from './navigation.utils';

describe('admin navigation utilities', () => {
  it('finds the matching item and group for nested paths, while safely handling unknown paths', () => {
    expect(findNavItemByPath('/admin/customization/variants/control')?.id).toBe('customization');
    expect(findNavItemByPath('/admin/unknown')).toBeNull();
    expect(findActiveGroup('/admin/billing/credentials')?.id).toBe('billing');
    expect(findActiveGroup('/admin/unknown')).toBeNull();

    const billing = NAVIGATION_CONFIG.groups.find((group) => group.id === 'billing');
    if (!billing) throw new Error('billing group is required for navigation tests');
    expect(isGroupActive(billing, '/admin/billing')).toBe(true);
    expect(isGroupActive(billing, '/admin/customization')).toBe(false);
  });

  it('returns configured route labels and safely leaves unknown routes unlabeled', () => {
    expect(getRouteLabel('/admin/feedback')).toBe('Feedback');
    expect(getRouteLabel('/admin/not-a-route')).toBeUndefined();
  });

  it('builds breadcrumbs for static, analytics, variant, section, and unmatched routes', () => {
    expect(buildBreadcrumbs('/admin/feedback')).toEqual([
      { label: 'Admin', path: '/admin' },
      { label: 'Feedback', path: '/admin/feedback' },
    ]);
    expect(buildBreadcrumbs('/admin/analytics/control')).toEqual([
      { label: 'Admin', path: '/admin' },
      { label: 'Analytics', path: '/admin/analytics' },
      { label: 'Variant control' },
    ]);
    expect(buildBreadcrumbs('/admin/analytics')).toEqual([
      { label: 'Admin', path: '/admin' },
      { label: 'Analytics', path: '/admin/analytics' },
    ]);
    expect(buildBreadcrumbs('/admin/customization/variants/new')).toEqual([
      { label: 'Admin', path: '/admin' },
      { label: 'Customization', path: '/admin/customization' },
      { label: 'New Variant' },
    ]);
    expect(buildBreadcrumbs('/admin/customization')).toEqual([
      { label: 'Admin', path: '/admin' },
      { label: 'Customization', path: '/admin/customization' },
    ]);
    expect(buildBreadcrumbs('/admin/customization/variants/control')).toEqual([
      { label: 'Admin', path: '/admin' },
      { label: 'Customization', path: '/admin/customization' },
      { label: 'Variant control', path: '/admin/customization/variants/control' },
    ]);
    expect(buildBreadcrumbs('/admin/customization/variants/control/sections/42')).toEqual([
      { label: 'Admin', path: '/admin' },
      { label: 'Customization', path: '/admin/customization' },
      { label: 'Variant control', path: '/admin/customization/variants/control' },
      { label: 'Section 42' },
    ]);
    expect(buildBreadcrumbs('/admin/customization/variants/control/sections/new')).toEqual([
      { label: 'Admin', path: '/admin' },
      { label: 'Customization', path: '/admin/customization' },
      { label: 'Variant control', path: '/admin/customization/variants/control' },
      { label: 'New Section' },
    ]);
    expect(buildBreadcrumbs('/elsewhere')).toEqual([{ label: 'Admin', path: '/admin' }]);
  });

  it('flattens every grouped navigation item and identifies stub routes only', () => {
    const expectedCount = NAVIGATION_CONFIG.groups.reduce((count, group) => count + group.items.length, 0);
    expect(getAllNavItems()).toHaveLength(expectedCount);
    expect(isStubRoute('/admin/accounts')).toBe(true);
    expect(isStubRoute('/admin/customization')).toBe(false);
    expect(isStubRoute('/admin/unknown')).toBe(false);
  });
});
