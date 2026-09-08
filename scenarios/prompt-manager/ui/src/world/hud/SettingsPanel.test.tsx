import { useState } from 'react'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { WorldSettingsContent } from './SettingsPanel'
import { WorldClock } from '../config/clock'
import { tuning } from '../config'

function mount(clock?: WorldClock) {
  const changed = vi.fn()
  function Harness() {
    const [seed, setSeed] = useState(1)
    return <WorldSettingsContent worldClock={clock} levers={clock ? { tuning, override: {}, onChange: () => {}, onReset: () => {} } : undefined} seed={seed} onSeedChange={next => { changed(next); setSeed(next) }} weather="auto" onWeatherChange={() => {}}
      sceneId="park" onSceneChange={() => {}} quality={{ auto: false, profileId: 'high' }}
      onPickProfile={() => {}} onAutoChange={() => {}} periodMode={{ kind: 'clock' }}
      onPeriodModeChange={() => {}} showDiagnostics={false} onShowDiagnosticsChange={() => {}}
      onCameraHome={() => {}} zoomTarget="cursor" onZoomTargetChange={() => {}} />
  }
  render(<Harness />)
  return { changed, field: screen.getByRole('spinbutton', { name: 'World seed' }) }
}

describe('world seed settings', () => {
  it('plays from the selected instant, freezes there, and explicitly returns to wall time', () => {
    let wall = 1000000
    const clock = new WorldClock(() => wall, 'UTC'); clock.fix(10000)
    mount(clock)
    fireEvent.click(screen.getByText('World workbench', { exact: true }))
    fireEvent.click(screen.getByRole('button', { name: 'Play from here' }))
    expect(screen.getByText('Presentation time: Playing from selected time')).toBeInTheDocument()
    expect(clock.snapshot().utcMilliseconds).toBe(10000)
    wall += 2000
    expect(clock.snapshot().utcMilliseconds).toBe(12000)
    fireEvent.click(screen.getByRole('button', { name: 'Freeze time' }))
    wall += 10000
    expect(clock.snapshot().utcMilliseconds).toBe(12000)
    expect(screen.getByText('Presentation time: Frozen')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Resume live time' }))
    expect(clock.snapshot().utcMilliseconds).toBe(wall)
    expect(screen.getByText('Presentation time: Live')).toBeInTheDocument()
  })
  it('applies explicitly, retains keyboard focus, and avoids duplicate generation requests', () => {
    const { changed, field } = mount()
    field.focus()
    fireEvent.change(field, { target: { value: '42' } })
    fireEvent.blur(field)
    expect(changed).not.toHaveBeenCalled()
    field.focus()
    fireEvent.keyDown(field, { key: 'Enter' })
    expect(changed).toHaveBeenCalledWith(42)
    expect(field).toHaveFocus()
    expect(field).toHaveValue(42)
    fireEvent.click(screen.getByRole('button', { name: 'Apply seed' }))
    expect(changed).toHaveBeenCalledTimes(1)
  })

  it.each(['', '-1', '1.5', '4294967296'])('rejects invalid seed %j without applying it', raw => {
    const { changed, field } = mount()
    fireEvent.change(field, { target: { value: raw } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply seed' }))
    expect(changed).not.toHaveBeenCalled()
    expect(field).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('alert')).toHaveTextContent('Enter a whole number')
    fireEvent.change(field, { target: { value: '4294967295' } })
    fireEvent.keyDown(field, { key: 'Enter' })
    expect(changed).toHaveBeenCalledWith(4294967295)
    expect(field).toHaveAttribute('aria-invalid', 'false')
  })
})
