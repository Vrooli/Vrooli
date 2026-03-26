/**
 * Tests for DiscoverControls component.
 *
 * Covers: complexity buttons from budgetConfig, gear button toggle,
 * budget editor save/cancel, budget gauge rendering.
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { DiscoverControls } from './DiscoverControls'
import type { BudgetConfig } from '@/lib/schemas'

const defaultConfig: BudgetConfig = {
  minor: 4000,
  moderate: 8000,
  major: 12000,
  architectural: 18000,
}

const noopToggle = vi.fn()
const noopComplexity = vi.fn()
const noopSave = vi.fn()

function renderControls(overrides: Record<string, unknown> = {}) {
  const props = {
    useDiscover: true,
    onToggleDiscover: noopToggle,
    complexity: undefined as string | undefined,
    onComplexityChange: noopComplexity,
    budgetConfig: defaultConfig,
    onBudgetConfigSave: noopSave,
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

  describe('Gear button / budget editor', () => {
    it('toggles budget editor visibility on gear click', () => {
      renderControls()

      // Editor should not be visible initially
      expect(screen.queryByText('Save')).toBeNull()

      // Click gear button
      const gearButton = screen.getByTitle('Configure budget tiers')
      fireEvent.click(gearButton)

      // Editor should now be visible
      expect(screen.getByText('Save')).toBeDefined()
      expect(screen.getByText('Cancel')).toBeDefined()
    })

    it('calls onBudgetConfigSave with edited values on save', () => {
      renderControls()

      // Open editor
      fireEvent.click(screen.getByTitle('Configure budget tiers'))

      // Find inputs and change a value
      const inputs = screen.getAllByRole('spinbutton')
      expect(inputs).toHaveLength(4)

      // Change minor to 5000
      fireEvent.change(inputs[0]!, { target: { value: '5000' } })

      // Save
      fireEvent.click(screen.getByText('Save'))

      expect(noopSave).toHaveBeenCalledWith({
        minor: 5000,
        moderate: 8000,
        major: 12000,
        architectural: 18000,
      })
    })

    it('discards changes on cancel', () => {
      renderControls()

      // Open editor
      fireEvent.click(screen.getByTitle('Configure budget tiers'))

      // Change a value
      const inputs = screen.getAllByRole('spinbutton')
      fireEvent.change(inputs[0]!, { target: { value: '999' } })

      // Cancel
      fireEvent.click(screen.getByText('Cancel'))

      // Editor should be closed
      expect(screen.queryByText('Save')).toBeNull()

      // onBudgetConfigSave should NOT have been called
      expect(noopSave).not.toHaveBeenCalled()
    })

    it('disables save when values are not ascending', () => {
      renderControls()

      fireEvent.click(screen.getByTitle('Configure budget tiers'))

      // Set minor higher than moderate (invalid)
      const inputs = screen.getAllByRole('spinbutton')
      fireEvent.change(inputs[0]!, { target: { value: '9000' } })

      // Save button should be disabled
      const saveButton = screen.getByText('Save')
      expect(saveButton.hasAttribute('disabled')).toBe(true)

      // Validation message should show
      expect(screen.getByText('Values must be ascending')).toBeDefined()
    })
  })

  describe('Budget gauge', () => {
    it('renders gauge when complexity is selected and budgetChars provided', () => {
      renderControls({
        complexity: 'moderate',
        budgetChars: 8000,
        budgetStatus: 'under',
        totalContentChars: 3000,
      })

      expect(screen.getByText('3.0K / 8.0K chars')).toBeDefined()
    })

    it('shows over budget warning', () => {
      renderControls({
        complexity: 'minor',
        budgetChars: 4000,
        budgetStatus: 'over',
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
