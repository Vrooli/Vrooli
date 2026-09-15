import { describe, expect, it } from 'vitest'
import { AnimationLeases } from '../../engine/animationLeases'
import { meteorCandidate } from '../../sim/ambient/schedule'
import { SkyPresentation } from './skyPresentation'

const eligibility = { ambientEnabled: true, night: true, clearSky: true, reducedMotion: false }
describe('sky presentation animation ownership', () => {
  it('keeps a frozen meteor visible without owning motion', () => {
    const leases = new AnimationLeases()
    const presentation = new SkyPresentation(leases)
    const event = meteorCandidate(42, 1000)
    presentation.update(42, event.start + 0.1, eligibility, undefined, false)
    expect(presentation.meteors.stats().active).toBe(1)
    expect(leases.count).toBe(0)
    presentation.dispose()
  })
  it('admits forced previews through the same slots while retaining accessibility gates', () => {
    const leases = new AnimationLeases()
    const presentation = new SkyPresentation(leases)
    const event = meteorCandidate(42, 1000)
    const daylight = { ...eligibility, night: false, clearSky: false }
    presentation.update(42, event.start + 0.1, daylight, event)
    expect(presentation.meteors.slots.some(slot => slot.event?.id === event.id)).toBe(true)
    expect(leases.count).toBe(1)
    presentation.update(42, event.start + 0.2, { ...daylight, ambientEnabled: false }, event)
    expect(leases.count).toBe(0)
    presentation.dispose()
  })
  it('owns one lease for moving events and releases on expiry, disable and disposal', () => {
    const leases = new AnimationLeases()
    const presentation = new SkyPresentation(leases)
    const event = meteorCandidate(42, 1000)
    presentation.update(42, event.start + 0.1, eligibility)
    expect(presentation.meteors.stats().active).toBe(1)
    expect(leases.count).toBe(1)
    presentation.update(42, event.start + 0.2, eligibility)
    expect(leases.count).toBe(1)
    presentation.update(42, event.end, eligibility)
    expect(leases.count).toBe(0)
    presentation.update(42, event.start + 0.1, eligibility)
    presentation.update(42, event.start + 0.2, { ...eligibility, ambientEnabled: false })
    expect(leases.count).toBe(0)
    expect(presentation.meteors.stats().active).toBe(0)
    presentation.update(42, event.start + 0.1, eligibility)
    presentation.dispose(); presentation.dispose()
    expect(leases.count).toBe(0)
    expect(presentation.meteors.stats().released).toBe(presentation.meteors.stats().acquired)
    expect(() => presentation.update(42, event.start, eligibility)).toThrow('disposed')
  })
  it('suppresses meteor motion immediately when reduced motion changes', () => {
    const leases = new AnimationLeases()
    const presentation = new SkyPresentation(leases)
    const event = meteorCandidate(42, 1000)
    presentation.update(42, event.start + 0.1, eligibility)
    presentation.update(42, event.start + 0.2, { ...eligibility, reducedMotion: true })
    expect(leases.count).toBe(0)
    expect(presentation.meteors.stats().active).toBe(0)
    presentation.dispose()
  })
})
