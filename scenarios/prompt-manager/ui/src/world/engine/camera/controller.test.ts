import { describe, expect, it, vi } from 'vitest'
import { CameraController } from './controller'

describe('camera command ownership', () => {
  it('records ordered causes and keeps only the latest 32 immutable snapshots', () => {
    let now = 100
    const changed = vi.fn()
    const controller = new CameraController(vi.fn(), changed, () => now++)
    controller.begin('intro')
    controller.navigate('reduced-motion')
    controller.edit(true)
    controller.edit(false)
    expect(controller.history().map(event => event.cause)).toEqual(['begin', 'reduced-motion', 'edit-start', 'edit-end'])
    const early = controller.history()
    expect(early.map(event => event.at)).toEqual([100, 101, 102, 103])
    for (let i = 0; i < 40; i++) controller.navigate()
    const history = controller.history()
    expect(history).toHaveLength(32)
    expect(history[0]?.sequence).toBe(13)
    expect(history[31]?.sequence).toBe(44)
    expect(early).toHaveLength(4)
    if (history[0]) history[0].owner = 'editing'
    expect(controller.history()[0]?.owner).toBe('explore')
    const published = changed.mock.lastCall?.[1]
    published?.splice(0)
    expect(controller.history()).toHaveLength(32)
  })
  it('rejects stale intro completion after user navigation and a subsequent focus command', () => {
    const stop = vi.fn()
    const removeListener = vi.fn()
    const controller = new CameraController(stop)
    const intro = controller.begin('intro')
    if (intro === null) throw new Error('Intro rejected')
    controller.onCancel(intro, removeListener)
    controller.navigate()
    expect(removeListener).toHaveBeenCalledTimes(1)
    expect(stop).toHaveBeenCalledTimes(2)
    const focus = controller.begin('focus')
    expect(controller.complete(intro)).toBe(false)
    expect(controller.read()).toMatchObject({ owner: 'focus', moving: true, command: focus })
  })

  it('keeps follow while navigation cancels framing; home explicitly detaches it', () => {
    const controller = new CameraController(vi.fn())
    controller.begin('focus')
    controller.follow(true)
    controller.navigate()
    expect(controller.read()).toMatchObject({ owner: 'follow', moving: false })
    controller.begin('overview')
    controller.navigate()
    expect(controller.read()).toMatchObject({ owner: 'explore', moving: false })
  })

  it('suspends commands during editing and resumes follow without replaying the cancelled transition', () => {
    const controller = new CameraController(vi.fn())
    const command = controller.begin('focus')
    controller.follow(true)
    controller.edit(true)
    controller.navigate()
    expect(controller.begin('overview')).toBeNull()
    expect(controller.read()).toMatchObject({ owner: 'editing', moving: false })
    controller.edit(false)
    expect(controller.read()).toMatchObject({ owner: 'follow', moving: false })
    expect(controller.complete(command ?? -1)).toBe(false)
  })

  it('completes commands once and removes listeners on replacement and disposal', () => {
    const controller = new CameraController(vi.fn())
    const cleanups = [vi.fn(), vi.fn(), vi.fn()]
    for (const cleanup of cleanups) {
      const command = controller.begin('intro')
      if (command === null) throw new Error('Intro rejected')
      controller.onCancel(command, cleanup)
    }
    const command = controller.read().command
    expect(controller.complete(command)).toBe(true)
    expect(controller.complete(command)).toBe(false)
    expect(controller.read()).toMatchObject({ owner: 'overview', moving: false })
    controller.dispose()
    for (const cleanup of cleanups) expect(cleanup).toHaveBeenCalledTimes(1)
  })
})
