/**
 * Tests for DiscoverControls component.
 *
 * Covers: complexity buttons from budgetConfig, gear button toggle,
 * settings panel (budget editor + filter editor) save/cancel, budget gauge rendering.
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { DiscoverControls } from './DiscoverControls'
import type { BudgetConfig, DiscoverFilterConfig } from '@/lib/schemas'

const defaultBudgetConfig: BudgetConfig = {
  minor: 4000,
  moderate: 8000,
  major: 12000,
  architectural: 18000,
}

const defaultFilterConfig: DiscoverFilterConfig = {
  includeDrafts: false,
  excludeModes: ['scope'],
  excludeIds: [],
  excludeTags: [],
}

const noopToggle = vi.fn()
const noopComplexity = vi.fn()
const noopBudgetSave = vi.fn()
const noopFilterSave = vi.fn()

function renderControls(overrides: Record<string, unknown> = {}) {
  const props = {
    useDiscover: true,
    onToggleDiscover: noopToggle,
    complexity: undefined as string | undefined,
    onComplexityChange: noopComplexity,
    budgetConfig: defaultBudgetConfig,
    onBudgetConfigSave: noopBudgetSave,
    filterConfig: defaultFilterConfig,
    onFilterConfigSave: noopFilterSave,
    availableModes: ['steer', 'scope', 'tools', 'meta'],
    availableTags: ['go', 'react', 'deprecated'],
    ...overrides,
  }
  return render(<DiscoverControls {...props} />)
}

describe('DiscoverControls', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Complexity buttons', () => {
    it('renders all 4 tier buttons when discover is enabled', () => {
      renderControls()

      expect(screen.getByText('Minor')).toBeDefined()
      expect(screen.getByText('Moderate')).toBeDefined()
      expect(screen.getByText('Major')).toBeDefined()
      expect(screen.getByText('Architectural')).toBeDefined()
    })

    it('shows budget in title from budgetConfig', () => {
      renderControls()

      const minor = screen.getByText('Minor')
      expect(minor.getAttribute('title')).toContain('4.0K chars')
    })

    it('does not show complexity buttons when discover is off', () => {
      renderControls({ useDiscover: false })

      expect(screen.queryByText('Minor')).toBeNull()
      expect(screen.queryByText('Budget:')).toBeNull()
    })
  })

  describe('Settings panel', () => {
    it('toggles settings visibility on gear click', () => {
      renderControls()

      // Settings should not be visible initially
      expect(screen.queryByText('Budget Tiers')).toBeNull()

      // Click gear button
      const gearButton = screen.getByTitle('Configure discovery settings')
      fireEvent.click(gearButton)

      // Settings panel should now be visible with both sections
      expect(screen.getByText('Budget Tiers')).toBeDefined()
      expect(screen.getByText('Discovery Filters')).toBeDefined()
      expect(screen.getByText('Save')).toBeDefined()
      expect(screen.getByText('Cancel')).toBeDefined()
    })

    it('calls onBudgetConfigSave with edited values on save', () => {
      renderControls()

      // Open settings
      fireEvent.click(screen.getByTitle('Configure discovery settings'))

      // Find inputs and change a value
      const inputs = screen.getAllByRole('spinbutton')
      expect(inputs).toHaveLength(4)

      // Change minor to 5000
      const minorInput = inputs[0] as HTMLElement
      fireEvent.change(minorInput, { target: { value: '5000' } })

      // Save
      fireEvent.click(screen.getByText('Save'))

      expect(noopBudgetSave).toHaveBeenCalledWith({
        minor: 5000,
        moderate: 8000,
        major: 12000,
        architectural: 18000,
      })
    })

    it('discards changes on cancel', () => {
      renderControls()

      // Open settings
      fireEvent.click(screen.getByTitle('Configure discovery settings'))

      // Change a value
      const inputs = screen.getAllByRole('spinbutton')
      fireEvent.change(inputs[0] as HTMLElement, { target: { value: '999' } })

      // Cancel
      fireEvent.click(screen.getByText('Cancel'))

      // Settings should be closed
      expect(screen.queryByText('Budget Tiers')).toBeNull()

      // Neither save callback should have been called
      expect(noopBudgetSave).not.toHaveBeenCalled()
      expect(noopFilterSave).not.toHaveBeenCalled()
    })

    it('shows validation error when budget values are not ascending', () => {
      renderControls()

      fireEvent.click(screen.getByTitle('Configure discovery settings'))

      // Set minor higher than moderate (invalid)
      const inputs = screen.getAllByRole('spinbutton')
      fireEvent.change(inputs[0] as HTMLElement, { target: { value: '9000' } })

      // Validation message should show
      expect(screen.getByText('Values must be ascending')).toBeDefined()
    })
  })

  describe('Filter controls', () => {
    it('renders include drafts toggle in settings', () => {
      renderControls()
      fireEvent.click(screen.getByTitle('Configure discovery settings'))

      expect(screen.getByText('Include drafts')).toBeDefined()
    })

    it('renders available modes as chips', () => {
      renderControls()
      fireEvent.click(screen.getByTitle('Configure discovery settings'))

      expect(screen.getByText('steer')).toBeDefined()
      expect(screen.getByText('scope')).toBeDefined()
      expect(screen.getByText('tools')).toBeDefined()
    })

    it('renders available tags as chips', () => {
      renderControls()
      fireEvent.click(screen.getByTitle('Configure discovery settings'))

      expect(screen.getByText('go')).toBeDefined()
      expect(screen.getByText('react')).toBeDefined()
      expect(screen.getByText('deprecated')).toBeDefined()
    })

    it('calls onFilterConfigSave on save', () => {
      renderControls()
      fireEvent.click(screen.getByTitle('Configure discovery settings'))

      // Save without changes
      fireEvent.click(screen.getByText('Save'))

      expect(noopFilterSave).toHaveBeenCalledWith(defaultFilterConfig)
    })

    it('toggles include drafts and saves', () => {
      renderControls()
      fireEvent.click(screen.getByTitle('Configure discovery settings'))

      // Find the include drafts toggle (second switch - first is topic context)
      const switches = screen.getAllByRole('switch')
      const draftsToggle = switches[switches.length - 1] as HTMLElement
      fireEvent.click(draftsToggle)

      fireEvent.click(screen.getByText('Save'))

      expect(noopFilterSave).toHaveBeenCalledWith(
        expect.objectContaining({ includeDrafts: true })
      )
    })
  })

  describe('Budget gauge', () => {
    it('renders gauge when complexity is selected and budgetChars provided', () => {
      renderControls({
        complexity: 'moderate',
        budgetChars: 8000,
        totalContentChars: 3000,
      })

      expect(screen.getByText('3.0K / 8.0K chars')).toBeDefined()
    })

    it('shows over budget warning', () => {
      renderControls({
        complexity: 'minor',
        budgetChars: 4000,
        totalContentChars: 6000,
      })

      expect(screen.getByText('over budget')).toBeDefined()
    })

    it('does not render gauge when no complexity selected', () => {
      renderControls({
        complexity: undefined,
        budgetChars: undefined,
      })

      expect(screen.queryByText(/chars$/)).toBeNull()
    })
  })

  describe('Topic context toggle', () => {
    it('calls onToggleDiscover when switch is clicked', () => {
      renderControls({ useDiscover: false })

      const toggle = screen.getByRole('switch')
      fireEvent.click(toggle)

      expect(noopToggle).toHaveBeenCalledWith(true)
    })
  })
})
