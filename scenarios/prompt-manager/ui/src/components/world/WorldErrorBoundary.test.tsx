/**
 * Tests for WorldErrorBoundary component.
 *
 * Tests cover:
 * - Normal rendering of children
 * - Error catching and fallback display
 * - Minimal mode (silent failures)
 * - Component name tracking
 * - Reset functionality
 * - WebGL error detection
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { WorldErrorBoundary } from './WorldErrorBoundary'

// Component that throws an error
function ThrowError({
  shouldThrow,
  errorMessage = 'Test error',
}: {
  shouldThrow: boolean
  errorMessage?: string
}) {
  if (shouldThrow) {
    throw new Error(errorMessage)
  }
  return <div data-testid="child">Child content</div>
}

describe('WorldErrorBoundary', () => {
  // Suppress console.error for cleaner test output
  beforeEach(() => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.spyOn(console, 'warn').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('normal rendering', () => {
    it('should render children when there is no error', () => {
      render(
        <WorldErrorBoundary>
          <ThrowError shouldThrow={false} />
        </WorldErrorBoundary>
      )

      expect(screen.getByTestId('child')).toBeInTheDocument()
    })

    it('should pass through multiple children', () => {
      render(
        <WorldErrorBoundary>
          <div data-testid="child1">First</div>
          <div data-testid="child2">Second</div>
        </WorldErrorBoundary>
      )

      expect(screen.getByTestId('child1')).toBeInTheDocument()
      expect(screen.getByTestId('child2')).toBeInTheDocument()
    })
  })

  describe('error handling', () => {
    it('should render fallback UI when there is an error', () => {
      render(
        <WorldErrorBoundary>
          <ThrowError shouldThrow={true} />
        </WorldErrorBoundary>
      )

      expect(screen.getByText('3D World Error')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Try Again/i })).toBeInTheDocument()
    })

    it('should display component name in error message', () => {
      render(
        <WorldErrorBoundary componentName="TestComponent">
          <ThrowError shouldThrow={true} />
        </WorldErrorBoundary>
      )

      expect(screen.getByText('TestComponent')).toBeInTheDocument()
    })

    it('should display error message in details', () => {
      render(
        <WorldErrorBoundary>
          <ThrowError shouldThrow={true} errorMessage="Custom error message" />
        </WorldErrorBoundary>
      )

      const details = screen.getByText('Technical details')
      fireEvent.click(details)

      expect(screen.getByText(/Custom error message/)).toBeInTheDocument()
    })

    it('should call onError callback when error occurs', () => {
      const onError = vi.fn()

      render(
        <WorldErrorBoundary onError={onError} componentName="TestComponent">
          <ThrowError shouldThrow={true} />
        </WorldErrorBoundary>
      )

      expect(onError).toHaveBeenCalledWith(expect.any(Error), 'TestComponent')
    })

    it('should log error to console', () => {
      render(
        <WorldErrorBoundary componentName="TestComponent">
          <ThrowError shouldThrow={true} />
        </WorldErrorBoundary>
      )

      expect(console.error).toHaveBeenCalled()
    })
  })

  describe('minimal mode', () => {
    it('should return null on error when minimal is true', () => {
      const { container } = render(
        <WorldErrorBoundary minimal>
          <ThrowError shouldThrow={true} />
        </WorldErrorBoundary>
      )

      // Should not render any error UI
      expect(container.innerHTML).toBe('')
      expect(screen.queryByText('3D World Error')).not.toBeInTheDocument()
    })

    it('should still call onError in minimal mode', () => {
      const onError = vi.fn()

      render(
        <WorldErrorBoundary minimal onError={onError}>
          <ThrowError shouldThrow={true} />
        </WorldErrorBoundary>
      )

      expect(onError).toHaveBeenCalled()
    })

    it('should still log to console in minimal mode', () => {
      render(
        <WorldErrorBoundary minimal>
          <ThrowError shouldThrow={true} />
        </WorldErrorBoundary>
      )

      expect(console.error).toHaveBeenCalled()
    })
  })

  describe('WebGL error detection', () => {
    it('should detect WebGL context errors', () => {
      render(
        <WorldErrorBoundary>
          <ThrowError shouldThrow={true} errorMessage="WebGL context lost" />
        </WorldErrorBoundary>
      )

      expect(console.warn).toHaveBeenCalledWith(
        expect.stringContaining('WebGL/Three.js error detected')
      )
    })

    it('should detect THREE.js errors', () => {
      render(
        <WorldErrorBoundary>
          <ThrowError shouldThrow={true} errorMessage="THREE.WebGLRenderer error" />
        </WorldErrorBoundary>
      )

      expect(console.warn).toHaveBeenCalledWith(
        expect.stringContaining('WebGL/Three.js error detected')
      )
    })

    it('should show WebGL-specific message in fallback', () => {
      render(
        <WorldErrorBoundary>
          <ThrowError shouldThrow={true} errorMessage="WebGL context lost" />
        </WorldErrorBoundary>
      )

      expect(screen.getByText(/WebGL rendering issue/)).toBeInTheDocument()
    })
  })

  describe('reset functionality', () => {
    it('should reset when Try Again is clicked', () => {
      let shouldThrow = true
      const ControlledThrow = () => {
        if (shouldThrow) {
          throw new Error('Test error')
        }
        return <div data-testid="child">Child content</div>
      }

      const { rerender } = render(
        <WorldErrorBoundary>
          <ControlledThrow />
        </WorldErrorBoundary>
      )

      expect(screen.getByText('3D World Error')).toBeInTheDocument()

      // Change the state before clicking reset
      shouldThrow = false

      // Click Try Again
      fireEvent.click(screen.getByRole('button', { name: /Try Again/i }))

      // Force re-render after reset
      rerender(
        <WorldErrorBoundary>
          <ControlledThrow />
        </WorldErrorBoundary>
      )

      expect(screen.getByTestId('child')).toBeInTheDocument()
    })
  })

  describe('custom fallback', () => {
    it('should use custom fallback when provided', () => {
      const customFallback = (error: Error, reset: () => void, componentName: string) => (
        <div>
          <span data-testid="custom-error">Custom: {componentName} - {error.message}</span>
          <button onClick={reset}>Custom Reset</button>
        </div>
      )

      render(
        <WorldErrorBoundary fallback={customFallback} componentName="TestComponent">
          <ThrowError shouldThrow={true} />
        </WorldErrorBoundary>
      )

      expect(screen.getByTestId('custom-error')).toHaveTextContent(
        'Custom: TestComponent - Test error'
      )
      expect(screen.getByRole('button', { name: 'Custom Reset' })).toBeInTheDocument()
    })

    it('should not use custom fallback in minimal mode', () => {
      const customFallback = vi.fn(() => <div>Custom</div>)

      const { container } = render(
        <WorldErrorBoundary fallback={customFallback} minimal>
          <ThrowError shouldThrow={true} />
        </WorldErrorBoundary>
      )

      expect(customFallback).not.toHaveBeenCalled()
      expect(container.innerHTML).toBe('')
    })
  })

  describe('className prop', () => {
    it('should apply className to fallback UI', () => {
      render(
        <WorldErrorBoundary className="custom-class">
          <ThrowError shouldThrow={true} />
        </WorldErrorBoundary>
      )

      const fallbackDiv = screen.getByTestId('world-error-fallback')
      expect(fallbackDiv).toHaveClass('custom-class')
    })
  })
})
