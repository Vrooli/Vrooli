import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { WorldSettingsContent } from './WorldSettingsContent'
import { usePerformanceStore } from '@/stores/performanceStore'
import { selectors } from '@/constants/selectors'

const worldScaleState = vi.hoisted(() => ({
  agent: 1,
  furniture: 1,
  decoration: 1,
  overlay: 1,
  setScale: vi.fn(),
  resetAll: vi.fn(),
}))

vi.mock('@/stores/worldScaleStore', () => ({
  useWorldScaleStore: vi.fn((selector: (state: typeof worldScaleState) => unknown) =>
    selector(worldScaleState)
  ),
}))

describe('WorldSettingsContent', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    usePerformanceStore.getState().setConfig({
      showOverlay: false,
      showTraceCharts: true,
      maxFps: 'auto',
    })
  })

  it('toggles FPS overlay from world settings', () => {
    render(<WorldSettingsContent />)

    const overlayToggle = screen.getByTestId(selectors.settings.fpsOverlayToggle)
    expect(usePerformanceStore.getState().config.showOverlay).toBe(false)

    fireEvent.click(overlayToggle)
    expect(usePerformanceStore.getState().config.showOverlay).toBe(true)

    fireEvent.click(overlayToggle)
    expect(usePerformanceStore.getState().config.showOverlay).toBe(false)
  })

  it('disables trace toggle when overlay is hidden and enables it when shown', () => {
    render(<WorldSettingsContent />)

    const overlayToggle = screen.getByTestId(selectors.settings.fpsOverlayToggle)
    const traceToggle = screen.getByTestId(selectors.settings.fpsTraceToggle)
    expect(traceToggle).toBeDisabled()

    fireEvent.click(overlayToggle)
    expect(traceToggle).not.toBeDisabled()

    fireEvent.click(traceToggle)
    expect(usePerformanceStore.getState().config.showTraceCharts).toBe(false)
  })

  it('updates max FPS from world settings select', () => {
    render(<WorldSettingsContent />)

    const maxFpsSelect = screen.getByTestId(selectors.settings.fpsMaxSelect)
    fireEvent.change(maxFpsSelect, { target: { value: '45' } })
    expect(usePerformanceStore.getState().config.maxFps).toBe(45)

    fireEvent.change(maxFpsSelect, { target: { value: 'auto' } })
    expect(usePerformanceStore.getState().config.maxFps).toBe('auto')
  })
})
