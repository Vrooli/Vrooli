import { act, fireEvent, render, screen } from '@/test-utils/renderWithProviders'
import { describe, expect, it, vi } from 'vitest'
import { CameraToolbar } from './CameraToolbar'
import { createNavigationTelemetry } from './navigationState'
import { DEFAULT_NAVIGATION, INITIAL_NAVIGATION_STATE } from '../config/navigation'

describe('camera navigation HUD', () => {
  it('exposes accessible exploration actions and changes to walking controls with live telemetry', () => {
    const telemetry = createNavigationTelemetry(), command = vi.fn(), onMode = vi.fn(), onPreferences = vi.fn()
    render(<CameraToolbar telemetry={telemetry} preferences={DEFAULT_NAVIGATION} onPreferences={onPreferences} tool="orbit" onTool={vi.fn()}
      onMode={onMode} onCommand={command} onPreset={vi.fn()} onHome={vi.fn()} onFrame={vi.fn()} canFrame={false} onLock={vi.fn()} zoomTarget="cursor" onZoomTarget={vi.fn()} />)
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
})
