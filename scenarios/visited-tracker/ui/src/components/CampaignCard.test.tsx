import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CampaignCard } from './CampaignCard';

const mockCampaign = {
  id: 'test-id',
  name: 'Test Campaign',
  from_agent: 'test-agent',
  status: 'active' as const,
  patterns: ['**/*.tsx'],
  total_files: 50,
  visited_files: 25,
  coverage_percent: 50,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

describe('CampaignCard', () => {
  it('should render campaign name', () => {
    render(
      <CampaignCard
        campaign={mockCampaign}
        onView={vi.fn()}
        onDelete={vi.fn()}
      />
    );

    expect(screen.getByText('Test Campaign')).toBeInTheDocument();
  });

  it('should display progress information', () => {
    render(
      <CampaignCard
        campaign={mockCampaign}
        onView={vi.fn()}
        onDelete={vi.fn()}
      />
    );

    // Check that file counts are displayed
    expect(screen.getByText('Files')).toBeInTheDocument();
    expect(screen.getByText('Visited')).toBeInTheDocument();
    expect(screen.getByText('Cover')).toBeInTheDocument(); // Abbreviated from iteration 5
  });

  it('should show patterns', () => {
    const campaignWithPatterns = {
      ...mockCampaign,
      patterns: ['**/*.tsx', '**/*.ts', 'api/**/*.go']
    };

    render(
      <CampaignCard
        campaign={campaignWithPatterns}
        onView={vi.fn()}
        onDelete={vi.fn()}
      />
    );

    // Check patterns appear in the card
    expect(screen.getByText('**/*.tsx')).toBeInTheDocument();
    expect(screen.getByText('**/*.ts')).toBeInTheDocument();
  });

  it('should render view and delete buttons', () => {
    render(
      <CampaignCard
        campaign={mockCampaign}
        onView={vi.fn()}
        onDelete={vi.fn()}
      />
    );

    expect(screen.getByRole('button', { name: /view/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /delete/i })).toBeInTheDocument();
  });

  it('should display status badge', () => {
    render(
      <CampaignCard
        campaign={mockCampaign}
        onView={vi.fn()}
        onDelete={vi.fn()}
      />
    );

    expect(screen.getByText('active')).toBeInTheDocument();
  });

  it('should show agent/creator name', () => {
    render(
      <CampaignCard
        campaign={mockCampaign}
        onView={vi.fn()}
        onDelete={vi.fn()}
      />
    );

    expect(screen.getByText(/test-agent/i)).toBeInTheDocument();
  });
});


describe('campaign navigation ownership', () => {
  it('opens the focused card but lets nested buttons own their keyboard activation', () => {
    const onView = vi.fn();
    const onDelete = vi.fn();
    render(<CampaignCard campaign={mockCampaign} onView={onView} onDelete={onDelete} />);
    const card = screen.getByRole('article');
    fireEvent.keyDown(card, { key: 'Enter' });
    fireEvent.keyDown(card, { key: ' ' });
    expect(onView).toHaveBeenCalledTimes(2);
    onView.mockClear();
    const remove = screen.getByRole('button', { name: /delete/i });
    fireEvent.keyDown(remove, { key: 'Enter' });
    fireEvent.click(remove);
    expect(onView).not.toHaveBeenCalled();
    expect(onDelete).toHaveBeenCalledTimes(1);
    expect(onDelete).toHaveBeenCalledWith('test-id', 'Test Campaign');
    fireEvent.click(screen.getByRole('button', { name: /view/i }));
    expect(onView).toHaveBeenCalledTimes(1);
    expect(onView).toHaveBeenCalledWith('test-id');
  });
});
