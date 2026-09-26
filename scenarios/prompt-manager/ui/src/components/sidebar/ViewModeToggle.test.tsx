import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { ViewModeToggle } from './ViewModeToggle'

describe('ViewModeToggle', () => {
  it('renders three view mode buttons', () => {
    render(<ViewModeToggle viewMode="tree" onViewModeChange={() => {}} detailMode="full" onDetailModeChange={() => {}} />)
    expect(screen.getByLabelText('Tree view')).toBeInTheDocument()
    expect(screen.getByLabelText('List view')).toBeInTheDocument()
    expect(screen.getByLabelText('Card view')).toBeInTheDocument()
  })

  it('marks the active mode as checked', () => {
    render(<ViewModeToggle viewMode="list" onViewModeChange={() => {}} detailMode="full" onDetailModeChange={() => {}} />)
    expect(screen.getByLabelText('List view')).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByLabelText('Tree view')).toHaveAttribute('aria-checked', 'false')
  })

  it('calls onViewModeChange when a mode is clicked', () => {
    const onChange = vi.fn()
    render(<ViewModeToggle viewMode="tree" onViewModeChange={onChange} detailMode="full" onDetailModeChange={() => {}} />)

    fireEvent.click(screen.getByLabelText('Card view'))
    expect(onChange).toHaveBeenCalledWith('card')
  })
})
