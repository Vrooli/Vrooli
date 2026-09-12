import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { renderWithProviders } from '../../../test-utils';
import { Customization } from './Customization';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async (importActual) => {
  const actual = await importActual<typeof import('react-router-dom')>();
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock('../components/AdminLayout', () => ({
  AdminLayout: ({ children }: { children: React.ReactNode }) => <div data-testid="admin-layout">{children}</div>,
}));

const { mockListVariants, mockArchiveVariant, mockDeleteVariant } = vi.hoisted(() => ({
  mockListVariants: vi.fn(),
  mockArchiveVariant: vi.fn(),
  mockDeleteVariant: vi.fn(),
}));

vi.mock('../../../shared/api', () => ({
  listVariants: mockListVariants,
  archiveVariant: mockArchiveVariant,
  deleteVariant: mockDeleteVariant,
}));

const { mockBuildDateRange, mockFetchAnalyticsSummary } = vi.hoisted(() => ({
  mockBuildDateRange: vi.fn(),
  mockFetchAnalyticsSummary: vi.fn(),
}));

vi.mock('../controllers/analyticsController', () => ({
  buildDateRange: mockBuildDateRange,
  fetchAnalyticsSummary: mockFetchAnalyticsSummary,
}));

vi.mock('../controllers/variantEditorController', () => ({
  loadVariantEditorData: vi.fn(),
}));

const DAY = 24 * 60 * 60 * 1000;

const variant = (over: Record<string, unknown>) => ({
  id: 1n,
  name: 'Variant',
  slug: 'variant',
  status: 'active',
  weight: 50,
  description: '',
  axes: {},
  ...over,
});

const activeVariants = [
  variant({ id: 1n, name: 'Hero', slug: 'hero', weight: 60, description: 'Primary', axes: { tone: 'bold' }, updatedAt: timestampFromDate(new Date()) }),
  variant({ id: 2n, name: 'Stale One', slug: 'stale', weight: 30, updatedAt: timestampFromDate(new Date(Date.now() - 20 * DAY)) }),
  variant({ id: 3n, name: 'Never Touched', slug: 'nevermore', weight: 20, updatedAt: undefined }),
];
const archived = [
  variant({ id: 9n, name: 'Old Variant', slug: 'oldie', status: 'archived', archivedAt: timestampFromDate(new Date()) }),
];

const analytics = {
  variantStats: [
    { variantSlug: 'hero', conversionRate: 5, views: 1000, conversions: 50, trend: 1 },
    { variantSlug: 'stale', conversionRate: 1, views: 500, conversions: 5, trend: -1 },
    { variantSlug: 'nevermore', conversionRate: 3, views: 200, conversions: 6, trend: 0 },
  ],
};

beforeEach(() => {
  vi.clearAllMocks();
  mockListVariants.mockResolvedValue([...activeVariants, ...archived]);
  mockArchiveVariant.mockResolvedValue(undefined);
  mockDeleteVariant.mockResolvedValue(undefined);
  mockBuildDateRange.mockReturnValue({ startDate: '2025-01-01', endDate: '2025-01-08' });
  mockFetchAnalyticsSummary.mockResolvedValue(analytics);
  vi.stubGlobal('confirm', vi.fn(() => true));
  vi.stubGlobal('alert', vi.fn());
  vi.spyOn(window, 'open').mockImplementation(() => null);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('Customization [REQ:VARIANT-MGMT]', () => {
  it('renders active and archived variant cards after loading', async () => {
    renderWithProviders(<Customization />);
    expect(await screen.findByRole('heading', { name: 'Customization' })).toBeInTheDocument();
    expect(screen.getByTestId('variant-card-hero')).toBeInTheDocument();
    expect(screen.getByTestId('variant-card-stale')).toBeInTheDocument();
    expect(screen.getByText('Archived Variants')).toBeInTheDocument();
    expect(screen.getByTestId('variant-card-archived-oldie')).toBeInTheDocument();
  });

  it('summarizes over-allocated traffic weight (110%)', async () => {
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-list-summary');
    const summary = screen.getByTestId('variant-list-summary');
    expect(within(summary).getByText('110%')).toBeInTheDocument();
    // Ops panel explains the over-allocation.
    expect(screen.getByText(/exceeding 100% by 10%/)).toBeInTheDocument();
  });

  it('flags stale, never-customized, and underperforming variants for attention', async () => {
    renderWithProviders(<Customization />);
    const opsPanel = await screen.findByTestId('experience-ops-panel');
    // Stale variant surfaces in freshness column.
    expect(within(opsPanel).getByText(/Updated 20 days ago/)).toBeInTheDocument();
    // Never-customized callout.
    expect(screen.getByText(/Never customized: Never Touched/)).toBeInTheDocument();
    // Underperforming = lowest conversion (stale at 1%).
    expect(within(opsPanel).getByText('Lowest conversion')).toBeInTheDocument();
    expect(within(opsPanel).getByText('1.00%')).toBeInTheDocument();
  });

  it('filters the active grid by search query', async () => {
    const user = userEvent.setup();
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-card-hero');
    await user.type(screen.getByTestId('variant-search-input'), 'hero');
    expect(screen.getByTestId('variant-card-hero')).toBeInTheDocument();
    expect(screen.queryByTestId('variant-card-stale')).not.toBeInTheDocument();
    expect(screen.getByText('Showing 1 of 3')).toBeInTheDocument();
  });

  it('shows a reset affordance when filters exclude everything', async () => {
    const user = userEvent.setup();
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-card-hero');
    await user.type(screen.getByTestId('variant-search-input'), 'zzz-nothing');
    const reset = await screen.findByTestId('clear-variant-filters');
    await user.click(reset);
    expect(screen.getByTestId('variant-card-hero')).toBeInTheDocument();
  });

  it('restricts the grid to attention variants when the filter is on', async () => {
    const user = userEvent.setup();
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-card-hero');
    await user.click(screen.getByTestId('variant-attention-filter'));
    // hero is healthy (fresh + best conversion) so it drops out.
    expect(screen.queryByTestId('variant-card-hero')).not.toBeInTheDocument();
    expect(screen.getByTestId('variant-card-stale')).toBeInTheDocument();
  });

  it('archives an active variant after confirmation and refetches', async () => {
    const user = userEvent.setup();
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-card-hero');
    await user.click(screen.getByTestId('archive-variant-hero'));
    await waitFor(() => expect(mockArchiveVariant).toHaveBeenCalledWith('hero'));
    expect(mockListVariants).toHaveBeenCalledTimes(2);
  });

  it('does not archive when the confirmation is declined', async () => {
    (globalThis.confirm as ReturnType<typeof vi.fn>).mockReturnValue(false);
    const user = userEvent.setup();
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-card-hero');
    await user.click(screen.getByTestId('archive-variant-hero'));
    expect(mockArchiveVariant).not.toHaveBeenCalled();
  });

  it('permanently deletes an archived variant', async () => {
    const user = userEvent.setup();
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-card-archived-oldie');
    await user.click(screen.getByTestId('delete-variant-oldie'));
    await waitFor(() => expect(mockDeleteVariant).toHaveBeenCalledWith('oldie'));
  });

  it('navigates to the new-variant and agent routes from the header', async () => {
    const user = userEvent.setup();
    renderWithProviders(<Customization />);
    await screen.findByTestId('create-variant');
    await user.click(screen.getByTestId('create-variant'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization/variants/new');
    await user.click(screen.getByTestId('trigger-agent-customization'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization/agent');
  });

  it('shows the empty state when there are no active variants', async () => {
    mockListVariants.mockResolvedValue([]);
    renderWithProviders(<Customization />);
    expect(await screen.findByText('No active variants yet')).toBeInTheDocument();
  });

  it('renders an error state with retry when variants fail to load', async () => {
    mockListVariants.mockRejectedValueOnce(new Error('list failed'));
    const user = userEvent.setup();
    renderWithProviders(<Customization />);
    expect(await screen.findByText(/Error: list failed/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Retry' }));
    await waitFor(() => expect(screen.getByTestId('variant-card-hero')).toBeInTheDocument());
  });

  it('runs the experiment-ops action handlers (refresh, inspect, tune, highlight)', async () => {
    const user = userEvent.setup();
    renderWithProviders(<Customization />);
    const opsPanel = await screen.findByTestId('experience-ops-panel');

    // Freshness column: "Refresh copy" for the stale variant.
    await user.click(within(opsPanel).getByRole('button', { name: /Refresh copy/i }));
    // Needs-attention column: inspect analytics, tune copy, highlight.
    await user.click(within(opsPanel).getByRole('button', { name: /Inspect analytics/i }));
    await user.click(within(opsPanel).getByRole('button', { name: /Tune copy/i }));
    await user.click(within(opsPanel).getByTestId('needs-attention-focus'));

    expect(mockNavigate).toHaveBeenCalled();
  });

  it('previews and edits a variant from its card actions', async () => {
    const user = userEvent.setup();
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-card-hero');
    await user.click(screen.getByTestId('preview-variant-hero'));
    expect(window.open).toHaveBeenCalledWith('/?variant=hero', '_blank');
    await user.click(screen.getByTestId('edit-variant-hero'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization/variants/hero');
    await user.click(screen.getByTestId('variant-analytics-hero'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/analytics?variant=hero');
  });

  it('warns when the analytics snapshot is unavailable', async () => {
    mockFetchAnalyticsSummary.mockRejectedValue(new Error('no metrics'));
    renderWithProviders(<Customization />);
    expect(await screen.findByText(/Metrics snapshot unavailable right now/)).toBeInTheDocument();
  });

  it('applies a focus deep-link to highlight and jump to a variant section', async () => {
    renderWithProviders(<Customization />, {
      routerEntries: ['/admin/customization?focus=stale&focusSectionType=hero'],
    });
    await screen.findByTestId('variant-card-stale');
    // The focus effect highlights the variant (attention filter + search) and
    // resolves its section editor via loadVariantEditorData.
    await waitFor(() => expect(mockNavigate).toHaveBeenCalled());
  });

  it('labels a variant updated yesterday distinctly', async () => {
    mockListVariants.mockResolvedValue([
      variant({ id: 1n, name: 'Hero', slug: 'hero', weight: 100, updatedAt: timestampFromDate(new Date(Date.now() - DAY - 1000)) }),
    ]);
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-card-hero');
    expect(screen.getByText(/Updated yesterday/)).toBeInTheDocument();
  });

  it('falls back to stable trend and blank archive date when data is missing', async () => {
    mockListVariants.mockResolvedValue([
      variant({ id: 1n, name: 'Hero', slug: 'hero', weight: 100, updatedAt: timestampFromDate(new Date()) }),
      variant({ id: 9n, name: 'Old', slug: 'oldie', status: 'archived', archivedAt: undefined }),
    ]);
    mockFetchAnalyticsSummary.mockResolvedValue({
      variantStats: [{ variantSlug: 'hero', conversionRate: 5, views: 100, conversions: 5, trend: undefined }],
    });
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-card-hero');
    // Performance summary renders a stable trend when none is reported.
    expect(within(screen.getByTestId('variant-performance-hero')).getByText('stable')).toBeInTheDocument();
    // Archived card without an archive date still renders.
    expect(screen.getByTestId('variant-card-archived-oldie')).toBeInTheDocument();
  });

  it('reports a balanced traffic split when weights total 100', async () => {
    mockListVariants.mockResolvedValue([
      variant({ id: 1n, name: 'Hero', slug: 'hero', weight: 60, updatedAt: timestampFromDate(new Date()) }),
      variant({ id: 2n, name: 'Beta', slug: 'beta', weight: 40, updatedAt: timestampFromDate(new Date()) }),
    ]);
    renderWithProviders(<Customization />);
    const summary = await screen.findByTestId('variant-list-summary');
    expect(within(summary).getByText('100%')).toBeInTheDocument();
    expect(screen.getByText(/Traffic is fully allocated across variants/)).toBeInTheDocument();
  });

  it('reports under-allocated traffic when weights total less than 100', async () => {
    mockListVariants.mockResolvedValue([
      variant({ id: 1n, name: 'Hero', slug: 'hero', weight: 30, updatedAt: timestampFromDate(new Date()) }),
    ]);
    renderWithProviders(<Customization />);
    await screen.findByTestId('variant-list-summary');
    expect(screen.getByText(/70% of visitors are idle/)).toBeInTheDocument();
  });

  it('renders healthy freshness copy when all variants are recently updated', async () => {
    mockListVariants.mockResolvedValue([
      variant({ id: 1n, name: 'Hero', slug: 'hero', weight: 100, updatedAt: timestampFromDate(new Date()) }),
    ]);
    mockFetchAnalyticsSummary.mockResolvedValue({ variantStats: [] });
    renderWithProviders(<Customization />);
    await screen.findByTestId('experience-ops-panel');
    expect(screen.getByText(/have been edited within the last/)).toBeInTheDocument();
    // With no analytics stats, the attention column shows the guidance copy.
    expect(screen.getByText(/Drive traffic to gather enough data/)).toBeInTheDocument();
  });
});
