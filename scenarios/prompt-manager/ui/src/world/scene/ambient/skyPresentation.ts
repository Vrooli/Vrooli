import { ambientPolicy } from '../../config/ambient'
import type { AnimationLeases } from '../../engine/animationLeases'
import { skyEventsAt, type CometEvent, type MeteorEvent, type SkyEligibility, type SkyEvent } from '../../sim/ambient/schedule'
import { PresentationPool } from './pool'

/** One scene's sky admission and animation ownership. GPU presenters consume slots. */
export class SkyPresentation {
  readonly meteors = new PresentationPool<MeteorEvent>(ambientPolicy.meteor.maximumConcurrent)
  readonly comets = new PresentationPool<CometEvent>(ambientPolicy.comet.maximumConcurrent)
  private releaseMotion: (() => void) | null = null
  private disposed = false
  constructor(private readonly leases: AnimationLeases) {}

  update(seed: number, absoluteSeconds: number, eligibility: SkyEligibility, preview?: SkyEvent, moving = true) {
    if (this.disposed) throw new Error('Ambient presentation is disposed')
    const events = preview
      ? eligibility.ambientEnabled && (!eligibility.reducedMotion || preview.family === 'comet') ? [preview] : []
      : skyEventsAt(seed, absoluteSeconds, eligibility)
    this.meteors.sync(events.filter((event): event is MeteorEvent => event.family === 'meteor'), absoluteSeconds)
    this.comets.sync(events.filter((event): event is CometEvent => event.family === 'comet'), absoluteSeconds)
    if (moving && this.meteors.stats().active > 0) this.releaseMotion ??= this.leases.acquire()
    else { this.releaseMotion?.(); this.releaseMotion = null }
  }

  dispose() {
    this.meteors.clear()
    this.comets.clear()
    this.releaseMotion?.()
    this.releaseMotion = null
    this.disposed = true
  }
}
