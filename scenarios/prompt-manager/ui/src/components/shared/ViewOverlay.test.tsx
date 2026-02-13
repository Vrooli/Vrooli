import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { ViewOverlay } from './ViewOverlay'
import { useSelectionStore } from '@/stores/selectionStore'

vi.mock('./StatsBar', () => ({
  StatsBar: () => <div data-testid="view-overlay-stats" />,
}))

describe('ViewOverlay', () => {
  it('opens settings and help as independent floating panels', () => {
    useSelectionStore.setState({ graphViewActive: true })

    const { container } = render(
      <ViewOverlay
        leftPanelContent={<div>Left Panel</div>}
        settingsContent={<div>Settings Body</div>}
        settingsTitle="Graph Settings"
        helpContent={<div>Help Body</div>}
        helpTitle="Graph Help"
      />,
    )

    fireEvent.click(screen.getByTestId('view-overlay-settings-button'))
    expect(screen.getByText('Graph Settings')).toBeInTheDocument()
    expect(screen.getByText('Settings Body')).toBeInTheDocument()

    fireEvent.click(screen.getByTestId('view-overlay-help-button'))
    expect(screen.getByText('Graph Help')).toBeInTheDocument()
    expect(screen.getByText('Help Body')).toBeInTheDocument()

    // Both can stay open concurrently.
    expect(screen.getAllByRole('dialog')).toHaveLength(2)

    // New behavior: no blocking modal backdrop.
    expect(container.querySelector('.bg-black\\/50')).toBeNull()
  })
})
