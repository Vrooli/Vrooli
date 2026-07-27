import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { PlanPreview } from './PlanPreview';

vi.mock('../../../public-landing/sections/PricingSection', () => ({
  PricingSection: () => <button type="button">Preview CTA</button>,
}));

const data = (monthlyCount: number, placeholderCount: number) => ({ overview: {} as never, monthlyCount, placeholderCount });

describe('PlanPreview', () => {
  it('explains saved and placeholder plan composition with correct singular grammar', () => {
    const { rerender } = render(<PlanPreview data={data(0, 2)} />);
    expect(screen.getByText(/No saved monthly plans yet/)).toBeInTheDocument();
    rerender(<PlanPreview data={data(1, 1)} />);
    expect(screen.getByText('Showing 1 saved plan plus 1 demo placeholder to fill the preview.')).toBeInTheDocument();
    rerender(<PlanPreview data={data(2, 0)} />);
    expect(screen.getByText('Showing 2 saved monthly plans.')).toBeInTheDocument();
  });

  it('prevents preview interactions from triggering visitor-facing controls', () => {
    const onClick = vi.fn();
    render(<div onClick={onClick}><PlanPreview data={data(1, 0)} /></div>);
    fireEvent.click(screen.getByRole('button', { name: 'Preview CTA' }));
    expect(onClick).not.toHaveBeenCalled();
    fireEvent.keyDown(screen.getByText('Preview CTA').parentElement!.parentElement!, { key: 'Enter' });
  });
});
