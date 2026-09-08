import { ambientPolicy, METEOR_VARIANTS, type MeteorVariant } from '../../config/ambient'
import { hashString, Rng } from '../rng'

interface TimedEvent { id: string; start: number; end: number; seed: number }
export interface MeteorEvent extends TimedEvent { family: 'meteor'; variant: MeteorVariant }
export interface CometEvent extends TimedEvent { family: 'comet' }
export type SkyEvent = MeteorEvent | CometEvent

/** Seconds on one absolute clock, not elapsed frames. Negative Unix times are valid. */
function validate(seed: number, time: number) {
  if (!Number.isInteger(seed) || seed < 0 || seed > 0xffffffff || !Number.isFinite(time) || Math.abs(time) > 1e12) {
    throw new Error('Invalid ambient seed or absolute time')
  }
}

function stream(seed: number, family: SkyEvent['family'], bucket: number) {
  const id = `ambient-v1:${seed}:${family}:sky:${bucket}`
  const eventSeed = hashString(id)
  return { id, eventSeed, rng: new Rng(eventSeed) }
}

/** One jittered opportunity per bucket gives the declared long-term mean rate. */
export function meteorCandidate(seed: number, bucket: number): MeteorEvent {
  validate(seed, 0)
  if (!Number.isSafeInteger(bucket) || Math.abs(bucket) > 1e10) throw new Error('Invalid ambient bucket')
  const { id, eventSeed, rng } = stream(seed, 'meteor', bucket)
  const start = (bucket + rng.next()) * ambientPolicy.meteor.meanEligibleSeconds
  const variant = METEOR_VARIANTS[rng.weighted(ambientPolicy.meteor.weights)] ?? 'white'
  const duration = variant === 'great-fireball' ? ambientPolicy.meteor.fireballDurationSeconds : ambientPolicy.meteor.durationSeconds
  return { id, seed: eventSeed, family: 'meteor', variant, start, end: start + rng.range(duration[0], duration[1]) }
}

function allowedFireball(seed: number, bucket: number, event: MeteorEvent): boolean {
  if (event.variant !== 'great-fireball') return true
  const { fireballCooldownSeconds, meanEligibleSeconds } = ambientPolicy.meteor
  // Inspect candidates, including suppressed ones: conservative dead time, no mutable
  // history or replay on tab resume. At most 481 candidates for the current policy.
  for (let previous = bucket - 1; previous >= bucket - Math.ceil(fireballCooldownSeconds / meanEligibleSeconds) - 1; previous--) {
    const candidate = meteorCandidate(seed, previous)
    if (event.start - candidate.start >= fireballCooldownSeconds) break
    if (candidate.variant === 'great-fireball') return false
  }
  return true
}

export interface SkyEligibility {
  night: boolean
  clearSky: boolean
  ambientEnabled: boolean
  reducedMotion: boolean
}

/** Stateless seek: return only events alive now, never a backlog of missed events.
 * Camera and quality are deliberately absent. Presenters may cull these identities.
 */
export function skyEventsAt(seed: number, absoluteSeconds: number, eligibility: SkyEligibility): SkyEvent[] {
  validate(seed, absoluteSeconds)
  if (!eligibility.ambientEnabled || !eligibility.night || !eligibility.clearSky) return []
  const events: SkyEvent[] = []
  if (!eligibility.reducedMotion) {
    const bucket = Math.floor(absoluteSeconds / ambientPolicy.meteor.meanEligibleSeconds)
    for (const candidateBucket of [bucket - 1, bucket]) {
      const candidate = meteorCandidate(seed, candidateBucket)
      if (candidate.start <= absoluteSeconds && absoluteSeconds < candidate.end && allowedFireball(seed, candidateBucket, candidate)) events.push(candidate)
    }
  }
  const comet = cometCandidate(seed, Math.floor(absoluteSeconds / ambientPolicy.comet.opportunitySeconds))
  if (comet && comet.start <= absoluteSeconds && absoluteSeconds < comet.end) events.push(comet)
  return events
}

function cometCandidate(seed: number, bucket: number): CometEvent | null {
  const policy = ambientPolicy.comet
  const { id, eventSeed, rng } = stream(seed, 'comet', bucket)
  if (rng.next() < policy.probability) {
    const days = policy.visibleDays[0] + rng.int(policy.visibleDays[1] - policy.visibleDays[0] + 1)
    // Keep the entire multi-night window within its opportunity to enforce max=1.
    const start = bucket * policy.opportunitySeconds + rng.next() * (policy.opportunitySeconds - policy.visibleDays[1] * 86400)
    const end = start + days * 86400
    return { id, seed: eventSeed, family: 'comet', start, end }
  }
  return null
}

/** Next possible membership change. Bounded lookahead, including the next comet
 * opportunity boundary; never scan months of empty opportunities or replay history.
 */
export function nextSkyBoundary(seed: number, now: number, eligibility: SkyEligibility, preview?: SkyEvent): number | null {
  validate(seed, now)
  if (!eligibility.ambientEnabled) return null
  if (preview && now < preview.end) {
    if (eligibility.reducedMotion && preview.family === 'meteor') return preview.end
    return now < preview.start ? preview.start : preview.end
  }
  if (!eligibility.night || !eligibility.clearSky) return null
  const cometBucket = Math.floor(now / ambientPolicy.comet.opportunitySeconds)
  let next = (cometBucket + 1) * ambientPolicy.comet.opportunitySeconds
  const consider = (event: SkyEvent | null) => {
    if (!event) return
    if (event.start > now) next = Math.min(next, event.start)
    if (event.end > now) next = Math.min(next, event.end)
  }
  consider(cometCandidate(seed, cometBucket))
  if (!eligibility.reducedMotion) {
    const bucket = Math.floor(now / ambientPolicy.meteor.meanEligibleSeconds)
    for (const candidateBucket of [bucket - 1, bucket, bucket + 1]) consider(meteorCandidate(seed, candidateBucket))
  }
  return next
}
