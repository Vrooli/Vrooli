import { act, fireEvent, render, screen } from '@/test-utils/renderWithProviders'
import { describe, expect, it, vi } from 'vitest'
import { CameraToolbar } from './CameraToolbar'
import { createNavigationTelemetry } from './navigationState'
import { DEFAULT_NAVIGATION, INITIAL_NAVIGATION_STATE } from '../config/navigation'

function renderToolbar() {
  const telemetry = createNavigationTelemetry(), command = vi.fn(), onMode = vi.fn(), onPreferences = vi.fn()
  const onHome = vi.fn(), onTool = vi.fn(), onPreset = vi.fn(), onFrame = vi.fn(), onLock = vi.fn(), onZoomTarget = vi.fn()
  render(<CameraToolbar telemetry={telemetry} preferences={DEFAULT_NAVIGATION} onPreferences={onPreferences} tool="orbit" onTool={onTool}
    onMode={onMode} onCommand={command} onPreset={onPreset} onHome={onHome} onFrame={onFrame} canFrame={false} onLock={onLock} zoomTarget="cursor" onZoomTarget={onZoomTarget} />)
  return { telemetry, command, onMode, onPreferences, onHome }
}

describe('camera navigation HUD', () => {
  it('keeps the camera panel behind a labeled camera control until it is opened', () => {
    renderToolbar()
    expect(screen.queryByRole('region', { name: 'Camera navigation' })).not.toBeInTheDocument()
    const toggle = screen.getByRole('button', { name: 'Camera controls' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(toggle)
    expect(screen.getByRole('region', { name: 'Camera navigation' })).toBeVisible()
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
  })

  it('exposes accessible exploration actions and changes to walking controls with live telemetry', () => {
    const { telemetry, command, onMode } = renderToolbar()
    fireEvent.click(screen.getByRole('button', { name: 'Camera controls' }))
    expect(screen.getByRole('button', { name: 'Frame selection' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Pan left' }))
    expect(command).toHaveBeenCalledWith('left')
    fireEvent.change(screen.getByLabelText('Camera mode'), { target: { value: 'first-person' } })
    expect(onMode).toHaveBeenCalledWith('first-person')
    act(() => telemetry.publish({ ...INITIAL_NAVIGATION_STATE, mode: 'first-person', heading: 90, blocked: true }))
    expect(screen.getByRole('button', { name: 'Zoom in' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Step forward' })).toBeVisible()
    expect(screen.getByRole('status')).toHaveTextContent('Movement blocked')
    fireEvent.click(screen.getByRole('button', { name: 'Return to Explore' }))
    expect(onMode).toHaveBeenLastCalledWith('explore')
  })

  it('restores the home view from the camera panel', () => {
    const { onHome } = renderToolbar()
    fireEvent.click(screen.getByRole('button', { name: 'Camera controls' }))
    fireEvent.click(screen.getByTestId('world-hud-home'))
    expect(onHome).toHaveBeenCalled()
  })
})
