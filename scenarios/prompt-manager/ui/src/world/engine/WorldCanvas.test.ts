import { describe, expect, it, vi } from 'vitest'
import { PerspectiveCamera, Raycaster, Vector2 } from 'three'
import type { RootState } from '@react-three/fiber'
import { worldEvents } from './WorldCanvas'

describe('world picking coordinates', () => {
  it.each([false, true])('aims at the visible centre with pointer lock %s', locked => {
    const canvas = document.createElement('canvas')
    vi.spyOn(canvas, 'getBoundingClientRect').mockReturnValue({ left: 280, top: 40, width: 1000, height: 600 } as DOMRect)
    const old = Object.getOwnPropertyDescriptor(document, 'pointerLockElement')
    Object.defineProperty(document, 'pointerLockElement', { configurable: true, value: locked ? canvas : null })
    try {
      const camera = new PerspectiveCamera(60, 1000 / 600, .1, 100)
      const raycaster = new Raycaster()
      const pointer = new Vector2()
      const state = { gl: { domElement: canvas }, camera, raycaster, pointer } as unknown as RootState
      const handler = worldEvents({} as Parameters<typeof worldEvents>[0])
      handler.compute?.(new MouseEvent('click', { clientX: locked ? 50 : 780, clientY: locked ? 50 : 340 }), state)
      expect(pointer.toArray()).toEqual([0, 0])
      expect(raycaster.ray.direction.x).toBeCloseTo(0)
      expect(raycaster.ray.direction.y).toBeCloseTo(0)
      expect(raycaster.ray.direction.z).toBeCloseTo(-1)
    } finally {
      if (old) Object.defineProperty(document, 'pointerLockElement', old)
      else Reflect.deleteProperty(document, 'pointerLockElement')
      vi.restoreAllMocks()
    }
  })
})
