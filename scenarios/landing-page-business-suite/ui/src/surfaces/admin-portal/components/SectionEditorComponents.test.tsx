import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import {
  PreviewPanel,
  StylingGuardrailsCard,
  VariantContextCard,
  VariantSectionTimeline,
} from './SectionEditorComponents';

const sections = [
  { key: 'hero-1', id: 1, section_type: 'hero', order: 1, enabled: true, content: {} },
  { key: 'faq-2', id: 2, section_type: 'faq', order: 2, enabled: false, content: {} },
] as never;

describe('section editor supporting components', () => {
  it('renders loading, error, empty, and actionable timeline states without allowing invalid reorders', () => {
    const onAddSection = vi.fn();
    const onNavigateSection = vi.fn();
    const onReorderSection = vi.fn();
    const props = { variantName: 'Control', currentSectionKey: null, currentSectionType: 'hero' as const, onAddSection, onNavigateSection, onReorderSection, reorderingSectionId: null, reorderError: null };
    const view = render(<VariantSectionTimeline {...props} sections={[]} loading error="Timeline unavailable" />);
    expect(screen.getByText('Loading sections...')).toBeInTheDocument();
    expect(screen.getByText('Timeline unavailable')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'New Section' }));
    expect(onAddSection).toHaveBeenCalledOnce();

    view.rerender(<VariantSectionTimeline {...props} sections={[]} loading={false} error={null} />);
    expect(screen.getByText(/no sections yet/i)).toBeInTheDocument();
    view.rerender(<VariantSectionTimeline {...props} sections={sections} loading={false} error={null} reorderError="Could not reorder" />);
    const hero = screen.getByRole('button', { name: /hero/i });
    fireEvent.click(hero);
    fireEvent.keyDown(hero, { key: 'Enter' });
    fireEvent.keyDown(hero, { key: ' ' });
    expect(onNavigateSection).toHaveBeenCalledTimes(3);
    const moveUpButtons = screen.getAllByRole('button', { name: 'Move up' });
    const moveDownButtons = screen.getAllByRole('button', { name: 'Move down' });
    expect(moveUpButtons[0]).toBeDisabled();
    expect(moveDownButtons[0]).not.toBeDisabled();
    fireEvent.click(moveDownButtons[0]!);
    expect(onReorderSection).toHaveBeenCalledWith(expect.objectContaining({ key: 'hero-1' }), 'down');
    expect(screen.getByText('Could not reorder')).toBeInTheDocument();
    expect(screen.getByText('Disabled')).toBeInTheDocument();
  });

  it('renders context guidance only when it exists and leaves the editor uncluttered otherwise', () => {
    const view = render(<VariantContextCard context={null} error={null} loading={false} />);
    expect(screen.queryByTestId('variant-context-card')).toBeNull();
    view.rerender(<VariantContextCard context={null} error="Guidance unavailable" loading />);
    expect(screen.getByText('Loading variant guidance...')).toBeInTheDocument();
    expect(screen.getByText('Guidance unavailable')).toBeInTheDocument();
    view.rerender(<VariantContextCard
      loading={false}
      error={null}
      context={{
        variant: { name: 'Founder' },
        axes: [{ axisId: 'tone', axisLabel: 'Tone', axisNote: 'Stay concise', selectionLabel: '', selectionDescription: 'Practical copy', agentHints: ['Lead with outcomes'] }],
        variantSpace: { agentGuidelines: ['Avoid jargon'] },
      } as never}
    />);
    expect(screen.getByText('Founder')).toBeInTheDocument();
    expect(screen.getByText('Not selected')).toBeInTheDocument();
    expect(screen.getByText('Stay concise')).toBeInTheDocument();
    expect(screen.getByText('Lead with outcomes')).toBeInTheDocument();
    expect(screen.getByText('Avoid jargon')).toBeInTheDocument();
  });

  it('uses a stable section-type fallback when a legacy section has no key', () => {
    const onNavigateSection = vi.fn();
    render(<VariantSectionTimeline
      variantName="Legacy"
      sections={[{ id: 9, section_type: 'footer', order: 9, content: {} }] as never}
      loading={false}
      error={null}
      currentSectionKey="footer-legacy"
      currentSectionType="footer"
      onNavigateSection={onNavigateSection}
      onAddSection={vi.fn()}
      onReorderSection={vi.fn()}
      reorderingSectionId={null}
      reorderError={null}
    />);
    const legacySection = screen.getByRole('button', { name: /footer/i });
    fireEvent.click(legacySection);
    expect(onNavigateSection).toHaveBeenCalledWith(expect.objectContaining({ section_type: 'footer' }));
    expect(screen.getByText('Enabled')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Move up' })).toBeNull();
  });

  it('renders styling guidance and gives preview renderers the current section contract', () => {
    const renderer = vi.fn(() => <div>Rendered preview</div>);
    const view = render(<StylingGuardrailsCard variantSlug="unknown-variant" />);
    expect(screen.getByTestId('styling-guardrails-card')).toBeInTheDocument();
    view.rerender(<PreviewPanel title="Hero preview" variantLabel="Control" renderer={renderer} content={{ headline: 'Ship' }} sectionType="hero" config={null} sectionEnabled={false} missingSectionMessage="Preview unavailable" />);
    expect(screen.getByText('Section is currently disabled')).toBeInTheDocument();
    expect(screen.getByText('Rendered preview')).toBeInTheDocument();
    expect(renderer).toHaveBeenCalledWith({ content: { headline: 'Ship' }, sectionType: 'hero', config: null });
    view.rerender(<PreviewPanel title="Fallback" variantLabel="Control" content={{}} sectionType="hero" config={null} sectionEnabled missingSectionMessage="Preview unavailable" />);
    expect(screen.getByText('Preview unavailable')).toBeInTheDocument();
  });
});
