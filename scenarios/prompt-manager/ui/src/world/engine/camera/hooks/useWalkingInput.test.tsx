import { describe, expect, it, vi } from 'vitest'
import { renderHook } from '@/test-utils/renderWithProviders'
import { DEFAULT_NAVIGATION } from '../../../config/navigation'
import type { Walker } from '../walking'
import { useWalkingInput } from './useWalkingInput'

function setup() {
  const canvas = document.createElement('canvas')
  const jump = vi.fn()
  const keys = { current: new Set(['w']) }
  const onExit = vi.fn(), onLock = vi.fn()
  const hook = renderHook(() => useWalkingInput({ mode: 'first-person', canvas,
    walker: { current: { jump } as unknown as Walker }, keys, navigation: DEFAULT_NAVIGATION,
    onExit, onLock, onError: vi.fn(), invalidate: vi.fn() }))
  return { ...hook, jump, canvas, keys, onExit, onLock }
}

describe('walking input lifetime', () => {
  it('clears held movement on blur and removes Escape handling on disposal', () => {
    const view = setup()
    window.dispatchEvent(new Event('blur'))
    expect(view.keys.current.size).toBe(0)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', cancelable: true }))
    expect(view.onExit).toHaveBeenCalledOnce()
    view.unmount()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', cancelable: true }))
    expect(view.onExit).toHaveBeenCalledOnce()
    expect(view.onLock).toHaveBeenLastCalledWith(false)
  })

  it('allows text fields to own Escape and restores scene click handling after exit', () => {
    const view = setup()
    const input = document.createElement('input')
    document.body.append(input)
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    expect(view.onExit).not.toHaveBeenCalled()
    const sceneClick = vi.fn()
    view.canvas.addEventListener('click', sceneClick)
    view.canvas.dispatchEvent(new MouseEvent('click'))
    expect(sceneClick).not.toHaveBeenCalled()
    view.unmount()
    view.canvas.dispatchEvent(new MouseEvent('click'))
    expect(sceneClick).toHaveBeenCalledOnce()
    input.remove()
  })
})


it('jumps on focused Space once, ignores repeats and leaves unfocused Space alone', () => {
  const view = setup()
  view.canvas.tabIndex = 0
  document.body.append(view.canvas)
  view.canvas.focus()
  const press = new KeyboardEvent('keydown', { code: 'Space', key: ' ', cancelable: true, bubbles: true })
  view.canvas.dispatchEvent(press)
  expect(press.defaultPrevented).toBe(true)
  view.canvas.dispatchEvent(new KeyboardEvent('keydown', { code: 'Space', repeat: true, bubbles: true }))
  expect(view.jump).toHaveBeenCalledOnce()
  view.canvas.blur()
  window.dispatchEvent(new KeyboardEvent('keydown', { code: 'Space' }))
  expect(view.jump).toHaveBeenCalledOnce()
  view.unmount()
  view.canvas.remove()
})
