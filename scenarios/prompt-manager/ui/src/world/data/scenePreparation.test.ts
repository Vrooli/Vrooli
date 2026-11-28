import { describe, expect, it, vi } from 'vitest'
import { ScenePreparationCoordinator } from './scenePreparation'

describe('scene preparation ownership', () => {
  it('records concurrent spans, closes pending spans on cancel and isolates returned snapshots', async () => {
    const coordinator = new ScenePreparationCoordinator()
    let finish: () => void = () => { throw new Error('not started') }
    const request = coordinator.start(async (_signal, stage) => {
      await Promise.all([
        stage('assets', () => new Promise<void>(resolve => { finish = resolve })),
        stage('generation', () => Promise.resolve()),
      ])
    }, vi.fn())
    await Promise.resolve()
    await Promise.resolve()
    const before = coordinator.read()
    expect(before.stages.map(stage => stage.stage)).toEqual(['assets', 'generation'])
    request.cancel()
    expect(coordinator.read().stages.find(stage => stage.stage === 'assets')?.outcome).toBe('cancelled')
    expect(before.stages[0]?.outcome).toBe('running')
    finish()
    expect(await request.promise).toBe(false)
    expect(coordinator.read().stages.every(stage => stage.completedAt !== null)).toBe(true)
  })
  it('commits only the newest epoch when an obsolete provider ignores cancellation', async () => {
    const coordinator = new ScenePreparationCoordinator()
    let finishOld: (value: string) => void = () => { throw new Error('not started') }
    const commit = vi.fn()
    const old = coordinator.start(() => new Promise<string>(resolve => { finishOld = resolve }), commit)
    await Promise.resolve()
    const next = coordinator.start(() => Promise.resolve('office'), commit)
    expect(next.epoch).toBeGreaterThan(old.epoch)
    expect(await next.promise).toBe(true)
    finishOld('park')
    expect(await old.promise).toBe(false)
    expect(commit).toHaveBeenCalledOnce()
    expect(commit).toHaveBeenCalledWith('office')
    expect(coordinator.read()).toMatchObject({ epoch: next.epoch, state: 'committed' })
  })

  it('preserves the prior commit on failure and allows retry', async () => {
    const coordinator = new ScenePreparationCoordinator()
    let visible = 'park'
    const commit = (value: string) => { visible = value }
    await expect(coordinator.start<string>(() => Promise.reject(new Error('decode failed')), commit).promise).rejects.toThrow('decode failed')
    expect(visible).toBe('park')
    expect(coordinator.read().state).toBe('failed')
    await coordinator.start(() => Promise.resolve('office'), commit).promise
    expect(visible).toBe('office')
  })

  it('cancels queued work before it starts and makes cancellation idempotent', async () => {
    const coordinator = new ScenePreparationCoordinator()
    const prepare = vi.fn(() => Promise.resolve('park'))
    const commit = vi.fn()
    const request = coordinator.start(prepare, commit)
    request.cancel()
    request.cancel()
    expect(await request.promise).toBe(false)
    expect(prepare).not.toHaveBeenCalled()
    expect(commit).not.toHaveBeenCalled()
    expect(coordinator.read().state).toBe('cancelled')
  })
})
