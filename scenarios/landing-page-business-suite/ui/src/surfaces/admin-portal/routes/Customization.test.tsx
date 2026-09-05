import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Customization } from './Customization';
import * as customizationHook from '../hooks/useCustomizationPage';

vi.mock('../hooks/useCustomizationPage');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title, actions }: { title: string; actions?: React.ReactNode }) => <><h1>{title}</h1>{actions}</> }));
vi.mock('../components/RuntimeSignalStrip', () => ({ RuntimeSignalStrip: () => <div>Runtime signal</div> }));

function pageState(overrides: Record<string, unknown> = {}) {
  return { activeVariants: [], archivedVariants: [], filteredActiveVariants: [], savingWeights: {}, totalAssignedWeight: 0, weightStatus: 'empty', staleVariants: [], neverUpdatedVariants: [], underperformingInfo: null, attentionCandidateSlugs: new Set(), variantAttentionReasons: new Map(), statsBySlug: new Map(), variantQuery: '', attentionOnly: false, loading: false, error: null, analyticsLoading: false, analyticsError: null, operationAlert: null, clearOperationAlert: vi.fn(), variantListRef: { current: null }, snapshotDays: 7, fetchVariants: vi.fn(), handleArchive: vi.fn(), handleDelete: vi.fn(), persistWeight: vi.fn(), setWeightDraft: vi.fn(), setVariantQuery: vi.fn(), setAttentionOnly: vi.fn(), clearVariantFilters: vi.fn(), getWeight: vi.fn((variant: { weight?: number }) => variant.weight ?? 0), normalizeShare: vi.fn((weight: number) => weight), highlightVariantInList: vi.fn(), navigateToVariantEditor: vi.fn(), navigateToSectionEditor: vi.fn(), navigateToAgentCustomization: vi.fn(), navigateToNewVariant: vi.fn(), navigateToAnalytics: vi.fn(), openVariantPreview: vi.fn(), ...overrides } as unknown as ReturnType<typeof customizationHook.useCustomizationPage>;
}

describe('Customization', () => {
  beforeEach(() => { vi.clearAllMocks(); vi.stubGlobal('confirm', vi.fn(() => true)); });

  it('shows loading/error recovery and a no-variant creation path', () => {
    vi.mocked(customizationHook.useCustomizationPage).mockReturnValue(pageState({ loading: true }));
    const { rerender } = render(<Customization />);
    expect(screen.getByText('Loading variants...')).toBeInTheDocument();
    const failed = pageState({ error: 'Variant service unavailable' });
    vi.mocked(customizationHook.useCustomizationPage).mockReturnValue(failed);
    rerender(<Customization />);
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(failed.fetchVariants).toHaveBeenCalledOnce();
    const empty = pageState();
    vi.mocked(customizationHook.useCustomizationPage).mockReturnValue(empty);
    rerender(<Customization />);
    fireEvent.click(screen.getByRole('button', { name: 'Create Your First Variant' }));
    expect(empty.navigateToNewVariant).toHaveBeenCalledOnce();
  });

  it('filters and manages active/archived variants while persisting traffic allocation', async () => {
    const active = { id: 1, slug: 'enterprise', name: 'Enterprise', description: 'For enterprise buyers', weight: 60, updated_at: '2026-01-01T00:00:00Z', axes: { persona: 'enterprise' } };
    const archived = { id: 2, slug: 'legacy', name: 'Legacy', archived_at: '2026-01-01T00:00:00Z', axes: {} };
    const state = pageState({ activeVariants: [active], filteredActiveVariants: [active], archivedVariants: [archived], totalAssignedWeight: 60, weightStatus: 'under', variantQuery: 'enter', attentionOnly: true, attentionCandidateSlugs: new Set(['enterprise']), variantAttentionReasons: new Map([['enterprise', ['Stale copy']]]), statsBySlug: new Map([['enterprise', { variant_slug: 'enterprise', views: 100, conversions: 4, conversion_rate: 4, trend: 'up' }]]), staleVariants: [{ variant: active, daysSinceUpdate: 12 }] });
    vi.mocked(customizationHook.useCustomizationPage).mockReturnValue(state);
    render(<Customization />);
    expect(screen.getByTestId('variant-list-summary')).toHaveTextContent('Traffic assigned60%');
    fireEvent.change(screen.getByTestId('variant-search-input'), { target: { value: 'enterprise' } });
    fireEvent.click(screen.getByTestId('variant-attention-filter'));
    fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }));
    fireEvent.change(screen.getByTestId('live-weight-slider-enterprise'), { target: { value: '70' } });
    fireEvent.mouseUp(screen.getByTestId('live-weight-slider-enterprise'), { target: { value: '70' } });
    fireEvent.click(screen.getByTestId('edit-variant-enterprise'));
    fireEvent.click(screen.getByRole('button', { name: 'Preview Enterprise' }));
    fireEvent.click(screen.getByRole('button', { name: 'Archive Enterprise' }));
    fireEvent.click(screen.getByTestId('variant-analytics-enterprise'));
    fireEvent.click(screen.getByTestId('delete-variant-legacy'));
    await waitFor(() => { expect(state.handleArchive).toHaveBeenCalledWith('enterprise'); });
    expect(state.setVariantQuery).toHaveBeenCalledWith('enterprise');
    expect(state.setAttentionOnly).toHaveBeenCalledWith(false);
    expect(state.clearVariantFilters).toHaveBeenCalledOnce();
    expect(state.setWeightDraft).toHaveBeenCalledWith('enterprise', 70);
    expect(state.persistWeight).toHaveBeenCalledWith('enterprise', 70);
    expect(state.navigateToVariantEditor).toHaveBeenCalledWith('enterprise');
    expect(state.openVariantPreview).toHaveBeenCalledWith('enterprise');
    expect(state.navigateToAnalytics).toHaveBeenCalledWith('enterprise');
    expect(state.handleDelete).toHaveBeenCalledWith('legacy');
  });

  it('connects stale and underperforming experiment recommendations to editing and analytics', () => {
    const variant = { id: 1, slug: 'control', name: 'Control', weight: 100, updated_at: '2026-01-01T00:00:00Z' };
    const state = pageState({ activeVariants: [variant], filteredActiveVariants: [variant], totalAssignedWeight: 100, weightStatus: 'balanced', staleVariants: [{ variant, daysSinceUpdate: 11 }], neverUpdatedVariants: [variant], underperformingInfo: { variant, stats: { variant_slug: 'control', conversion_rate: 1.2 } } });
    vi.mocked(customizationHook.useCustomizationPage).mockReturnValue(state);
    render(<Customization />);
    fireEvent.click(screen.getByRole('button', { name: 'Refresh copy' }));
    fireEvent.click(screen.getByRole('button', { name: 'Inspect analytics' }));
    fireEvent.click(screen.getByRole('button', { name: 'Tune copy' }));
    fireEvent.click(screen.getByTestId('needs-attention-focus'));
    expect(state.navigateToSectionEditor).toHaveBeenCalledWith('control', { sectionType: 'hero' });
    expect(state.navigateToAnalytics).toHaveBeenCalledWith('control');
    expect(state.highlightVariantInList).toHaveBeenCalledWith('control');
  });
});
