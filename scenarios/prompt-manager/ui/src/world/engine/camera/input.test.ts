import { afterEach, describe, expect, it, vi } from 'vitest'
import CameraControls from 'camera-controls'
import { tuning, type CameraTuning } from '../../config'
import { MouseActionSchema, SingleTouchActionSchema, MultiTouchActionSchema } from '../../config/tuning.schema'
import { applyInputMap, bindCameraHome, bindNavigationKeys, navigationDelta, normalizeWheelUnits } from './input'

function controls() {
  return { mouseButtons: { left: -1, middle: -1, right: -1, wheel: -1 }, touches: { one: -1, two: -1, three: -1 } }
}

describe('declared camera grammar', () => {
  it('assigns every button and finger slot, with semantic mouse/touch parity', () => {
    const c = controls()
    applyInputMap(c, tuning.camera.input)
    const A = CameraControls.ACTION
    expect(c.mouseButtons).toEqual({ left: A.ROTATE, middle: A.TRUCK, right: A.TRUCK, wheel: A.DOLLY })
    // Mouse and touch have different library constants for the same semantic action.
    expect(tuning.camera.input.touch.one).toBe(tuning.camera.input.mouse.left)
    expect(c.touches).toEqual({ one: A.TOUCH_ROTATE, two: A.TOUCH_DOLLY_TRUCK, three: A.TOUCH_TRUCK })
    expect(tuning.camera.input.touch.two.split('-')).toEqual([tuning.camera.input.mouse.wheel, tuning.camera.input.mouse.right])
    expect(tuning.camera.dollyToCursor).toBe(true)
  })
  it('maps every schema value to an actual library action', () => {
    const defined = new Set<number>(Object.values(CameraControls.ACTION))
    const c = controls()
    for (const value of MouseActionSchema.options) {
      applyInputMap(c, { ...tuning.camera.input, mouse: { left: value, middle: value, right: value, wheel: value } })
      for (const result of Object.values(c.mouseButtons)) expect(defined.has(result)).toBe(true)
    }
    for (const value of SingleTouchActionSchema.options) {
      applyInputMap(c, { ...tuning.camera.input, touch: { ...tuning.camera.input.touch, one: value } })
      expect(defined.has(c.touches.one)).toBe(true)
    }
    for (const value of MultiTouchActionSchema.options) {
      applyInputMap(c, { ...tuning.camera.input, touch: { ...tuning.camera.input.touch, two: value, three: value } })
      expect(defined.has(c.touches.two)).toBe(true)
      expect(defined.has(c.touches.three)).toBe(true)
    }
  })
  it('rejects unmapped runtime values atomically, including prototype property names', () => {
    const c = controls()
    const before = structuredClone(c)
    for (const bad of ['teleport', 'toString']) {
      const input = { ...tuning.camera.input, touch: { ...tuning.camera.input.touch, three: bad } } as CameraTuning['input']
      expect(() => applyInputMap(c, input)).toThrow('Unmapped')
      expect(c).toEqual(before)
    }
  })
})

