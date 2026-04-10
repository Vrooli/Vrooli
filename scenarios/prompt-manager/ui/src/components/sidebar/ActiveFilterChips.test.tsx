import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ActiveFilterChips } from './ActiveFilterChips'
import { DEFAULT_FILTER_STATE } from '@/types/filterSort'
import type { FilterState } from '@/types/filterSort'

describe('ActiveFilterChips', () => {
  it('renders nothing when no filters active', () => {
    const { container } = render(
      <ActiveFilterChips filterState={DEFAULT_FILTER_STATE} onFilterStateChange={() => {}} />
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders chips for active filters', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, tags: ['automation'], storage: ['core'] }
    render(<ActiveFilterChips filterState={state} onFilterStateChange={() => {}} />)

    expect(screen.getByText('Core')).toBeInTheDocument()
    expect(screen.getByText('automation')).toBeInTheDocument()
  })

  it('removes a chip on click', () => {
    const onChange = vi.fn()
    const state: FilterState = { ...DEFAULT_FILTER_STATE, tags: ['a', 'b'] }
    render(<ActiveFilterChips filterState={state} onFilterStateChange={onChange} />)

    fireEvent.click(screen.getByTestId('remove-chip-tag:a'))
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ tags: ['b'] }))
  })

  it('shows Clear all when 2+ chips', () => {
    const state: FilterState = { ...DEFAULT_FILTER_STATE, tags: ['a', 'b'] }
    render(<ActiveFilterChips filterState={state} onFilterStateChange={() => {}} />)
    expect(screen.getByTestId('clear-all-filters')).toBeInTheDocument()
  })

  it('clears all on Clear all click', () => {
    const onChange = vi.fn()
    const state: FilterState = { ...DEFAULT_FILTER_STATE, tags: ['a', 'b'] }
    render(<ActiveFilterChips filterState={state} onFilterStateChange={onChange} />)

    fireEvent.click(screen.getByTestId('clear-all-filters'))
    expect(onChange).toHaveBeenCalledWith(DEFAULT_FILTER_STATE)
  })
})
