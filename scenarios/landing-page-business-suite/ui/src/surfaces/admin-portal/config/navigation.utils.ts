import type { BreadcrumbSegment, NavItem, NavGroup, NavigationConfig } from './navigation.types';
import { NAVIGATION_CONFIG, ROUTE_LABELS } from './navigation';

/**
 * Find a navigation item by its path across all groups and direct links
 */
export function findNavItemByPath(path: string, config: NavigationConfig = NAVIGATION_CONFIG): NavItem | null {
  // Check all groups for matching item
  for (const group of config.groups) {
    const found = group.items.find((item) => path.startsWith(item.path));
    if (found) {
      return found;
    }
  }
  return null;
}

/**
 * Find a navigation group that contains items matching the current path
 */
export function findActiveGroup(path: string, config: NavigationConfig = NAVIGATION_CONFIG): NavGroup | null {
  for (const group of config.groups) {
    const hasActiveItem = group.items.some((item) => path.startsWith(item.path));
    if (hasActiveItem) {
      return group;
    }
  }
  return null;
}

/**
 * Check if any item in a group is currently active based on the path
 */
export function isGroupActive(group: NavGroup, currentPath: string): boolean {
  return group.items.some((item) => currentPath.startsWith(item.path));
}

/**
 * Get the label for a route path from the configuration
 */
export function getRouteLabel(path: string): string | undefined {
  return ROUTE_LABELS[path];
}

/**
 * Build breadcrumb segments from the current path
 * Handles both simple routes and dynamic nested routes
 */
export function buildBreadcrumbs(pathname: string): BreadcrumbSegment[] {
  const segments: BreadcrumbSegment[] = [{ label: 'Admin', path: '/admin' }];

  // Handle simple static routes first
  const routeLabel = getRouteLabel(pathname);
  if (routeLabel && pathname !== '/admin') {
    segments.push({ label: routeLabel, path: pathname });
    return segments;
  }

  // Handle analytics routes with variant slugs
  if (pathname.startsWith('/admin/analytics')) {
    segments.push({ label: 'Analytics', path: '/admin/analytics' });
    const variantMatch = pathname.match(/\/admin\/analytics\/(.+)/);
    const variantSlug = variantMatch?.[1];
    if (variantSlug) {
      segments.push({ label: `Variant ${variantSlug}` });
    }
    return segments;
  }

  // Handle customization routes with variants and sections
  if (pathname.startsWith('/admin/customization')) {
    segments.push({ label: 'Customization', path: '/admin/customization' });

    if (pathname.includes('/variants/new')) {
      segments.push({ label: 'New Variant' });
    } else {
      const variantMatch = pathname.match(/\/variants\/([^/]+)/);
      const variantSlug = variantMatch?.[1];
      if (variantSlug) {
        segments.push({
          label: `Variant ${variantSlug}`,
          path: `/admin/customization/variants/${variantSlug}`,
        });

        const sectionMatch = pathname.match(/\/sections\/(\d+|new)/);
        const sectionId = sectionMatch?.[1];
        if (sectionId) {
          segments.push({
            label: sectionId === 'new' ? 'New Section' : `Section ${sectionId}`,
          });
        }
      }
    }
    return segments;
  }

  return segments;
}

/**
 * Get all nav items as a flat list (for search functionality)
 */
export function getAllNavItems(config: NavigationConfig = NAVIGATION_CONFIG): NavItem[] {
  const items: NavItem[] = [];

  for (const group of config.groups) {
    items.push(...group.items);
  }

  return items;
}

/**
 * Check if a path corresponds to a stub page
 */
export function isStubRoute(path: string, config: NavigationConfig = NAVIGATION_CONFIG): boolean {
  const item = findNavItemByPath(path, config);
  return item?.isStub ?? false;
}
