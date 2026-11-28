export interface PreparationStage {
  stage: 'assets' | 'generation' | 'presentation'
  startedAt: number
  completedAt: number | null
  outcome: 'running' | 'complete' | 'failed' | 'cancelled'
}
type StageRunner = <T>(name: PreparationStage['stage'], work: () => Promise<T>) => Promise<T>
export interface PreparationSnapshot {
  epoch: number
  state: 'idle' | 'preparing' | 'committed' | 'cancelled' | 'failed'
  requestedAt: number | null
  completedAt: number | null
  stages: PreparationStage[]
}

/** Owns requested epochs separately from renderer commit epochs. Obsolete work
 * cannot commit even when a preparation provider ignores its abort signal.
 */
export class ScenePreparationCoordinator {
  private nextEpoch = 0
  private active?: { epoch: number; controller: AbortController }
  private snapshot: PreparationSnapshot = { epoch: 0, state: 'idle', requestedAt: null, completedAt: null, stages: [] }
  read = (): PreparationSnapshot => ({ ...this.snapshot, stages: this.snapshot.stages.map(stage => ({ ...stage })) })

  start<T>(prepare: (signal: AbortSignal, runStage: StageRunner) => Promise<T>, commit: (value: T) => void) {
    this.cancel()
    const current = { epoch: ++this.nextEpoch, controller: new AbortController() }
    this.active = current
    this.snapshot = { epoch: current.epoch, state: 'preparing', requestedAt: performance.now(), completedAt: null, stages: [] }
    const runStage: StageRunner = async (stage, work) => {
      current.controller.signal.throwIfAborted()
      if (this.active !== current) throw new DOMException('Preparation epoch is no longer active', 'AbortError')
      const span: PreparationStage = { stage, startedAt: performance.now(), completedAt: null, outcome: 'running' }
      this.snapshot.stages = [...this.snapshot.stages.slice(-31), span]
      try {
        const value = await work()
        if (this.active === current) { span.completedAt = performance.now(); span.outcome = 'complete' }
        return value
      } catch (error) {
        if (this.active === current) { span.completedAt = performance.now(); span.outcome = 'failed' }
        throw error
      }
    }
    const promise = Promise.resolve().then(() => {
      current.controller.signal.throwIfAborted()
      return prepare(current.controller.signal, runStage)
    }).then(value => {
      if (this.active !== current || current.controller.signal.aborted) return false
      commit(value)
      this.snapshot = { ...this.snapshot, state: 'committed', completedAt: performance.now() }
      this.active = undefined
      return true
    }).catch((error: unknown) => {
      if (this.active !== current || current.controller.signal.aborted) return false
      current.controller.abort()
      this.finishRunningStages()
      this.active = undefined
      this.snapshot = { ...this.snapshot, state: 'failed', completedAt: performance.now() }
      throw error
    })
    return { epoch: current.epoch, promise, cancel: () => { if (this.active === current) this.cancel() } }
  }

  cancel(): void {
    if (!this.active) return
    this.active.controller.abort()
    this.finishRunningStages()
    this.active = undefined
    this.snapshot = { ...this.snapshot, state: 'cancelled', completedAt: performance.now() }
  }

  private finishRunningStages(): void {
    const now = performance.now()
    for (const stage of this.snapshot.stages) if (stage.outcome === 'running') {
      stage.outcome = 'cancelled'
      stage.completedAt = now
    }
  }
}
