import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { FilterSortToolbar } from './FilterSortToolbar'
import { DEFAULT_FILTER_STATE, DEFAULT_SORT_CONFIG, DEFAULT_VIEW_MODE, DEFAULT_DETAIL_MODE } from '@/types/filterSort'

describe('FilterSortToolbar', () => {
  const defaultProps = {
    filterState: DEFAULT_FILTER_STATE,
    onFilterStateChange: vi.fn(),
    sortConfig: DEFAULT_SORT_CONFIG,
    onSortConfigChange: vi.fn(),
    viewMode: DEFAULT_VIEW_MODE,
    onViewModeChange: vi.fn(),
    detailMode: DEFAULT_DETAIL_MODE,
    onDetailModeChange: vi.fn(),
    availableTags: ['a', 'b'],
    availableFolders: ['core', 'local'],
  }

  it('renders filter and view mode controls', () => {
    render(<FilterSortToolbar {...defaultProps} />)
    expect(screen.getByTestId('filter-trigger')).toBeInTheDocument()
    expect(screen.getByLabelText('Tree view')).toBeInTheDocument()
  })

  it('hides sort dropdown in tree view mode', () => {
    render(<FilterSortToolbar {...defaultProps} viewMode="tree" />)
    expect(screen.queryByTestId('sort-dropdown-trigger')).not.toBeInTheDocument()
  })

  it('shows sort dropdown in list view mode', () => {
    render(<FilterSortToolbar {...defaultProps} viewMode="list" />)
    expect(screen.getByTestId('sort-dropdown-trigger')).toBeInTheDocument()
  })

  it('shows sort dropdown in card view mode', () => {
    render(<FilterSortToolbar {...defaultProps} viewMode="card" />)
    expect(screen.getByTestId('sort-dropdown-trigger')).toBeInTheDocument()
  })

  it('shows active filter count badge', () => {
    const state = { ...DEFAULT_FILTER_STATE, tags: ['a', 'b'], storage: ['core'] }
    render(<FilterSortToolbar {...defaultProps} filterState={state} />)
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('opens filter popover on click', () => {
    render(<FilterSortToolbar {...defaultProps} />)
    fireEvent.click(screen.getByTestId('filter-trigger'))
    expect(screen.getByTestId('filter-popover')).toBeInTheDocument()
  })

  it('delegates view mode change', () => {
    const onChange = vi.fn()
    render(<FilterSortToolbar {...defaultProps} onViewModeChange={onChange} />)
    fireEvent.click(screen.getByLabelText('List view'))
    expect(onChange).toHaveBeenCalledWith('list')
  })
})
