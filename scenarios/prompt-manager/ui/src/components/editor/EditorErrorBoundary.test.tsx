/**
 * Tests for EditorErrorBoundary component.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { EditorErrorBoundary } from './EditorErrorBoundary'

// Component that throws an error
function ThrowError({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) {
    throw new Error('Test error')
  }
  return <div data-testid="child">Child content</div>
}

describe('EditorErrorBoundary', () => {
  // Suppress console.error for cleaner test output
  beforeEach(() => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should render children when there is no error', () => {
    render(
      <EditorErrorBoundary>
        <ThrowError shouldThrow={false} />
      </EditorErrorBoundary>
    )

    expect(screen.getByTestId('child')).toBeInTheDocument()
  })

  it('should render fallback UI when there is an error', () => {
    render(
      <EditorErrorBoundary>
        <ThrowError shouldThrow={true} />
      </EditorErrorBoundary>
    )

    expect(screen.getByText('Editor Error')).toBeInTheDocument()
    expect(screen.getByText(/Something went wrong/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Try Again/i })).toBeInTheDocument()
  })

  it('should display error message in details', () => {
    render(
      <EditorErrorBoundary>
        <ThrowError shouldThrow={true} />
      </EditorErrorBoundary>
    )

    const details = screen.getByText('Technical details')
    fireEvent.click(details)

    expect(screen.getByText('Test error')).toBeInTheDocument()
  })

  it('should call onError callback when error occurs', () => {
    const onError = vi.fn()

    render(
      <EditorErrorBoundary onError={onError}>
        <ThrowError shouldThrow={true} />
      </EditorErrorBoundary>
    )

    expect(onError).toHaveBeenCalledWith(
      expect.any(Error),
      expect.objectContaining({ componentStack: expect.any(String) })
    )
  })

  it('should reset when Try Again is clicked', () => {
    // Use a stateful wrapper to control whether the child throws
    let shouldThrow = true
    const ControlledThrow = () => {
      if (shouldThrow) {
        throw new Error('Test error')
      }
      return <div data-testid="child">Child content</div>
    }

    const { rerender } = render(
      <EditorErrorBoundary>
        <ControlledThrow />
      </EditorErrorBoundary>
    )

    expect(screen.getByText('Editor Error')).toBeInTheDocument()

    // Change the state before clicking reset
    shouldThrow = false

    // Click Try Again
    fireEvent.click(screen.getByRole('button', { name: /Try Again/i }))

    // Force re-render after reset
    rerender(
      <EditorErrorBoundary>
        <ControlledThrow />
      </EditorErrorBoundary>
    )

    expect(screen.getByTestId('child')).toBeInTheDocument()
  })

  it('should use custom fallback when provided', () => {
    const customFallback = (error: Error, reset: () => void) => (
      <div>
        <span data-testid="custom-error">Custom: {error.message}</span>
        <button onClick={reset}>Custom Reset</button>
      </div>
    )

    render(
      <EditorErrorBoundary fallback={customFallback}>
        <ThrowError shouldThrow={true} />
      </EditorErrorBoundary>
    )

    expect(screen.getByTestId('custom-error')).toHaveTextContent('Custom: Test error')
    expect(screen.getByRole('button', { name: 'Custom Reset' })).toBeInTheDocument()
  })

  it('should apply className to fallback UI', () => {
    const { container } = render(
      <EditorErrorBoundary className="custom-class">
        <ThrowError shouldThrow={true} />
      </EditorErrorBoundary>
    )

    // The fallback div should have the custom class
    const fallbackDiv = container.querySelector('.custom-class')
    expect(fallbackDiv).toBeInTheDocument()
  })
})
