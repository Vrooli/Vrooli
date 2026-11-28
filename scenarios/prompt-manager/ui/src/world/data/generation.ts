import { scenes, resolveTerrain, parseTuning, type WorldTuning } from '../config'
import type { CreateWorldInput, GeneratedWorld } from '../sim/model'
import { MAX_TERRAIN_CELLS } from '../sim/terrain/field'
import { MAX_NAV_CELLS } from '../sim/nav/grid'
import { canonical, canonicalRoster, worldRecipe } from '../sim/recipe'
import { runCooperatively } from '../sim/cooperative'
import { generateWorldSteps, type GenerationProgress } from '../sim/world'

export interface GenerationJob {
  input: Omit<CreateWorldInput, 'now'>
  tuning: WorldTuning
}
export type GenerationRequest =
  | { kind: 'generate'; epoch: number; job: GenerationJob }
  | { kind: 'cancel'; epoch: number }
export type GenerationReply =
  | { kind: 'progress'; epoch: number; progress: GenerationProgress }
  | { kind: 'complete'; epoch: number; world: GeneratedWorld }
  | { kind: 'cancelled'; epoch: number }
  | { kind: 'failed'; epoch: number; code: 'invalid-input' | 'generation-failed'; detail: string }

export function validateGenerationJob(job: GenerationJob): GenerationJob {
  // Worker messages cross a runtime trust boundary despite the TypeScript contract.
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
  if (!job || !job.input || !job.tuning) throw new Error('Missing generation input')
  if (!Object.prototype.hasOwnProperty.call(scenes, job.input.scene)) throw new Error('Unknown world scene')
  canonical(job)
  const tuning = parseTuning(job.tuning)
  const input = { ...job.input, ...canonicalRoster(job.input) }
  worldRecipe(input, tuning)
  const terrain = resolveTerrain(scenes[input.scene], tuning).base()
  const terrainCols = Math.ceil(terrain.radius * 2 / terrain.cellSize) + 1
  const navCols = Math.ceil(terrain.radius * 2 / tuning.layout.cellSize)
  if (!Number.isSafeInteger(terrainCols) || terrainCols * terrainCols > MAX_TERRAIN_CELLS) throw new Error('Terrain grid exceeds the allocation budget')
  if (!Number.isSafeInteger(navCols) || navCols * navCols > MAX_NAV_CELLS) throw new Error('Navigation grid exceeds the allocation budget')
  return { input, tuning }
}

/** Only call on freshly generated, unpublished output. Transfer releases every
 * backing buffer to the receiver; neither a visible world nor a cache owns it.
 */
export function generationTransfers(world: GeneratedWorld): ArrayBuffer[] {
  const buffers = new Set<ArrayBuffer>()
  const seen = new Set<object>()
  function visit(value: unknown) {
    if (value === null || typeof value !== 'object' || seen.has(value)) return
    seen.add(value)
    if (ArrayBuffer.isView(value)) {
      if (!(value.buffer instanceof ArrayBuffer)) throw new Error('Shared generation buffers are unsupported')
      buffers.add(value.buffer)
    } else if (value instanceof ArrayBuffer) buffers.add(value)
    else for (const child of Object.values(value)) visit(child)
  }
  visit(world)
  return [...buffers]
}

/** One worker owns one current request. Superseded jobs cannot publish output. */
export function createGenerationService(send: (reply: GenerationReply, transfers?: ArrayBuffer[]) => void, options: { transfer?: boolean } = {}) {
  let active: { epoch: number; controller: AbortController } | undefined
  let latestEpoch = 0
  const cancel = () => {
    if (!active) return
    const previous = active
    active = undefined
    previous.controller.abort()
    send({ kind: 'cancelled', epoch: previous.epoch })
  }
  return {
    async receive(request: GenerationRequest): Promise<void> {
      if (!Number.isSafeInteger(request.epoch) || request.epoch <= 0) return
      if (request.kind === 'cancel') {
        if (active?.epoch === request.epoch) cancel()
        return
      }
      if (request.epoch <= latestEpoch) return
      latestEpoch = request.epoch
      cancel()
      let job: GenerationJob
      try { job = validateGenerationJob(request.job) } catch (error) {
        send({ kind: 'failed', epoch: request.epoch, code: 'invalid-input', detail: String(error) })
        return
      }
      const current = { epoch: request.epoch, controller: new AbortController() }
      active = current
      let lastProgressAt = -Infinity
      let lastStage: GenerationProgress['stage'] | undefined
      try {
        const world = await runCooperatively(generateWorldSteps(job.input, job.tuning), {
          signal: current.controller.signal,
          onProgress: progress => {
            const now = performance.now()
            // Keep cancellation checkpoints cheap without flooding the UI message queue.
            if (active === current && (progress.stage !== lastStage || now - lastProgressAt >= 32)) {
              send({ kind: 'progress', epoch: current.epoch, progress })
              lastProgressAt = now
              lastStage = progress.stage
            }
          },
        })
        if (active !== current) return
        send({ kind: 'complete', epoch: current.epoch, world }, options.transfer === false ? undefined : generationTransfers(world))
      } catch (error) {
        if (active === current) send({ kind: 'failed', epoch: current.epoch, code: 'generation-failed', detail: String(error) })
      } finally {
        if (active === current) active = undefined
      }
    },
    dispose: cancel,
  }
}
