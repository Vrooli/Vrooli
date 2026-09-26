import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@/test-utils/renderWithProviders'
import { SettingsOverlay } from './SettingsOverlay'
import { WORLD_SETTINGS_SECTIONS } from './settingsGroups'
import { useIsMobile } from '@/hooks/useMediaQuery'

vi.mock('@/hooks/useMediaQuery', () => ({
  useIsMobile: vi.fn(() => false),
}))

function renderOverlay() {
  return render(
    <SettingsOverlay
      title="World Settings"
      sections={WORLD_SETTINGS_SECTIONS}
      renderSection={(id) => <div data-testid={`section-${id}`}>{id}</div>}
    />,
  )
}

describe('SettingsOverlay', () => {
  beforeEach(() => {
    vi.mocked(useIsMobile).mockReturnValue(false)
  })

  it('uses side navigation on desktop and renders only the active group', () => {
    renderOverlay()
    expect(screen.getByTestId('world-settings-overlay-nav-world')).toHaveAttribute('aria-current', 'page')
    expect(screen.getByTestId('section-world')).toBeInTheDocument()
    expect(screen.queryByTestId('section-appearance')).not.toBeInTheDocument()

    fireEvent.click(screen.getByTestId('world-settings-overlay-nav-advanced'))
    expect(screen.getByTestId('section-advanced')).toBeInTheDocument()
    expect(screen.queryByTestId('section-world')).not.toBeInTheDocument()
    expect(screen.getByTestId('world-settings-overlay')).toHaveAttribute('data-active-section', 'advanced')
  })

  it('uses compact tabs on narrow screens with the same selection state', () => {
    vi.mocked(useIsMobile).mockReturnValue(true)
    renderOverlay()
    expect(screen.queryByTestId('world-settings-overlay-nav-world')).not.toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'World' })).toHaveAttribute('aria-selected', 'true')

    fireEvent.click(screen.getByRole('tab', { name: 'Appearance' }))
    expect(screen.getByTestId('section-appearance')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Appearance' })).toHaveAttribute('aria-selected', 'true')
  })

  it('surfaces a failed save as an alert with a recovery retry', () => {
    const onRetry = vi.fn()
    render(
      <SettingsOverlay
        title="World Settings"
        sections={WORLD_SETTINGS_SECTIONS}
        renderSection={(id) => <div data-testid={`section-${id}`}>{id}</div>}
        saveState={{ status: 'unavailable', message: null, at: null, onRetry }}
      />,
    )
    const status = screen.getByTestId('world-settings-overlay-save-status')
    expect(status).toHaveAttribute('role', 'alert')
    expect(status).toHaveTextContent(/unavailable/i)
    fireEvent.click(screen.getByTestId('world-settings-overlay-save-retry'))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('reports saving and saved without offering a retry', () => {
    const { rerender } = render(
      <SettingsOverlay
        title="World Settings"
        sections={WORLD_SETTINGS_SECTIONS}
        renderSection={(id) => <div data-testid={`section-${id}`}>{id}</div>}
        saveState={{ status: 'saving', message: null, at: null, onRetry: vi.fn() }}
      />,
    )
    expect(screen.getByTestId('world-settings-overlay-save-status')).toHaveAttribute('role', 'status')
    expect(screen.queryByTestId('world-settings-overlay-save-retry')).not.toBeInTheDocument()

    rerender(
      <SettingsOverlay
        title="World Settings"
        sections={WORLD_SETTINGS_SECTIONS}
        renderSection={(id) => <div data-testid={`section-${id}`}>{id}</div>}
        saveState={{ status: 'saved', message: null, at: null }}
      />,
    )
    expect(screen.getByTestId('world-settings-overlay-save-status')).toHaveTextContent('Changes saved.')
  })
})
