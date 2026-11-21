/**
 * Tests for ToolbarButton component.
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { ToolbarButton, ToolbarDivider } from './ToolbarButton'

describe('ToolbarButton', () => {
  it('should render children', () => {
    render(
      <ToolbarButton onClick={() => {}} title="Test">
        <span data-testid="child">Icon</span>
      </ToolbarButton>
    )

    expect(screen.getByTestId('child')).toBeInTheDocument()
  })

  it('should call onClick when clicked', () => {
    const onClick = vi.fn()
    render(
      <ToolbarButton onClick={onClick} title="Test">
        Click me
      </ToolbarButton>
    )

    fireEvent.click(screen.getByRole('button'))
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('should have title attribute for tooltip', () => {
    render(
      <ToolbarButton onClick={() => {}} title="My Tooltip">
        Button
      </ToolbarButton>
    )

    expect(screen.getByRole('button')).toHaveAttribute('title', 'My Tooltip')
  })

  it('should be disabled when disabled prop is true', () => {
    const onClick = vi.fn()
    render(
      <ToolbarButton onClick={onClick} title="Test" disabled>
        Button
      </ToolbarButton>
    )

    const button = screen.getByRole('button')
    expect(button).toBeDisabled()

    fireEvent.click(button)
    expect(onClick).not.toHaveBeenCalled()
  })

  it('should apply active styles when isActive is true', () => {
    render(
      <ToolbarButton onClick={() => {}} title="Test" isActive>
        Button
      </ToolbarButton>
    )

    const button = screen.getByRole('button')
    expect(button.className).toContain('bg-primary/30')
  })

  it('should apply inactive styles when isActive is false', () => {
    render(
      <ToolbarButton onClick={() => {}} title="Test" isActive={false}>
        Button
      </ToolbarButton>
    )

    const button = screen.getByRole('button')
    expect(button.className).toContain('text-muted-foreground')
  })

  it('should have type="button" to prevent form submission', () => {
    render(
      <ToolbarButton onClick={() => {}} title="Test">
        Button
      </ToolbarButton>
    )

    expect(screen.getByRole('button')).toHaveAttribute('type', 'button')
  })

  it('should apply custom className', () => {
    render(
      <ToolbarButton onClick={() => {}} title="Test" className="custom-class">
        Button
      </ToolbarButton>
    )

    expect(screen.getByRole('button').className).toContain('custom-class')
  })
})

describe('ToolbarDivider', () => {
  it('should render a divider element', () => {
    const { container } = render(<ToolbarDivider />)

    const divider = container.firstChild
    expect(divider).toBeInTheDocument()
    expect(divider).toHaveClass('w-px')
    expect(divider).toHaveClass('h-6')
  })

  it('should apply custom className', () => {
    const { container } = render(<ToolbarDivider className="custom-class" />)

    expect(container.firstChild).toHaveClass('custom-class')
  })
})
