import { describe, expect, it, vi } from 'vitest'
import { renderHook } from '@/test-utils/renderWithProviders'
import { DEFAULT_NAVIGATION } from '../../../config/navigation'
import type { Walker } from '../walking'
import { useWalkingInput } from './useWalkingInput'

function setup() {
  const canvas = document.createElement('canvas')
  const jump = vi.fn(), look = vi.fn()
  const keys = { current: new Set(['w']) }
  const onExit = vi.fn(), onLock = vi.fn()
  const hook = renderHook(() => useWalkingInput({ mode: 'first-person', canvas,
    walker: { current: { jump, look } as unknown as Walker }, keys, navigation: DEFAULT_NAVIGATION,
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
    expect(sceneClick).toHaveBeenCalledOnce()
    view.unmount()
    view.canvas.dispatchEvent(new MouseEvent('click'))
    expect(sceneClick).toHaveBeenCalledTimes(2)
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


it('passes a click to scene picking but suppresses a drag release', () => {
  const view = setup()
  view.canvas.setPointerCapture = vi.fn()
  const selected = vi.fn()
  view.canvas.addEventListener('click', selected)
  const pointer = (type: string, movementX = 0) => {
    const event = new Event(type, { bubbles: true })
    Object.assign(event, { pointerId: 1, button: 0, movementX, movementY: 0 })
    view.canvas.dispatchEvent(event)
  }
  pointer('pointerdown')
  pointer('pointerup')
  view.canvas.dispatchEvent(new MouseEvent('click'))
  expect(selected).toHaveBeenCalledOnce()
  pointer('pointerdown')
  pointer('pointermove', 20)
  pointer('pointerup')
  view.canvas.dispatchEvent(new MouseEvent('click'))
  expect(selected).toHaveBeenCalledOnce()
  view.unmount()
})


it('uses pointer lock without requesting incompatible pointer capture', () => {
  const view = setup()
  const capture = vi.fn(() => { throw new DOMException('InvalidStateError') })
  view.canvas.setPointerCapture = capture
  const old = Object.getOwnPropertyDescriptor(document, 'pointerLockElement')
  Object.defineProperty(document, 'pointerLockElement', { configurable: true, value: view.canvas })
  try {
    const down = new Event('pointerdown', { bubbles: true })
    Object.assign(down, { pointerId: 1, button: 0 })
    view.canvas.dispatchEvent(down)
    expect(capture).not.toHaveBeenCalled()
    const selected = vi.fn()
    view.canvas.addEventListener('click', selected)
    view.canvas.dispatchEvent(new MouseEvent('click'))
    expect(selected).toHaveBeenCalledOnce()
  } finally {
    if (old) Object.defineProperty(document, 'pointerLockElement', old)
    else Reflect.deleteProperty(document, 'pointerLockElement')
    view.unmount()
  }
})
