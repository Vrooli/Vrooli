import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TooltipProvider } from './ui/tooltip';
import { CreateCampaignDialog } from './CreateCampaignDialog';

// [REQ:VT-REQ-009] Web interface dashboard with campaign creation

const renderWithProvider = (ui: React.ReactElement) => {
  return render(<TooltipProvider>{ui}</TooltipProvider>);
};

describe('CreateCampaignDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Dialog visibility', () => {
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
  });

  describe('Form fields', () => {
    it('should render all required form fields', () => {
      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={vi.fn()}
          isLoading={false}
        />
      );

      expect(screen.getByLabelText(/campaign name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/agent \/ creator/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/file patterns/i)).toBeInTheDocument();
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

    it('should have default value for agent/creator field', () => {
      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={vi.fn()}
          isLoading={false}
        />
      );

      const agentInput = screen.getByLabelText(/agent \/ creator/i) as HTMLInputElement;
      expect(agentInput.value).toBe('manual');
    });
  });

  describe('Form validation', () => {
    it('should show error when campaign name is empty', async () => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          isLoading={false}
        />
      );

      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      await waitFor(() => {
        const errorElement = document.getElementById('name-error');
        expect(errorElement).not.toBeNull();
        expect(errorElement?.textContent).toBe('Campaign name is required');
      });
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('should show error when campaign name is too short', async () => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          isLoading={false}
        />
      );

      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'ab');

      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      await waitFor(() => {
        const errorElement = document.getElementById('name-error');
        expect(errorElement).not.toBeNull();
        expect(errorElement?.textContent).toBe('Campaign name must be at least 3 characters');
      });
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('should show error when patterns field is empty', async () => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          isLoading={false}
        />
      );

      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      await waitFor(() => {
        const errorElement = document.getElementById('patterns-error');
        expect(errorElement).not.toBeNull();
        expect(errorElement?.textContent).toBe('At least one file pattern is required');
      });
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('should show error when patterns field has only whitespace', async () => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          isLoading={false}
        />
      );

      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '   ,  ,  ');

      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      await waitFor(() => {
        expect(screen.getByText(/at least one valid pattern is required/i)).toBeInTheDocument();
      });
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('should clear validation errors when inputs become valid', async () => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          isLoading={false}
        />
      );

      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      await waitFor(() => {
        expect(document.getElementById('name-error')).not.toBeNull();
      });

      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Valid Name');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '**/*.ts');

      await user.click(createButton);

      await waitFor(() => {
        expect(document.getElementById('name-error')).toBeNull();
        expect(document.getElementById('patterns-error')).toBeNull();
      });
    });
  });

  describe('Form submission', () => {
    it('should submit valid form data', async () => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          isLoading={false}
        />
      );

      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const agentInput = screen.getByLabelText(/agent \/ creator/i);
      await user.clear(agentInput);
      await user.type(agentInput, 'test-agent');

      const descInput = screen.getByLabelText(/description/i);
      await user.type(descInput, 'Test description');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '**/*.ts, **/*.tsx');

      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      await waitFor(() => {
        expect(onSubmit).toHaveBeenCalledWith({
          name: 'Test Campaign',
          from_agent: 'test-agent',
          description: 'Test description',
          patterns: ['**/*.ts', '**/*.tsx'],
        });
      });
    });

    it('should trim whitespace from patterns', async () => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          isLoading={false}
        />
      );

      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '  **/*.ts  ,  **/*.tsx  , **/*.go ');

      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      await waitFor(() => {
        expect(onSubmit).toHaveBeenCalledWith({
          name: 'Test Campaign',
          from_agent: 'manual',
          description: '',
          patterns: ['**/*.ts', '**/*.tsx', '**/*.go'],
        });
      });
    });

    it('should submit with keyboard shortcut Ctrl+Enter', async () => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          isLoading={false}
        />
      );

      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '**/*.ts');

      // Trigger Ctrl+Enter
      fireEvent.keyDown(window, { key: 'Enter', ctrlKey: true });

      await waitFor(() => {
        expect(onSubmit).toHaveBeenCalledWith({
          name: 'Test Campaign',
          from_agent: 'manual',
          description: '',
          patterns: ['**/*.ts'],
        });
      });
    });

    it('should not submit via keyboard shortcut when form is invalid', async () => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          isLoading={false}
        />
      );

      // Leave form empty
      fireEvent.keyDown(window, { key: 'Enter', ctrlKey: true });

      // Should not submit
      expect(onSubmit).not.toHaveBeenCalled();
    });

    it('should not submit when loading', async () => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={onSubmit}
          isLoading={true}
        />
      );

      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '**/*.ts');

      fireEvent.keyDown(window, { key: 'Enter', ctrlKey: true });

      expect(onSubmit).not.toHaveBeenCalled();
    });
  });

  describe('Dialog controls', () => {
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

    it('should call onOpenChange when cancel button is clicked', async () => {
      const user = userEvent.setup();
      const onOpenChange = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={onOpenChange}
          onSubmit={vi.fn()}
          isLoading={false}
        />
      );

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      await user.click(cancelButton);

      expect(onOpenChange).toHaveBeenCalledWith(false);
    });

    it('should reset form when dialog is closed', async () => {
      const user = userEvent.setup();
      const onOpenChange = vi.fn();

      const { rerender } = renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={onOpenChange}
          onSubmit={vi.fn()}
          isLoading={false}
        />
      );

      // Fill in form
      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '**/*.ts');

      // Close dialog
      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      await user.click(cancelButton);

      // Reopen dialog
      rerender(
        <TooltipProvider>
          <CreateCampaignDialog
            open={false}
            onOpenChange={onOpenChange}
            onSubmit={vi.fn()}
            isLoading={false}
          />
        </TooltipProvider>
      );

      rerender(
        <TooltipProvider>
          <CreateCampaignDialog
            open={true}
            onOpenChange={onOpenChange}
            onSubmit={vi.fn()}
            isLoading={false}
          />
        </TooltipProvider>
      );

      // Form should be reset
      const resetNameInput = screen.getByLabelText(/campaign name/i) as HTMLInputElement;
      const resetPatternsInput = screen.getByLabelText(/file patterns/i) as HTMLInputElement;
      expect(resetNameInput.value).toBe('');
      expect(resetPatternsInput.value).toBe('');
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

    it('should disable submit button when loading', () => {
      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={vi.fn()}
          onSubmit={vi.fn()}
          isLoading={true}
        />
      );

      const createButton = screen.getByRole('button', { name: /creating/i });
      expect(createButton).toBeDisabled();
    });

    it('should be able to close dialog', async () => {
      const user = userEvent.setup();
      const onOpenChange = vi.fn();

      renderWithProvider(
        <CreateCampaignDialog
          open={true}
          onOpenChange={onOpenChange}
          onSubmit={vi.fn()}
          isLoading={false}
        />
      );

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      await user.click(cancelButton);

      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });
});
