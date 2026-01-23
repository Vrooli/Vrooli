import type { DownloadApp, LandingSection } from '../../../shared/api';
import { getSectionAnchorId } from '../../../shared/lib/sections';

/**
 * Navigation item for public landing page
 */
export interface NavItem {
  id: string;
  label: string;
}

/**
 * Section types that appear in navigation, in display order
 */
const SECTION_NAV_ORDER = ['hero', 'video', 'features', 'pricing', 'testimonials', 'faq', 'cta', 'footer'] as const;

/**
 * Human-readable labels for section types in navigation
 */
const SECTION_NAV_LABELS: Record<string, string> = {
  hero: 'Overview',
  video: 'Demo',
  features: 'Features',
  pricing: 'Pricing',
  testimonials: 'Proof',
  faq: 'FAQ',
  cta: 'Call to Action',
  footer: 'More',
};

/**
 * Human-readable labels for download platforms
 */
const DOWNLOAD_PLATFORM_LABELS: Record<string, string> = {
  windows: 'Windows',
  mac: 'macOS',
  linux: 'Linux',
};

/**
 * Checks if a download app has any download targets (platforms or storefronts)
 */
export function hasDownloadTargets(app: DownloadApp): boolean {
  return (app.platforms?.length ?? 0) > 0 || (app.storefronts?.length ?? 0) > 0;
}

/**
 * Formats a platform string to a human-readable label
 * @param platform - Platform identifier (e.g., 'windows', 'mac', 'linux')
 * @returns Human-readable platform label
 */
export function formatDownloadPlatform(platform?: string): string {
  if (!platform) {
    return 'Download';
  }
  return DOWNLOAD_PLATFORM_LABELS[platform.toLowerCase()] ?? platform;
}

/**
 * Gets the label for a section type for navigation display
 * @param sectionType - The section type
 * @returns Human-readable label for the section
 */
export function getSectionNavLabel(sectionType: string): string {
  return SECTION_NAV_LABELS[sectionType] ?? sectionType;
}

/**
 * Builds navigation items from landing sections
 * @param sections - Landing page sections
 * @param includeDownloads - Whether to include a downloads link
 * @param downloadAnchorId - The anchor ID for the downloads section
 * @returns Array of navigation items
 */
export function buildNavItems(
  sections: LandingSection[],
  includeDownloads: boolean,
  downloadAnchorId: string
): NavItem[] {
  const items: NavItem[] = [];

  for (const type of SECTION_NAV_ORDER) {
    const match = sections.find((section) => section.section_type === type);
    if (!match) {
      continue;
    }
    items.push({
      id: getSectionAnchorId(match),
      label: SECTION_NAV_LABELS[type] ?? type
    });
  }

  if (includeDownloads) {
    items.push({ id: downloadAnchorId, label: 'Downloads' });
  }

  return items;
}

/**
 * Gets the download button label based on available apps
 * @param downloadApps - Array of download apps
 * @returns Appropriate label for the download button
 */
export function getDownloadButtonLabel(downloadApps: DownloadApp[]): string {
  if (downloadApps.length === 0) {
    return 'Downloads';
  }

  if (downloadApps.length === 1) {
    const single = downloadApps[0];
    const singleInstaller = single.platforms?.[0];

    if ((single.platforms?.length ?? 0) === 1 && singleInstaller) {
      return `Download ${formatDownloadPlatform(singleInstaller.platform)}`;
    }
    if ((single.storefronts?.length ?? 0) === 1 && (single.platforms?.length ?? 0) === 0) {
      return `Open ${single.storefronts?.[0]?.label ?? 'store'}`;
    }
    return `View ${single.name}`;
  }

  return 'View downloads';
}
