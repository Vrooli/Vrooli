import { describe, expect, it, vi } from 'vitest'
import { tuning } from '../../config'
import { makeWorldInput } from '../../sim/__tests__/fixtures'
import { generateWorld } from '../../sim/world'
import { createGenerationService, generationTransfers, validateGenerationJob, type GenerationJob, type GenerationReply, type GenerationRequest } from '../generation'
import { createGenerationClient, GenerationClient, type GenerationWorker } from '../generationClient'

function fakeWorker(): GenerationWorker {
  return { postMessage: vi.fn(), terminate: vi.fn(), onmessage: null, onerror: null }
}
const job = () => ({ input: makeWorldInput({ teams: 1, agents: 2 }), tuning })

describe('generation worker protocol', () => {
  it('uses an observable cooperative fallback when worker construction fails', async () => {
    vi.stubGlobal('Worker', function () { throw new Error('workers disabled') })
    const client = createGenerationClient()
    let heartbeats = 0
    const timer = setInterval(() => heartbeats++, 0)
    try {
      expect(client.mode).toBe('fallback')
      expect(client.fallbackReason).toContain('workers disabled')
      const request = job()
      const world = await client.generate(request)
      expect(heartbeats).toBeGreaterThan(1)
      expect(world).toEqual(generateWorld(request.input, request.tuning))
      expect(world.terrain.height.byteLength).toBeGreaterThan(0)
    } finally { clearInterval(timer); client.dispose(); vi.unstubAllGlobals() }
  })

  it('cancels fallback at a generation checkpoint and can accept another request', async () => {
    vi.stubGlobal('Worker', undefined)
    const client = createGenerationClient()
    const controller = new AbortController()
    try {
      await expect(client.generate(job(), {
        signal: controller.signal,
        onProgress: progress => { if (progress.stage === 'terrain') controller.abort() },
      })).rejects.toMatchObject({ name: 'AbortError' })
      const world = await client.generate(job())
      expect(world.scene).toBe('park')
    } finally { client.dispose(); vi.unstubAllGlobals() }
  })

  it.each([
    ['excessive octaves', (job: GenerationJob) => { job.tuning.terrain.octaves = 1_000_000 }],
    ['excessive candidate count', (job: GenerationJob) => { job.tuning.layout.siteCandidates = 1_000_000 }],
    ['oversized terrain grid', (job: GenerationJob) => { job.tuning.terrain.radius = 500; job.tuning.terrain.cellSize = 0.25 }],
    ['oversized navigation grid', (job: GenerationJob) => { job.tuning.terrain.radius = 500; job.tuning.terrain.cellSize = 8; job.tuning.layout.cellSize = 0.1 }],
  ] as const)('rejects %s before emitting generation progress', async (_name, change) => {
    const request = { ...job(), tuning: structuredClone(tuning) }
    change(request)
    const replies: GenerationReply[] = []
    await createGenerationService(reply => replies.push(reply)).receive({ kind: 'generate', epoch: 1, job: request })
    expect(replies).toEqual([expect.objectContaining({ kind: 'failed', code: 'invalid-input' })])
  })

  it('accepts the declared 1000-actor scale input without imposing a smaller roster cap', () => {
    const request = { input: makeWorldInput({ teams: 4, agents: 1000 }), tuning }
    expect(validateGenerationJob(request).input.agents).toHaveLength(1000)
  })

  it('rejects invalid tuning on the client before creating a worker request', async () => {
    const worker = fakeWorker()
    const client = new GenerationClient(worker)
    const request = { ...job(), tuning: structuredClone(tuning) }
    request.tuning.terrain.octaves = 1000
    await expect(client.generate(request)).rejects.toThrow('octaves')
    expect(worker.postMessage).not.toHaveBeenCalled()
    client.dispose()
  })

  it.each(['park', 'office'] as const)('matches pure generation and transfers only fresh %s buffers', async scene => {
    const request = { input: makeWorldInput({ teams: 1, agents: 2, scene }), tuning }
    const visible = generateWorld(request.input, tuning)
    const original = visible.terrain.height.slice()
    let complete = false
    const failures: string[] = []
    const service = createGenerationService((reply, transfers = []) => {
      if (reply.kind === 'failed') failures.push(reply.detail)
      if (reply.kind !== 'complete') return
      expect(reply.world).toEqual(visible)
      expect(transfers.length).toBeGreaterThan(0)
      expect(new Set(transfers).size).toBe(transfers.length)
      expect(transfers).not.toContain(visible.terrain.height.buffer)
      // Actual platform transfer, not a mock of buffer detachment.
      const received = structuredClone(reply.world, { transfer: transfers })
      expect(reply.world.terrain.height.byteLength).toBe(0)
      expect(reply.world.waterGeometry.length).toBeGreaterThan(0)
      for (const surface of reply.world.waterGeometry) {
        expect(surface.positions.byteLength).toBe(0)
        expect(surface.normals.byteLength).toBe(0)
        expect(surface.indices.byteLength).toBe(0)
        expect(surface.shore.byteLength).toBe(0)
      }
      expect(visible.waterGeometry.every(surface => surface.positions.byteLength > 0)).toBe(true)
      expect(received.waterGeometry.every(surface => surface.positions.byteLength > 0)).toBe(true)
      expect(Array.from(received.terrain.height)).toEqual(Array.from(original))
      expect(visible.terrain.height).toEqual(original)
      complete = true
    })
    await service.receive({ kind: 'generate', epoch: 1, job: request })
    expect(failures).toEqual([])
    expect(complete).toBe(true)
  })

  it('cancels superseded generation and never publishes its arrays', async () => {
    const replies: GenerationReply[] = []
    const service = createGenerationService(reply => replies.push(reply))
    const first = service.receive({ kind: 'generate', epoch: 1, job: job() })
    const second = service.receive({ kind: 'generate', epoch: 2, job: job() })
    await Promise.all([first, second])
    expect(replies.filter(reply => reply.kind === 'cancelled').map(reply => reply.epoch)).toEqual([1])
    expect(replies.filter(reply => reply.kind === 'complete').map(reply => reply.epoch)).toEqual([2])
    await service.receive({ kind: 'generate', epoch: 1, job: job() })
    expect(replies.filter(reply => reply.kind === 'complete')).toHaveLength(1)
  })

  it('rejects invalid input before generation', async () => {
    const replies: GenerationReply[] = []
    const service = createGenerationService(reply => replies.push(reply))
    const invalid = job()
    invalid.input.seed = NaN
    await service.receive({ kind: 'generate', epoch: 1, job: invalid })
    expect(replies).toEqual([expect.objectContaining({ kind: 'failed', code: 'invalid-input' })])
  })

  it('ignores delayed replies and releases listeners on cancellation and disposal', async () => {
    const worker = fakeWorker()
    const client = new GenerationClient(worker)
    const first = client.generate(job())
    const firstRejected = expect(first).rejects.toMatchObject({ name: 'AbortError' })
    const progress = vi.fn()
    const second = client.generate(job(), { onProgress: progress })
    await firstRejected
    const world = generateWorld(job().input, tuning)
    worker.onmessage?.({ data: { kind: 'complete', epoch: 1, world } } as MessageEvent<GenerationReply>)
    worker.onmessage?.({ data: { kind: 'progress', epoch: 1, progress: { stage: 'layout', completed: 1, total: 1 } } } as MessageEvent<GenerationReply>)
    expect(progress).not.toHaveBeenCalled()
    worker.onmessage?.({ data: { kind: 'complete', epoch: 2, world } } as MessageEvent<GenerationReply>)
    expect(await second).toBe(world)
    client.dispose()
    client.dispose()
    expect(worker.terminate).toHaveBeenCalledTimes(1)
    expect(worker.onmessage).toBeNull()
    expect(worker.onerror).toBeNull()
  })

  it('rejects an in-flight job on worker failure', async () => {
    const worker = fakeWorker()
    const client = new GenerationClient(worker)
    const result = client.generate(job())
    worker.onerror?.({ message: 'worker crash' } as ErrorEvent)
    await expect(result).rejects.toThrow('worker crash')
    expect(worker.terminate).toHaveBeenCalledTimes(1)
    await expect(client.generate(job())).rejects.toThrow('disposed')
  })

  it('sends cancellation when the caller aborts and rejects late completion', async () => {
    const worker = fakeWorker()
    const client = new GenerationClient(worker)
    const controller = new AbortController()
    const result = client.generate(job(), { signal: controller.signal })
    controller.abort()
    await expect(result).rejects.toMatchObject({ name: 'AbortError' })
    expect(worker.postMessage).toHaveBeenLastCalledWith({ kind: 'cancel', epoch: 1 } satisfies GenerationRequest)
    client.dispose()
  })

  it('counts aliased backing buffers only once', () => {
    const world = generateWorld(job().input, tuning)
    const transfers = generationTransfers(world)
    expect(transfers.filter(buffer => buffer === world.terrain.height.buffer)).toHaveLength(1)
    expect(transfers.filter(buffer => buffer === world.habitats.buffer)).toHaveLength(1)
    expect(transfers.filter(buffer => buffer === world.habitatRegions.labels.buffer)).toHaveLength(1)
    expect(transfers.filter(buffer => buffer === world.habitatRegions.members.buffer)).toHaveLength(1)
  })
})
