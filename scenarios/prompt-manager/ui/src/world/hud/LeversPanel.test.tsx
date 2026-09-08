import { useState } from 'react'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { tuning, withTuningOverride, type TuningOverride } from '../config'
import { LeversPanel } from './LeversPanel'

function mount() {
  const changed = vi.fn()
  function Harness() {
    const [override, setOverride] = useState<TuningOverride>({})
    return <LeversPanel tuning={withTuningOverride(override, tuning)} override={override}
      onChange={next => { changed(next); setOverride(next) }} onReset={() => setOverride({})} />
  }
  render(<Harness />)
  fireEvent.click(screen.getByRole('button', { name: 'layout' }))
  return { changed, field: screen.getByRole('spinbutton', { name: 'deskPitch' }) }
}

describe('session workbench controls', () => {
  it('keeps focus while applying a value, avoids duplicate blur commits and displays reset defaults', () => {
    const { field, changed } = mount()
    field.focus()
    fireEvent.change(field, { target: { value: '3' } })
    fireEvent.keyDown(field, { key: 'Enter' })
    expect(field).toHaveFocus()
    expect(field).toHaveValue(3)
    expect(changed).toHaveBeenCalledTimes(1)
    fireEvent.blur(field)
    expect(changed).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: 'Reset' }))
    expect(field).toHaveValue(tuning.layout.deskPitch)
  })

  it('rejects blank input instead of applying zero and clears draft errors on reset', () => {
    const { field, changed } = mount()
    fireEvent.change(field, { target: { value: '' } })
    fireEvent.blur(field)
    expect(changed).not.toHaveBeenCalled()
    expect(field).toHaveAttribute('aria-invalid', 'true')
    expect(field).toHaveAccessibleDescription('Enter a finite number.')
    fireEvent.click(screen.getByRole('button', { name: 'Reset' }))
    expect(field).toHaveValue(tuning.layout.deskPitch)
    expect(field).toHaveAttribute('aria-invalid', 'false')
    expect(screen.queryByText('Enter a finite number.')).not.toBeInTheDocument()
  })
  it('edits authoritative AO choices and hides the derived compatibility flag', () => {
    const { changed } = mount()
    fireEvent.click(screen.getByRole('button', { name: 'quality' }))
    expect(screen.queryByRole('checkbox', { name: 'profiles.high.ao' })).not.toBeInTheDocument()
    const quality = screen.getByRole('combobox', { name: 'profiles.high.aoQuality' })
    expect([...quality.querySelectorAll('option')].map(option => option.value)).toEqual(['off', 'low', 'medium'])
    fireEvent.change(quality, { target: { value: 'off' } })
    expect(quality).toHaveValue('off')
    expect(changed).toHaveBeenLastCalledWith({ quality: { profiles: { high: { aoQuality: 'off' } } } })
    fireEvent.change(quality, { target: { value: 'medium' } })
    expect(quality).toHaveValue('medium')
    expect(changed).toHaveBeenCalledTimes(2)
  })

})
