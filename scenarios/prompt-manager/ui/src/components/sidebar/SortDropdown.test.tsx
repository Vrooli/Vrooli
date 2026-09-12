import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { SortDropdown } from './SortDropdown'

describe('SortDropdown', () => {
  const defaultConfig = { field: 'alphabetical' as const, direction: 'asc' as const }

  it('renders trigger with current sort label', () => {
    render(<SortDropdown sortConfig={defaultConfig} onSortConfigChange={() => {}} />)
    expect(screen.getByTestId('sort-dropdown-trigger')).toHaveTextContent('A–Z')
  })

  it('opens menu on click', () => {
    render(<SortDropdown sortConfig={defaultConfig} onSortConfigChange={() => {}} />)
    fireEvent.click(screen.getByTestId('sort-dropdown-trigger'))
    expect(screen.getByTestId('sort-dropdown-menu')).toBeInTheDocument()
  })

  it('selects a sort option', () => {
    const onChange = vi.fn()
    render(<SortDropdown sortConfig={defaultConfig} onSortConfigChange={onChange} />)

    fireEvent.click(screen.getByTestId('sort-dropdown-trigger'))
    fireEvent.click(screen.getByTestId('sort-option-mostUsed'))

    expect(onChange).toHaveBeenCalledWith({ field: 'mostUsed', direction: 'desc' })
  })

  it('toggles sort direction via menu', () => {
    const onChange = vi.fn()
    render(<SortDropdown sortConfig={defaultConfig} onSortConfigChange={onChange} />)

    fireEvent.click(screen.getByTestId('sort-dropdown-trigger'))
    fireEvent.click(screen.getByTestId('sort-direction-toggle'))
    expect(onChange).toHaveBeenCalledWith({ field: 'alphabetical', direction: 'desc' })
  })
})
