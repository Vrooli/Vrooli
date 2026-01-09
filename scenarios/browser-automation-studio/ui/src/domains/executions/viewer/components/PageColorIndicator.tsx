/**
 * PageColorIndicator Component
 *
 * A small colored dot that indicates which page/tab a timeline entry belongs to.
 * Uses consistent colors from the ExecutionTabBar color palette.
 */

import { useMemo } from 'react';
import type { ExecutionPage } from '../../store';

/** Color palette for page indicators (must match ExecutionTabBar). */
const PAGE_COLORS = [
  { bg: 'bg-blue-500', text: 'text-blue-500', border: 'border-blue-500', hex: '#3b82f6' },
  { bg: 'bg-emerald-500', text: 'text-emerald-500', border: 'border-emerald-500', hex: '#10b981' },
  { bg: 'bg-purple-500', text: 'text-purple-500', border: 'border-purple-500', hex: '#8b5cf6' },
  { bg: 'bg-orange-500', text: 'text-orange-500', border: 'border-orange-500', hex: '#f97316' },
  { bg: 'bg-pink-500', text: 'text-pink-500', border: 'border-pink-500', hex: '#ec4899' },
  { bg: 'bg-cyan-500', text: 'text-cyan-500', border: 'border-cyan-500', hex: '#06b6d4' },
  { bg: 'bg-yellow-500', text: 'text-yellow-500', border: 'border-yellow-500', hex: '#eab308' },
  { bg: 'bg-rose-500', text: 'text-rose-500', border: 'border-rose-500', hex: '#f43f5e' },
];

/** Get consistent color for a page based on its index. */
export function getPageColor(pageIndex: number) {
  return PAGE_COLORS[pageIndex % PAGE_COLORS.length];
}

/** Get page color by ID from a list of pages. */
export function getPageColorById(pageId: string | undefined, pages: ExecutionPage[]) {
  if (!pageId) return null;
  const index = pages.findIndex((p) => p.id === pageId);
  return index >= 0 ? getPageColor(index) : null;
}

/** Get page info by ID. */
export function getPageById(pageId: string | undefined, pages: ExecutionPage[]) {
  if (!pageId) return null;
  return pages.find((p) => p.id === pageId) ?? null;
}

interface PageColorIndicatorProps {
  /** The page ID to show the color for. */
  pageId?: string;
  /** List of all pages for color lookup. */
  pages: ExecutionPage[];
  /** Size variant. */
  size?: 'sm' | 'md' | 'lg';
  /** Whether to show the page title on hover. */
  showTooltip?: boolean;
  /** Additional CSS classes. */
  className?: string;
}

export function PageColorIndicator({
  pageId,
  pages,
  size = 'sm',
  showTooltip = true,
  className = '',
}: PageColorIndicatorProps) {
  const pageInfo = useMemo(() => {
    if (!pageId || pages.length <= 1) return null;
    const index = pages.findIndex((p) => p.id === pageId);
    if (index < 0) return null;
    return {
      page: pages[index],
      color: getPageColor(index),
      index,
    };
  }, [pageId, pages]);

  // Don't render if single page or page not found
  if (!pageInfo) return null;

  const sizeClasses = {
    sm: 'w-2 h-2',
    md: 'w-2.5 h-2.5',
    lg: 'w-3 h-3',
  };

  return (
    <span
      className={`inline-block rounded-full flex-shrink-0 ${pageInfo.color.bg} ${sizeClasses[size]} ${className}`}
      title={showTooltip ? `Page ${pageInfo.index + 1}: ${pageInfo.page.title || pageInfo.page.url}` : undefined}
    />
  );
}

interface PageBadgeProps {
  /** The page ID to show the badge for. */
  pageId?: string;
  /** List of all pages for lookup. */
  pages: ExecutionPage[];
  /** Whether to show the badge in compact mode. */
  compact?: boolean;
  /** Additional CSS classes. */
  className?: string;
}

/** A more detailed page badge with color and abbreviated title. */
export function PageBadge({ pageId, pages, compact = false, className = '' }: PageBadgeProps) {
  const pageInfo = useMemo(() => {
    if (!pageId || pages.length <= 1) return null;
    const index = pages.findIndex((p) => p.id === pageId);
    if (index < 0) return null;
    return {
      page: pages[index],
      color: getPageColor(index),
      index,
    };
  }, [pageId, pages]);

  // Don't render if single page or page not found
  if (!pageInfo) return null;

  const { page, color, index } = pageInfo;

  // Get short label for the page
  const label = useMemo(() => {
    if (page.isInitial) return 'main';
    // Try to get a meaningful short label from the URL
    try {
      const url = new URL(page.url);
      const path = url.pathname.split('/').filter(Boolean).pop();
      if (path && path.length <= 10) return path;
    } catch {
      // ignore
    }
    return `tab ${index + 1}`;
  }, [page, index]);

  if (compact) {
    return (
      <span
        className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[9px] font-medium ${className}`}
        style={{ backgroundColor: `${color.hex}20`, color: color.hex }}
        title={`${page.title || page.url}`}
      >
        <span className={`w-1.5 h-1.5 rounded-full ${color.bg}`} />
        {label}
      </span>
    );
  }

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium ${className}`}
      style={{ backgroundColor: `${color.hex}15`, color: color.hex }}
      title={`${page.title || page.url}`}
    >
      <span className={`w-2 h-2 rounded-full ${color.bg}`} />
      {label}
    </span>
  );
}

export default PageColorIndicator;
