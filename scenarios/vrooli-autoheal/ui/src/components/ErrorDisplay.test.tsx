// ErrorDisplay component tests
// [REQ:FAIL-SAFE-001]
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ErrorDisplay, InlineError } from './ErrorDisplay';
import { APIError } from '../lib/api';

describe('ErrorDisplay', () => {
  describe('with generic Error', () => {
    it('displays error message', () => {
      render(<ErrorDisplay error={new Error('Something broke')} />);

      expect(screen.getByText('Something broke')).toBeInTheDocument();
    });

    it('displays retry button when onRetry provided', () => {
      const onRetry = vi.fn();
      render(<ErrorDisplay error={new Error('Error')} onRetry={onRetry} />);

      const retryButton = screen.getByRole('button', { name: /retry/i });
      expect(retryButton).toBeInTheDocument();

      fireEvent.click(retryButton);
      expect(onRetry).toHaveBeenCalledTimes(1);
    });

    it('hides retry button when onRetry not provided', () => {
      render(<ErrorDisplay error={new Error('Error')} />);

      expect(screen.queryByRole('button', { name: /retry/i })).not.toBeInTheDocument();
    });
  });

  describe('with APIError', () => {
    it('displays user-friendly message for DATABASE_ERROR', () => {
      const error = new APIError('Raw DB error', 'DATABASE_ERROR', 500);
      render(<ErrorDisplay error={error} />);

      expect(screen.getByText(/database.*unavailable/i)).toBeInTheDocument();
    });

    it('displays request ID when available', () => {
      const error = new APIError('Error', 'DATABASE_ERROR', 500, 'req-abc-123');
      render(<ErrorDisplay error={error} />);

      expect(screen.getByText(/req-abc-123/)).toBeInTheDocument();
    });

    it('displays suggested action', () => {
      const error = new APIError('Error', 'DATABASE_ERROR', 500);
      render(<ErrorDisplay error={error} />);

      expect(screen.getByText(/try again/i)).toBeInTheDocument();
    });
  });

  describe('with title prop', () => {
    it('displays custom title', () => {
      render(<ErrorDisplay error={new Error('Err')} title="Connection Failed" />);

      expect(screen.getByText('Connection Failed')).toBeInTheDocument();
    });
  });

  describe('compact mode', () => {
    it('renders in compact layout', () => {
      render(<ErrorDisplay error={new Error('Error')} compact />);

      // Compact mode shouldn't have a full-sized button
      expect(screen.queryByRole('button', { name: /retry/i })).not.toBeInTheDocument();
    });

    it('shows compact retry link in compact mode', () => {
      const onRetry = vi.fn();
      render(<ErrorDisplay error={new Error('Error')} onRetry={onRetry} compact />);

      // In compact mode, we have a smaller retry link
      const retryLink = screen.getByText(/retry/i);
      expect(retryLink).toBeInTheDocument();

      fireEvent.click(retryLink);
      expect(onRetry).toHaveBeenCalledTimes(1);
    });
  });
});

describe('InlineError', () => {
  it('displays error message', () => {
    render(<InlineError message="Something went wrong" />);

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('displays retry button when onRetry provided', () => {
    const onRetry = vi.fn();
    render(<InlineError message="Error" onRetry={onRetry} />);

    const retryButton = screen.getByText(/retry/i);
    fireEvent.click(retryButton);
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('hides retry when onRetry not provided', () => {
    render(<InlineError message="Error" />);

    expect(screen.queryByText(/retry/i)).not.toBeInTheDocument();
  });
});
