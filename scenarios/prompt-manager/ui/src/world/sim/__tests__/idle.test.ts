import { describe, expect, it } from 'vitest'
import { tuning, type WorldTuning } from '../../config'
import { GATHERING_ID } from '../layout/generate'
import { atHome, insideCommons } from '../idle/behaviors'
import { NOW, run, makeWorld, makeWorldInput } from './fixtures'

function withWeights(weights: WorldTuning['sim']['idle']['weights'], extra: Partial<WorldTuning['sim']['idle']> = {}): WorldTuning {
  return { ...tuning, sim: { ...tuning.sim, idle: { ...tuning.sim.idle, weights, ...extra } } }
}

const MINUTES = 60 / tuning.sim.tickSeconds

describe('idle layer', () => {
  it('spawns every member at its desk and every unassigned agent in the commons', () => {
    const input = makeWorldInput({ teams: 2, agents: 6 })
    const s = makeWorld({ ...input, agents: [...input.agents.map(({ id }) => ({ id, name: id })), { id: 'free', name: 'Free' }], treeVariants: 3 })
    for (const id of s.actorOrder) {
      const actor = s.actors[id]
      if (!actor) throw new Error('missing actor')
      expect(atHome(s, actor, tuning.layout)).toBe(true)
      if (id === 'free') expect(insideCommons(s, actor.position, tuning.layout)).toBe(true)
      else expect(actor.seatId).toBe(actor.deskSeatId)
    }
  })

  it('never lets more than maxMoversRatio of idle actors walk at once', () => {
    const t = withWeights({ rest: 0, wander: 100, socialize: 0, sit: 0 }, { maxMoversRatio: 0.2, rollIntervalSeconds: 0.5 })
    let s = makeWorld({ teams: 2, agents: 20, treeVariants: 3 })
    let worst = 0
    for (let i = 0; i < 3000; i += 1) {
      s = run(s, 1, {}, t)
      const idle = Object.values(s.actors).filter((a) => a.state === 'idle')
      const walking = idle.filter((a) => a.path.length > 0).length
      worst = Math.max(worst, walking / Math.max(1, idle.length))
    }
    expect(worst).toBeLessThanOrEqual(Math.max(0.2, 1 / 20) + 1e-9)
  })

  it('sitting actors take a free campfire seat and hold it', () => {
    const t = withWeights({ rest: 0, wander: 0, socialize: 0, sit: 100 })
    let s = makeWorld({ teams: 1, agents: 4, treeVariants: 3 })
    let seated = 0
    for (let i = 0; i < 3000 && seated === 0; i += 1) {
      s = run(s, 1, {}, t)
      seated = Object.values(s.actors).filter((a) => a.anim.seated && a.seatId?.startsWith('seat:hearth')).length
    }
    expect(seated).toBeGreaterThan(0)
    const holders = Object.values(s.occupancy)
    expect(new Set(holders).size).toBe(holders.length)
  })

  it('wander outings target the commons and keep their spacing from everyone else', () => {
    const t = withWeights({ rest: 0, wander: 100, socialize: 0, sit: 0 }, { maxMoversRatio: 1 })
    let s = makeWorld({ teams: 1, agents: 6, treeVariants: 3 })
    const commons = s.places[GATHERING_ID]
    if (!commons) throw new Error('no commons')
    let outings = 0
    for (let i = 0; i < 4000; i += 1) {
      s = run(s, 1, {}, t)
      for (const a of Object.values(s.actors)) {
        if (a.idle.activity !== 'wander' || !a.destination) continue
        outings += 1
        expect(insideCommons(s, a.destination, tuning.layout)).toBe(true)
        for (const other of Object.values(s.actors)) {
          if (other.id === a.id) continue
          const theirs = other.destination ?? other.position
          expect(Math.hypot(theirs[0] - a.destination[0], theirs[1] - a.destination[1])).toBeGreaterThanOrEqual(tuning.sim.idle.spacing - 1e-9)
        }
      }
    }
    expect(outings).toBeGreaterThan(0)
  })

  it('with the shipped weights most members are home at any moment and some are out', () => {
    let s = run(makeWorld({ teams: 4, agents: 20, treeVariants: 3 }), MINUTES * 3)
    let homeShare = 0
    let outSeen = false
    const samples = 30
    for (let i = 0; i < samples; i += 1) {
      s = run(s, MINUTES / 6)
      const members = Object.values(s.actors).filter((a) => a.deskSeatId)
      const home = members.filter((a) => atHome(s, a, tuning.layout)).length
      homeShare += home / members.length / samples
      if (home < members.length) outSeen = true
    }
    expect(homeShare).toBeGreaterThan(0.5)
    expect(outSeen).toBe(true)
  })

  it('an actor finishing a run stays at its desk; a socializer walks home afterwards', () => {
    const t = withWeights({ rest: 100, wander: 0, socialize: 0, sit: 0 })
    const a = 'agent-0-0'
    let s = run(makeWorld({ teams: 1, agents: 2, treeVariants: 3 }), 1, { 0: [{ kind: 'run.started', agentId: a, runId: 'r', at: NOW }] }, t)
    for (let i = 0; i < 2000 && s.actors[a]?.state !== 'working'; i += 1) s = run(s, 1, {}, t)
    s = run(s, MINUTES, { 0: [{ kind: 'run.finished', agentId: a, runId: 'r', at: NOW }] }, t)
    const finished = s.actors[a]
    if (!finished) throw new Error('missing actor')
    expect(finished.state).toBe('idle')
    expect(atHome(s, finished, tuning.layout)).toBe(true)

    const social = withWeights({ rest: 0, wander: 0, socialize: 100, sit: 0 }, { socializeSeconds: { min: 2, max: 3 } })
    let w = makeWorld({ teams: 1, agents: 2, treeVariants: 3 })
    for (let i = 0; i < 3000 && !Object.values(w.actors).some((x) => x.state === 'socializing'); i += 1) w = run(w, 1, {}, social)
    expect(Object.values(w.actors).some((x) => x.state === 'socializing')).toBe(true)
    let home = false
    for (let i = 0; i < 4000 && !home; i += 1) {
      w = run(w, 1, {}, t)
      home = Object.values(w.actors).every((x) => atHome(w, x, tuning.layout))
    }
    expect(home).toBe(true)
  })

  it('an unassigned agent rests in the commons after its run', () => {
    const t = withWeights({ rest: 100, wander: 0, socialize: 0, sit: 0 })
    let s = makeWorld({ ...makeWorldInput({ teams: 1, agents: 1 }), agents: [{ id: 'agent-0-0', name: 'A' }, { id: 'free', name: 'Free' }], treeVariants: 3 })
    s = run(s, 1, { 0: [{ kind: 'run.started', agentId: 'free', runId: 'r', at: NOW }] }, t)
    expect(s.actors.free?.state).toBe('working')
    s = run(s, MINUTES, { 0: [{ kind: 'run.finished', agentId: 'free', runId: 'r', at: NOW }] }, t)
    const free = s.actors.free
    if (!free) throw new Error('missing actor')
    expect(free.state).toBe('idle')
    expect(insideCommons(s, free.position, tuning.layout)).toBe(true)
  })
})
