// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#ui-component-tests
// [REQ:REQ-P1-004a] Semantic HTML and ARIA - Tests for accessibility attributes
// [REQ:REQ-P1-004b] Keyboard Navigation - Tests for keyboard interaction
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ConfirmDialog } from './ConfirmDialog';

describe('ConfirmDialog', () => {
  const defaultProps = {
    isOpen: true,
    title: 'Delete Item',
    message: 'Are you sure you want to delete this item?',
    onConfirm: vi.fn(),
    onCancel: vi.fn(),
  };

  describe('rendering', () => {
    it('renders when open', () => {
      render(<ConfirmDialog {...defaultProps} />);

      expect(screen.getByTestId('confirm-dialog')).toBeInTheDocument();
      expect(screen.getByText('Delete Item')).toBeInTheDocument();
      expect(screen.getByText('Are you sure you want to delete this item?')).toBeInTheDocument();
    });

    it('does not render when closed', () => {
      render(<ConfirmDialog {...defaultProps} isOpen={false} />);

      expect(screen.queryByTestId('confirm-dialog')).not.toBeInTheDocument();
    });

    it('displays custom button labels', () => {
      render(
        <ConfirmDialog
          {...defaultProps}
          confirmLabel="Yes, Delete"
          cancelLabel="No, Keep It"
        />
      );

      expect(screen.getByText('Yes, Delete')).toBeInTheDocument();
      expect(screen.getByText('No, Keep It')).toBeInTheDocument();
    });

    it('uses default button labels', () => {
      render(<ConfirmDialog {...defaultProps} />);

      expect(screen.getByText('Confirm')).toBeInTheDocument();
      expect(screen.getByText('Cancel')).toBeInTheDocument();
    });
  });

  describe('accessibility', () => {
    it('has correct ARIA role', () => {
      render(<ConfirmDialog {...defaultProps} />);

      const dialog = screen.getByRole('dialog');
      expect(dialog).toBeInTheDocument();
      expect(dialog).toHaveAttribute('aria-modal', 'true');
    });

    it('has aria-labelledby pointing to title', () => {
      render(<ConfirmDialog {...defaultProps} />);

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveAttribute('aria-labelledby', 'confirm-dialog-title');

      const title = document.getElementById('confirm-dialog-title');
      expect(title).toBeInTheDocument();
      expect(title).toHaveTextContent('Delete Item');
    });

    it('has accessible heading', () => {
      render(<ConfirmDialog {...defaultProps} />);

      expect(screen.getByRole('heading', { name: 'Delete Item' })).toBeInTheDocument();
    });

    it('has accessible buttons', () => {
      render(<ConfirmDialog {...defaultProps} />);

      expect(screen.getByTestId('confirm-dialog-confirm')).toBeEnabled();
      expect(screen.getByTestId('confirm-dialog-cancel')).toBeEnabled();
    });
  });

  describe('keyboard navigation', () => {
    it('closes on Escape key', () => {
      const onCancel = vi.fn();
      render(<ConfirmDialog {...defaultProps} onCancel={onCancel} />);

      fireEvent.keyDown(document, { key: 'Escape' });

      expect(onCancel).toHaveBeenCalledTimes(1);
    });

    it('does not call onCancel on Escape when closed', () => {
      const onCancel = vi.fn();
      render(<ConfirmDialog {...defaultProps} isOpen={false} onCancel={onCancel} />);

      fireEvent.keyDown(document, { key: 'Escape' });

      expect(onCancel).not.toHaveBeenCalled();
    });

    it('does not close on other keys', () => {
      const onCancel = vi.fn();
      render(<ConfirmDialog {...defaultProps} onCancel={onCancel} />);

      fireEvent.keyDown(document, { key: 'Enter' });
      fireEvent.keyDown(document, { key: 'Space' });
      fireEvent.keyDown(document, { key: 'Tab' });

      expect(onCancel).not.toHaveBeenCalled();
    });
  });

  describe('interactions', () => {
    it('calls onConfirm when confirm button is clicked', () => {
      const onConfirm = vi.fn();
      render(<ConfirmDialog {...defaultProps} onConfirm={onConfirm} />);

      fireEvent.click(screen.getByTestId('confirm-dialog-confirm'));

      expect(onConfirm).toHaveBeenCalledTimes(1);
    });

    it('calls onCancel when cancel button is clicked', () => {
      const onCancel = vi.fn();
      render(<ConfirmDialog {...defaultProps} onCancel={onCancel} />);

      fireEvent.click(screen.getByTestId('confirm-dialog-cancel'));

      expect(onCancel).toHaveBeenCalledTimes(1);
    });

    it('calls onCancel when backdrop is clicked', () => {
      const onCancel = vi.fn();
      render(<ConfirmDialog {...defaultProps} onCancel={onCancel} />);

      fireEvent.click(screen.getByTestId('confirm-dialog-backdrop'));

      expect(onCancel).toHaveBeenCalledTimes(1);
    });

    it('does not close when dialog content is clicked', () => {
      const onCancel = vi.fn();
      const onConfirm = vi.fn();
      render(
        <ConfirmDialog
          {...defaultProps}
          onCancel={onCancel}
          onConfirm={onConfirm}
        />
      );

      // Click on the dialog title (inside the dialog)
      fireEvent.click(screen.getByText('Delete Item'));

      expect(onCancel).not.toHaveBeenCalled();
      expect(onConfirm).not.toHaveBeenCalled();
    });
  });

  describe('variants', () => {
    it('renders danger variant by default', () => {
      render(<ConfirmDialog {...defaultProps} />);

      // Default variant is danger - just verify dialog renders
      expect(screen.getByTestId('confirm-dialog')).toBeInTheDocument();
    });

    it('renders warning variant', () => {
      render(<ConfirmDialog {...defaultProps} variant="warning" />);

      expect(screen.getByTestId('confirm-dialog')).toBeInTheDocument();
    });
  });
});
