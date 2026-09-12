import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@/test-utils/renderWithProviders'
import { ViewOverlay } from './ViewOverlay'
import { useIsMobile } from '@/hooks/useMediaQuery'

vi.mock('./StatsBar', () => ({
  StatsBar: () => <div data-testid="view-overlay-stats" />,
}))

vi.mock('@/hooks/useMediaQuery', () => ({
  useIsMobile: vi.fn(() => false),
}))

describe('ViewOverlay', () => {
  beforeEach(() => {
    vi.mocked(useIsMobile).mockReturnValue(false)
  })

  it('opens settings and help as independent floating panels on desktop', () => {
    const { container } = render(
      <ViewOverlay
        homeView="graph"
        leftPanelContent={<div>Left Panel</div>}
        settingsContent={<div>Settings Body</div>}
        settingsTitle="Graph Settings"
        helpContent={<div>Help Body</div>}
        helpTitle="Graph Help"
      />,
    )

    expect(screen.getByTestId('view-overlay-stats')).toBeInTheDocument()

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
    expect(screen.queryByTestId('view-overlay-mobile-stats-button')).not.toBeInTheDocument()
  })

  it('shows compact mobile controls and opens/closes stats + queries sheets', () => {
    vi.mocked(useIsMobile).mockReturnValue(true)
    const onOpenMobileSidebar = vi.fn()

    render(
      <ViewOverlay
        onOpenMobileSidebar={onOpenMobileSidebar}
        leftPanelContent={<div>Query Body</div>}
        settingsContent={<div>Settings Body</div>}
        helpContent={<div>Help Body</div>}
      />,
    )

    fireEvent.click(screen.getByTestId('view-overlay-mobile-sidebar-button'))
    expect(onOpenMobileSidebar).toHaveBeenCalledTimes(1)

    expect(screen.getByTestId('view-overlay-mobile-stats-button')).toBeInTheDocument()
    expect(screen.getByTestId('view-overlay-mobile-left-panel-button')).toBeInTheDocument()
    expect(screen.queryByTestId('view-overlay-stats')).not.toBeInTheDocument()

    fireEvent.click(screen.getByTestId('view-overlay-mobile-stats-button'))
    expect(screen.getByTestId('view-overlay-mobile-panel-sheet')).toBeInTheDocument()
    expect(screen.getByText('Stats')).toBeInTheDocument()
    expect(screen.getByTestId('view-overlay-stats')).toBeInTheDocument()

    fireEvent.click(screen.getByTestId('view-overlay-mobile-panel-backdrop'))
    expect(screen.queryByTestId('view-overlay-mobile-panel-sheet')).not.toBeInTheDocument()

    fireEvent.click(screen.getByTestId('view-overlay-mobile-left-panel-button'))
    expect(screen.getByTestId('view-overlay-mobile-panel-sheet')).toBeInTheDocument()
    expect(screen.getByText('Queries')).toBeInTheDocument()
    expect(screen.getByText('Query Body')).toBeInTheDocument()
  })
})
