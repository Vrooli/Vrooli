// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#ui-component-tests
// [REQ:REQ-P1-005a] Route-Level Error Boundaries - Unit tests for error handling
// [REQ:REQ-P1-005b] Error Recovery UI - Tests for error display and retry
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { ErrorBoundary } from './ErrorBoundary';

// Component that throws an error
const ThrowingComponent = ({ shouldThrow }: { shouldThrow: boolean }) => {
  if (shouldThrow) {
    throw new Error('Test error message');
  }
  return <div data-testid="child-content">Child content rendered</div>;
};

describe('ErrorBoundary', () => {
  // Suppress error logging during tests
  const originalError = console.error;

  beforeEach(() => {
    console.error = vi.fn();
  });

  afterEach(() => {
    console.error = originalError;
  });

  describe('rendering children', () => {
    it('renders children when no error', () => {
      render(
        <ErrorBoundary>
          <ThrowingComponent shouldThrow={false} />
        </ErrorBoundary>
      );

      expect(screen.getByTestId('child-content')).toBeInTheDocument();
      expect(screen.getByText('Child content rendered')).toBeInTheDocument();
    });

    it('renders multiple children', () => {
      render(
        <ErrorBoundary>
          <div data-testid="first">First</div>
          <div data-testid="second">Second</div>
        </ErrorBoundary>
      );

      expect(screen.getByTestId('first')).toBeInTheDocument();
      expect(screen.getByTestId('second')).toBeInTheDocument();
    });
  });

  describe('error handling', () => {
    it('catches and displays error when child throws', () => {
      render(
        <ErrorBoundary>
          <ThrowingComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      expect(screen.getByText('Test error message')).toBeInTheDocument();
    });

    it('displays error description text', () => {
      render(
        <ErrorBoundary>
          <ThrowingComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByText(/An error occurred while rendering/)).toBeInTheDocument();
    });

    it('calls onError callback when error occurs', () => {
      const onError = vi.fn();

      render(
        <ErrorBoundary onError={onError}>
          <ThrowingComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(onError).toHaveBeenCalledTimes(1);
      const firstCall = onError.mock.calls[0];
      expect(firstCall).toBeDefined();
      const errorArg = firstCall?.[0] as Error;
      expect(errorArg).toBeInstanceOf(Error);
      expect(errorArg.message).toBe('Test error message');
    });
  });

  describe('recovery functionality', () => {
    it('displays Try Again button', () => {
      render(
        <ErrorBoundary>
          <ThrowingComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
    });

    it('resets error state when Try Again is clicked', () => {
      let shouldThrow = true;
      const TestComponent = () => {
        if (shouldThrow) {
          throw new Error('Test error');
        }
        return <div data-testid="recovered">Recovered!</div>;
      };

      const { rerender } = render(
        <ErrorBoundary>
          <TestComponent />
        </ErrorBoundary>
      );

      // Should show error initially
      expect(screen.getByText('Something went wrong')).toBeInTheDocument();

      // Fix the error condition
      shouldThrow = false;

      // Click Try Again
      fireEvent.click(screen.getByRole('button', { name: /try again/i }));

      // Force rerender with new component state
      rerender(
        <ErrorBoundary>
          <TestComponent />
        </ErrorBoundary>
      );

      // Should show recovered content
      expect(screen.getByTestId('recovered')).toBeInTheDocument();
    });
  });

  describe('custom fallback', () => {
    it('renders custom fallback when provided', () => {
      const CustomFallback = <div data-testid="custom-fallback">Custom error UI</div>;

      render(
        <ErrorBoundary fallback={CustomFallback}>
          <ThrowingComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByTestId('custom-fallback')).toBeInTheDocument();
      expect(screen.getByText('Custom error UI')).toBeInTheDocument();
      expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
    });

    it('does not render custom fallback when no error', () => {
      const CustomFallback = <div data-testid="custom-fallback">Custom error UI</div>;

      render(
        <ErrorBoundary fallback={CustomFallback}>
          <ThrowingComponent shouldThrow={false} />
        </ErrorBoundary>
      );

      expect(screen.queryByTestId('custom-fallback')).not.toBeInTheDocument();
      expect(screen.getByTestId('child-content')).toBeInTheDocument();
    });
  });

  describe('accessibility', () => {
    it('error message container is accessible', () => {
      render(
        <ErrorBoundary>
          <ThrowingComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // Heading should be present
      expect(screen.getByRole('heading', { name: /something went wrong/i })).toBeInTheDocument();

      // Button should be focusable
      const button = screen.getByRole('button', { name: /try again/i });
      expect(button).toBeEnabled();
    });
  });
});