describe('navigation input ownership', () => {
  it('normalizes line and page units once while retaining event identity and modifiers', () => {
    const element = document.createElement('canvas')
    element.style.lineHeight = '20px'
    element.getBoundingClientRect = () => ({ width: 800, height: 600 } as DOMRect)
    document.body.append(element)
    for (const [mode, x, y] of [[0, 0.5, -2], [1, 10, -40], [2, 400, -1200]]) {
      const event = new WheelEvent('wheel', { deltaMode: mode, deltaX: 0.5, deltaY: -2, ctrlKey: true, cancelable: true })
      normalizeWheelUnits(event, element)
      normalizeWheelUnits(event, element)
      expect(event.deltaMode).toBe(0)
      expect(event.deltaX).toBe(x)
      expect(event.deltaY).toBe(y)
      expect(event.ctrlKey).toBe(true)
      expect(event.defaultPrevented).toBe(false)
      event.preventDefault()
      expect(event.defaultPrevented).toBe(true)
    }
  })
  let dispose: (() => void) | undefined
  afterEach(() => { dispose?.(); document.body.replaceChildren() })
  const key = (target: EventTarget = window, options: KeyboardEventInit = {}) => {
    const event = new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true, cancelable: true, ...options })
    target.dispatchEvent(event)
    return event
  }

  it('returns home only for unclaimed Escape, leaving dialog and editor cancellation alone', () => {
    const home = vi.fn()
    dispose = bindCameraHome(home)
    for (const html of ['<input>', '<div contenteditable="true"><span></span></div>', '<div role="dialog"><span></span></div>']) {
      document.body.innerHTML = html
      const target = document.body.querySelector('span') ?? document.body.firstElementChild
      if (!target) throw new Error('Missing keyboard fixture')
      expect(key(target, { key: 'Escape' }).defaultPrevented).toBe(false)
    }
    const cancelEdit = (event: KeyboardEvent) => { event.preventDefault() }
    window.addEventListener('keydown', cancelEdit, true)
    key(window, { key: 'Escape' })
    window.removeEventListener('keydown', cancelEdit, true)
    expect(home).not.toHaveBeenCalled()
    expect(key(window, { key: 'Escape' }).defaultPrevented).toBe(true)
    expect(home).toHaveBeenCalledTimes(1)
    dispose()
    key(window, { key: 'Escape' })
    expect(home).toHaveBeenCalledTimes(1)
  })

  it('releases held input on blur, visibility changes, and cleanup without needing keyup', () => {
    const pressed = new Set<string>()
    dispose = bindNavigationKeys(pressed, () => true)
    for (const release of [() => window.dispatchEvent(new Event('blur')), () => document.dispatchEvent(new Event('visibilitychange')), () => dispose?.()]) {
      expect(key().defaultPrevented).toBe(true)
      expect(pressed.has('ArrowLeft')).toBe(true)
      release()
      expect(pressed.size).toBe(0)
    }
    expect(key().defaultPrevented).toBe(false)
  })

  it('leaves form, editable, modal, and composite widget keys to their owners', () => {
    const pressed = new Set<string>()
    dispose = bindNavigationKeys(pressed, () => true)
    for (const html of ['<input>', '<select></select>', '<button></button>', '<div contenteditable="true"><span></span></div>', '<div role="dialog"><span></span></div>', '<div role="slider"><span></span></div>', '<div role="radiogroup"><span></span></div>']) {
      document.body.innerHTML = html
      key()
      const target = document.body.querySelector('span') ?? document.body.firstElementChild
      if (!target) throw new Error('Missing input fixture')
      target.dispatchEvent(new FocusEvent('focusin', { bubbles: true }))
      expect(pressed.size).toBe(0)
      expect(key(target).defaultPrevented).toBe(false)
      expect(pressed.size).toBe(0)
    }
  })

  it('rejects input while editing disables navigation and respects browser shortcuts', () => {
    const pressed = new Set<string>()
    let enabled = true
    dispose = bindNavigationKeys(pressed, () => enabled)
    key()
    enabled = false
    expect(key().defaultPrevented).toBe(false)
    expect(pressed.size).toBe(0)
    enabled = true
    for (const options of [{ ctrlKey: true }, { metaKey: true }, { altKey: true }]) {
      expect(key(window, options).defaultPrevented).toBe(false)
      expect(pressed.size).toBe(0)
    }
  })

  it('bounds resumed movement while preserving ordinary frame timing', () => {
    expect(navigationDelta(1 / 60)).toBe(1 / 60)
    expect(navigationDelta(.2)).toBe(.2)
    expect(navigationDelta(120)).toBe(0.05)
    for (const invalid of [-1, Infinity, NaN]) expect(navigationDelta(invalid)).toBe(0)
  })
  it('releases movement when Shift changes letter case between press and release', () => {
    const pressed = new Set<string>()
    dispose = bindNavigationKeys(pressed, () => true)
    key(window, { key: 'W', shiftKey: true })
    expect(pressed.has('w')).toBe(true)
    window.dispatchEvent(new KeyboardEvent('keyup', { key: 'w' }))
    expect(pressed.size).toBe(0)
  })
})
