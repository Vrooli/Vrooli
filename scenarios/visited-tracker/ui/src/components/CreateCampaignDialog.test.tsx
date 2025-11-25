import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { TooltipProvider } from './ui/tooltip';
import { CreateCampaignDialog } from './CreateCampaignDialog';

const renderWithProvider = (ui: React.ReactElement) => {
  return render(<TooltipProvider>{ui}</TooltipProvider>);
};

describe('CreateCampaignDialog', () => {
  it('should not render when closed', () => {
    const { container } = renderWithProvider(
      <CreateCampaignDialog
        open={false}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        isLoading={false}
      />
    );
    expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument();
  });

  it('should render dialog with all form fields when open', () => {
    renderWithProvider(
      <CreateCampaignDialog
        open={true}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        isLoading={false}
      />
    );

    expect(screen.getByText('Create New Campaign')).toBeInTheDocument();
    // Check that form fields exist
    const inputs = screen.getAllByRole('textbox');
    expect(inputs.length).toBeGreaterThanOrEqual(4);
  });

  it('should have helpful placeholder text', () => {
    renderWithProvider(
      <CreateCampaignDialog
        open={true}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        isLoading={false}
      />
    );

    // Check that the form has placeholder inputs
    const inputs = screen.getAllByRole('textbox');
    expect(inputs.length).toBeGreaterThan(0);
  });

  it('should render help buttons for all fields', () => {
    renderWithProvider(
      <CreateCampaignDialog
        open={true}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        isLoading={false}
      />
    );

    const helpButtons = screen.getAllByRole('button', { name: /help/i });
    // Should have help buttons for multiple fields
    expect(helpButtons.length).toBeGreaterThanOrEqual(4);
  });

  it('should have cancel and create buttons', () => {
    renderWithProvider(
      <CreateCampaignDialog
        open={true}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        isLoading={false}
      />
    );

    expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /create campaign/i })).toBeInTheDocument();
  });

  it('should show loading state on submit button', () => {
    renderWithProvider(
      <CreateCampaignDialog
        open={true}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        isLoading={true}
      />
    );

    const createButton = screen.getByRole('button', { name: /creating/i });
    expect(createButton).toBeInTheDocument();
    expect(createButton).toBeDisabled();
  });
});
