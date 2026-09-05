import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { FilterPopover } from './FilterPopover'
import { DEFAULT_FILTER_STATE } from '@/types/filterSort'

describe('FilterPopover', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    onApply: vi.fn(),
    filterState: DEFAULT_FILTER_STATE,
    availableTags: ['automation', 'writing', 'coding'],
    availableFolders: ['core', 'local', 'drafts'],
  }

  it('renders all sections when open', () => {
    render(<FilterPopover {...defaultProps} />)
    expect(screen.getByText('Storage')).toBeInTheDocument()
    expect(screen.getByText('Tags')).toBeInTheDocument()
    expect(screen.getByText('Usage')).toBeInTheDocument()
    expect(screen.getByText('Rating')).toBeInTheDocument()
    expect(screen.getByText('Status')).toBeInTheDocument()
  })

  it('renders storage checkboxes', () => {
    render(<FilterPopover {...defaultProps} />)
    expect(screen.getByTestId('filter-storage-core')).toBeInTheDocument()
    expect(screen.getByTestId('filter-storage-local')).toBeInTheDocument()
    expect(screen.getByTestId('filter-storage-drafts')).toBeInTheDocument()
  })

  it('renders tag checkboxes', () => {
    render(<FilterPopover {...defaultProps} />)
    expect(screen.getByTestId('filter-tag-automation')).toBeInTheDocument()
    expect(screen.getByTestId('filter-tag-writing')).toBeInTheDocument()
  })

  it('applies pending state on Apply click', () => {
    const onApply = vi.fn()
    render(<FilterPopover {...defaultProps} onApply={onApply} />)

    // Check a storage option
    fireEvent.click(screen.getByTestId('filter-storage-core'))
    // Click Apply
    fireEvent.click(screen.getByTestId('filter-apply'))

    expect(onApply).toHaveBeenCalledWith(
      expect.objectContaining({ storage: ['core'] })
    )
  })

  it('does not apply on Cancel', () => {
    const onApply = vi.fn()
    const onClose = vi.fn()
    render(<FilterPopover {...defaultProps} onApply={onApply} onClose={onClose} />)

    fireEvent.click(screen.getByTestId('filter-storage-core'))
    fireEvent.click(screen.getByTestId('filter-cancel'))

    expect(onApply).not.toHaveBeenCalled()
    expect(onClose).toHaveBeenCalled()
  })

  it('clears all pending filters', () => {
    const onApply = vi.fn()
    const state = { ...DEFAULT_FILTER_STATE, tags: ['automation'] }
    render(<FilterPopover {...defaultProps} filterState={state} onApply={onApply} />)

    fireEvent.click(screen.getByTestId('filter-clear-all'))
    fireEvent.click(screen.getByTestId('filter-apply'))

    expect(onApply).toHaveBeenCalledWith(DEFAULT_FILTER_STATE)
  })

  it('toggles usage preset', () => {
    const onApply = vi.fn()
    render(<FilterPopover {...defaultProps} onApply={onApply} />)

    // Expand usage section first
    fireEvent.click(screen.getByText('Usage'))
    fireEvent.click(screen.getByTestId('filter-usage-neverUsed'))
    fireEvent.click(screen.getByTestId('filter-apply'))

    expect(onApply).toHaveBeenCalledWith(
      expect.objectContaining({ usagePreset: 'neverUsed' })
    )
  })

  it('sets rating threshold', () => {
    const onApply = vi.fn()
    render(<FilterPopover {...defaultProps} onApply={onApply} />)

    // Expand rating section
    fireEvent.click(screen.getByText('Rating'))
    fireEvent.click(screen.getByTestId('filter-rating-3'))
    fireEvent.click(screen.getByTestId('filter-apply'))

    expect(onApply).toHaveBeenCalledWith(
      expect.objectContaining({ minRating: 3 })
    )
  })

  it('sets status filter', () => {
    const onApply = vi.fn()
    render(<FilterPopover {...defaultProps} onApply={onApply} />)

    // Expand status section
    fireEvent.click(screen.getByText('Status'))
    fireEvent.click(screen.getByTestId('filter-status-draft'))
    fireEvent.click(screen.getByTestId('filter-apply'))

    expect(onApply).toHaveBeenCalledWith(
      expect.objectContaining({ status: 'draft' })
    )
  })
})
