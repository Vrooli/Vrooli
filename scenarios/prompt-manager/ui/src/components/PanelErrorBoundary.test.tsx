/**
 * Tests for PanelErrorBoundary component.
 *
 * Tests cover:
 * - Normal rendering of children
 * - Error catching and fallback display
 * - Retry functionality
 * - Panel name display in errors
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { PanelErrorBoundary } from './PanelErrorBoundary'

// Component that throws an error
function ThrowingComponent({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) {
    throw new Error('Test error message')
  }
  return <div data-testid="child-content">Child content</div>
}

// Suppress console.error for cleaner test output
beforeEach(() => {
  vi.spyOn(console, 'error').mockImplementation(() => {})
})

describe('PanelErrorBoundary', () => {
  describe('normal rendering', () => {
    it('should render children when no error occurs', () => {
      render(
        <PanelErrorBoundary>
          <div data-testid="child">Test child</div>
        </PanelErrorBoundary>
      )

      expect(screen.getByTestId('child')).toBeInTheDocument()
      expect(screen.getByText('Test child')).toBeInTheDocument()
    })

    it('should pass through multiple children', () => {
      render(
        <PanelErrorBoundary>
          <div data-testid="child1">First</div>
          <div data-testid="child2">Second</div>
        </PanelErrorBoundary>
      )

      expect(screen.getByTestId('child1')).toBeInTheDocument()
      expect(screen.getByTestId('child2')).toBeInTheDocument()
    })
  })

  describe('error handling', () => {
    it('should catch errors and display fallback UI', () => {
      render(
        <PanelErrorBoundary>
          <ThrowingComponent shouldThrow={true} />
        </PanelErrorBoundary>
      )

      // Should show error UI instead of child
      expect(screen.queryByTestId('child-content')).not.toBeInTheDocument()
      expect(screen.getByText('Something went wrong')).toBeInTheDocument()
      expect(screen.getByText('Test error message')).toBeInTheDocument()
    })

    it('should display panel name in error message when provided', () => {
      render(
        <PanelErrorBoundary panelName="Editor">
          <ThrowingComponent shouldThrow={true} />
        </PanelErrorBoundary>
      )

      expect(screen.getByText('Editor Error')).toBeInTheDocument()
    })

    it('should show generic message when no panel name provided', () => {
      render(
        <PanelErrorBoundary>
          <ThrowingComponent shouldThrow={true} />
        </PanelErrorBoundary>
      )

      expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    })

    it('should display the error message', () => {
      render(
        <PanelErrorBoundary>
          <ThrowingComponent shouldThrow={true} />
        </PanelErrorBoundary>
      )

      expect(screen.getByText('Test error message')).toBeInTheDocument()
    })

    it('should log error to console', () => {
      render(
        <PanelErrorBoundary panelName="TestPanel">
          <ThrowingComponent shouldThrow={true} />
        </PanelErrorBoundary>
      )

      expect(console.error).toHaveBeenCalled()
    })
  })

  describe('retry functionality', () => {
    it('should show Try Again button', () => {
      render(
        <PanelErrorBoundary>
          <ThrowingComponent shouldThrow={true} />
        </PanelErrorBoundary>
      )

      expect(screen.getByRole('button', { name: 'Try Again' })).toBeInTheDocument()
    })

    it('should reset error state when Try Again is clicked', () => {
      // Use a stateful wrapper to control when error is thrown
      let shouldThrow = true

      function StatefulWrapper() {
        // This is a bit hacky but needed because error boundaries
        // don't re-render children after reset - React needs the
        // children to be different to re-mount them
        if (!shouldThrow) {
          return <div data-testid="recovered">Recovered!</div>
        }
        throw new Error('Test error')
      }

      const { rerender } = render(
        <PanelErrorBoundary>
          <StatefulWrapper />
        </PanelErrorBoundary>
      )

      // Verify error state
      expect(screen.getByText('Something went wrong')).toBeInTheDocument()

      // Mark as no longer throwing
      shouldThrow = false

      // Click retry
      fireEvent.click(screen.getByRole('button', { name: 'Try Again' }))

      // Force rerender to pick up the state change
      rerender(
        <PanelErrorBoundary>
          <StatefulWrapper />
        </PanelErrorBoundary>
      )

      // Should show recovered content
      expect(screen.getByTestId('recovered')).toBeInTheDocument()
    })
  })

  describe('minimal mode', () => {
    it('should render children when no error occurs', () => {
      render(
        <PanelErrorBoundary minimal>
          <div data-testid="child">Test child</div>
        </PanelErrorBoundary>
      )

      expect(screen.getByTestId('child')).toBeInTheDocument()
    })

    it('should render nothing when child throws', () => {
      const { container } = render(
        <PanelErrorBoundary minimal panelName="Tooltip">
          <ThrowingComponent shouldThrow={true} />
        </PanelErrorBoundary>
      )

      // Should not show any error UI or child content
      expect(screen.queryByTestId('child-content')).not.toBeInTheDocument()
      expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument()
      expect(screen.queryByText('Tooltip Error')).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Try Again' })).not.toBeInTheDocument()
      // Container should be empty
      expect(container.innerHTML).toBe('')
    })

    it('should still log error to console', () => {
      render(
        <PanelErrorBoundary minimal panelName="Toolbar">
          <ThrowingComponent shouldThrow={true} />
        </PanelErrorBoundary>
      )

      expect(console.error).toHaveBeenCalled()
    })
  })

  describe('className prop', () => {
    it('should apply custom className to error container', () => {
      const { container } = render(
        <PanelErrorBoundary className="custom-error-class">
          <ThrowingComponent shouldThrow={true} />
        </PanelErrorBoundary>
      )

      // Find the error container and check for the class
      const errorContainer = container.querySelector('.custom-error-class')
      expect(errorContainer).toBeInTheDocument()
    })
  })
})
