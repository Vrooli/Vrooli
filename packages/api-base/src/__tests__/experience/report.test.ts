import { afterEach, describe, expect, it } from 'vitest'
import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import {
  analyzeExperience,
  buildSessions,
  loadDeclaredSpec,
  renderFrictionMarkdown,
  routeMatches,
} from '../../experience/report.js'
import type { ExperienceBatch, ExperienceEvent } from '../../experience/types.js'

const dirs: string[] = []

function tempDir(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'experience-report-'))
  dirs.push(dir)
  return dir
}

afterEach(() => {
  while (dirs.length > 0) fs.rmSync(dirs.pop()!, { recursive: true, force: true })
})

const element = (fp: string, testId: string, name: string) => ({
  tag: 'button', testId, name, role: 'button', path: `button[data-testid="${testId}"]`, fp,
})

function makeSession(sessionId: string, base: number): ExperienceBatch {
  let seq = 0
  const event = (type: string, ts: number, extra: Record<string, unknown>): ExperienceEvent => ({
    type, ts, seq: seq++, ...extra,
  } as ExperienceEvent)

  const events: ExperienceEvent[] = [
    event('session_start', base, { route: '/a', viewport: { width: 1280, height: 800, dpr: 1 }, reducedMotion: false, theme: 'light' }),
    event('route', base, { route: '/a', via: 'load', scrollDepth: 0 }),
    event('action', base + 500, { action: 'click', element: element('save', 'save', 'Save'), hoverMs: 200 }),
    event('action', base + 1000, { action: 'click', element: element('open', 'open', 'Open') }),
    event('friction', base + 1200, { kind: 'dead_click', element: element('save', 'save', 'Save'), route: '/a', detail: { waitedMs: 700 } }),
    event('field', base + 1500, { phase: 'focus', field: { type: 'password', name: '[redacted]', sensitive: true } }),
    event('field', base + 2500, { phase: 'blur', field: { type: 'password', name: '[redacted]', sensitive: true, durationMs: 1000, edited: true } }),
    event('route', base + 3000, { route: '/b', via: 'push', scrollDepth: 0.5 }),
    event('action', base + 3400, { action: 'click', element: element('next', 'next', 'Next') }),
    event('error', base + 3500, { kind: 'error', message: 'boom' }),
    event('session_end', base + 4000, { activeMs: 4000, reason: 'hidden' }),
  ]
  return { v: 1, sessionId, scenario: 'demo', receivedAt: base + 4000, events }
}

function writeSpec(): string {
  const root = tempDir()
  fs.writeFileSync(path.join(root, 'index.json'), JSON.stringify({
    kind: 'experience-index',
    schemaVersion: '1.0.0',
    pages: [
      { id: 'a', path: 'pages/a.json', title: 'A', status: 'active' },
      { id: 'b', path: 'pages/b.json', title: 'B', status: 'active' },
    ],
    journeys: [{ id: 'go', path: 'journeys/go.json', title: 'Go', status: 'active' }],
  }))
  fs.mkdirSync(path.join(root, 'pages'))
  fs.mkdirSync(path.join(root, 'journeys'))
  fs.writeFileSync(path.join(root, 'pages', 'a.json'), JSON.stringify({ page: { id: 'a', routes: ['/a'] } }))
  fs.writeFileSync(path.join(root, 'pages', 'b.json'), JSON.stringify({ page: { id: 'b', routes: ['/b'] } }))
  fs.writeFileSync(path.join(root, 'journeys', 'go.json'), JSON.stringify({
    journey: { id: 'go', title: 'Go somewhere', purpose: 'Complete the declared path' },
    steps: [{ page: 'a', intent: 'Start here' }, { page: 'b', intent: 'Finish here' }],
  }))
  return root
}

describe('buildSessions', () => {
  it('groups batches and removes retried duplicates', () => {
    const batch = makeSession('abcdefgh12345678', 1_000_000)
    const sessions = buildSessions([batch, batch])
    expect(sessions).toHaveLength(1)
    expect(sessions[0]!.events).toHaveLength(11)
    expect(sessions[0]!.startedAt).toBe(1_000_000)
    expect(sessions[0]!.endedAt).toBe(1_004_000)
  })
})

describe('analyzeExperience', () => {
  const batches = [makeSession('abcdefgh12345678', 1_000_000), makeSession('abcdefgh87654321', 2_000_000)]

  it('derives totals, routes, controls, friction, fields, and errors', () => {
    const { report } = analyzeExperience(batches, { scenario: 'demo', minFlowCount: 2 })
    expect(report.totals.sessions).toBe(2)
    expect(report.totals.actions).toBe(6)
    expect(report.totals.routes).toBe(2)

    const routeA = report.routes.find((route) => route.route === '/a')!
    expect(routeA.visits).toBe(2)
    expect(routeA.medianDwellMs).toBe(3000)
    expect(routeA.firstActionMs).toBe(500)
    expect(routeA.actionsPerVisit).toBe(2)

    const save = report.controls.find((control) => control.testId === 'save')!
    expect(save.count).toBe(2)
    expect(save.medianHoverMs).toBe(200)
    expect(save.friction).toBe(2)

    const dead = report.friction.find((item) => item.kind === 'dead_click')!
    expect(dead.count).toBe(2)
    expect(dead.elements[0]!.testId).toBe('save')

    const field = report.fields.find((item) => item.name === '[redacted]')!
    expect(field.focuses).toBe(2)
    expect(field.editRate).toBe(1)
    expect(field.abandonmentRate).toBe(0)

    expect(report.errors[0]!.message).toBe('boom')
    expect(report.errors[0]!.count).toBe(2)
  })

  it('ranks repeated flows and recommends the dead-click fix', () => {
    const { report, observedFlows } = analyzeExperience(batches, { scenario: 'demo', minFlowCount: 2 })
    expect(report.flows.length).toBeGreaterThan(0)
    expect(report.flows[0]!.count).toBeGreaterThanOrEqual(2)
    expect(observedFlows[0]!.kind).toBe('experience-observed-flow')
    expect(report.recommendations.some((item) => item.id === 'dead-clicks' && item.severity === 'high')).toBe(true)
    expect(report.notes.some((note) => note.includes('Fewer than three sessions'))).toBe(true)
  })

  it('compares declared journeys and keeps undeclared flows separate', () => {
    const specDir = writeSpec()
    const { report } = analyzeExperience(batches, { scenario: 'demo', specDir, minFlowCount: 2 })
    const journey = report.journeys.find((item) => item.journeyId === 'go')!
    expect(journey.sessionsMatched).toBe(2)
    expect(journey.coverage).toBe(1)
    expect(journey.medianDurationMs).toBe(3000)
    expect(journey.friction).toBe(2)
    expect(report.undeclaredFlows).toHaveLength(0)
  })

  it('renders a reviewable markdown report', () => {
    const { report } = analyzeExperience(batches, { scenario: 'demo', minFlowCount: 2 })
    const markdown = renderFrictionMarkdown(report)
    expect(markdown).toContain('# Experience friction: demo')
    expect(markdown).toContain('## What to fix first')
    expect(markdown).toContain('## Where time goes')
    expect(markdown).toContain('## Friction signals')
  })

  it('handles an empty dataset without throwing', () => {
    const { report } = analyzeExperience([], { scenario: 'empty' })
    expect(report.totals.sessions).toBe(0)
    expect(report.notes.some((note) => note.includes('Fewer than three sessions'))).toBe(true)
    expect(report.window.from).toBeNull()
  })
})

describe('leaving and returning', () => {
  function makeAwaySession(sessionId: string, base: number): ExperienceBatch {
    let seq = 0
    const event = (type: string, ts: number, extra: Record<string, unknown>): ExperienceEvent => ({
      type, ts, seq: seq++, ...extra,
    } as ExperienceEvent)
    return {
      v: 1,
      sessionId,
      scenario: 'demo',
      receivedAt: base + 80_000,
      events: [
        event('session_start', base, { route: '/slow', viewport: { width: 1280, height: 800, dpr: 1 }, reducedMotion: false, theme: 'light' }),
        event('route', base, { route: '/slow', via: 'load', scrollDepth: 0 }),
        event('visibility', base + 2_000, { state: 'hidden' }),
        event('visibility', base + 42_000, { state: 'visible' }),
        event('visibility', base + 45_000, { state: 'hidden' }),
        event('visibility', base + 75_000, { state: 'visible' }),
        event('session_end', base + 80_000, { activeMs: 10_000, reason: 'hidden' }),
      ],
    }
  }

  it('attributes hidden time to the current route and excludes it from active time', () => {
    const { report } = analyzeExperience([makeAwaySession('awaysession12345', 1_000_000)], { scenario: 'demo' })
    expect(report.totals.returns).toBe(2)
    expect(report.totals.awayMs).toBe(70_000)
    expect(report.totals.activeMs).toBe(10_000)

    const route = report.routes.find((item) => item.route === '/slow')!
    expect(route.returns).toBe(2)
    expect(route.awayMs).toBe(70_000)
    expect(route.medianAwayMs).toBe(35_000)
  })

  it('raises an advisory recommendation but never a verdict', () => {
    const { report } = analyzeExperience([makeAwaySession('awaysession12345', 1_000_000)], { scenario: 'demo' })
    const friction = report.friction.find((item) => item.kind === 'leave_and_return')!
    expect(friction.count).toBe(2)
    expect(friction.routes[0]).toEqual({ route: '/slow', count: 2 })

    const recommendation = report.recommendations.find((item) => item.id === 'leave-and-return')!
    expect(recommendation.severity).toBe('high')
    expect(recommendation.evidence).toContain('median 35s away')

    const markdown = renderFrictionMarkdown(report)
    expect(markdown).toContain('## Leaving and returning')
    expect(markdown).toContain('| /slow | 2 |')
  })

  it('does not flag a brief or single hide', () => {
    const batch = makeAwaySession('awaysession67890', 1_000_000)
    batch.events = batch.events.filter((event) => event.ts !== 1_045_000 && event.ts !== 1_075_000)
    const { report } = analyzeExperience([batch], { scenario: 'demo' })
    expect(report.totals.returns).toBe(1)
    expect(report.friction.some((item) => item.kind === 'leave_and_return')).toBe(false)
    expect(report.recommendations.some((item) => item.id === 'leave-and-return')).toBe(false)
  })
})

describe('declared capabilities', () => {
  function makeClickSession(sessionId: string, base: number, clicks: number): ExperienceBatch {
    let seq = 0
    const event = (type: string, ts: number, extra: Record<string, unknown>): ExperienceEvent => ({
      type, ts, seq: seq++, ...extra,
    } as ExperienceEvent)
    const events: ExperienceEvent[] = [
      event('session_start', base, { route: '/commit', viewport: { width: 1280, height: 800, dpr: 1 }, reducedMotion: false, theme: 'light' }),
      event('route', base, { route: '/commit', via: 'load', scrollDepth: 0 }),
    ]
    for (let index = 0; index < clicks; index += 1) {
      events.push(event('action', base + 500 + index * 1000, { action: 'click', element: element('other', 'other', 'Other') }))
    }
    events.push(event('session_end', base + 500 + clicks * 1000, { activeMs: 500 + clicks * 1000, reason: 'unload' }))
    return { v: 1, sessionId, scenario: 'demo', receivedAt: base + 500 + clicks * 1000, events }
  }

  it('reports a declared capability that recorded no clicks', () => {
    const { report } = analyzeExperience([makeClickSession('clicker12345678', 1_000_000, 12)], {
      scenario: 'demo',
      capabilities: [{ id: 'stage-queue', label: 'Queue files', testId: 'stage-during-suggestion-queue' }],
    })
    expect(report.capabilities).toHaveLength(1)
    expect(report.capabilities[0]).toMatchObject({ id: 'stage-queue', exercised: 0, state: 'unobserved' })

    const recommendation = report.recommendations.find((item) => item.id === 'capability-unobserved')!
    expect(recommendation.title).toContain('Queue files')
    expect(recommendation.suggestion).toContain('discovery')

    const markdown = renderFrictionMarkdown(report)
    expect(markdown).toContain('## Declared capabilities')
    expect(markdown).toContain('| Queue files | 0 | unobserved |')
  })

  it('does not judge a capability from too little interaction', () => {
    const { report } = analyzeExperience([makeClickSession('clicker87654321', 1_000_000, 3)], {
      scenario: 'demo',
      capabilities: [{ id: 'stage-queue', testId: 'stage-during-suggestion-queue' }],
    })
    expect(report.capabilities[0]!.state).toBe('insufficient-evidence')
    expect(report.recommendations.some((item) => item.id === 'capability-unobserved')).toBe(false)
  })

  it('marks a capability exercised when its control is clicked', () => {
    const batch = makeClickSession('clicker11112222', 1_000_000, 12)
    batch.events.push({
      type: 'action',
      ts: 1_012_000,
      seq: 99,
      action: 'click',
      element: element('queue', 'stage-during-suggestion-queue', 'Queue files'),
    } as ExperienceEvent)
    const { report } = analyzeExperience([batch], {
      scenario: 'demo',
      capabilities: [{ id: 'stage-queue', testId: 'stage-during-suggestion-queue' }],
    })
    expect(report.capabilities[0]).toMatchObject({ exercised: 1, state: 'exercised' })
  })
})

describe('loadDeclaredSpec', () => {
  it('degrades to no journeys for missing or malformed specs', () => {
    expect(loadDeclaredSpec(undefined).journeys).toEqual([])
    expect(loadDeclaredSpec(path.join(tempDir(), 'absent')).journeys).toEqual([])
    const broken = tempDir()
    fs.writeFileSync(path.join(broken, 'index.json'), '{oops')
    expect(loadDeclaredSpec(broken).journeys).toEqual([])
  })
})

describe('routeMatches', () => {
  it('honours parameters and wildcards but not extra segments', () => {
    expect(routeMatches('/repo/:id', '/repo/abc')).toBe(true)
    expect(routeMatches('/repo/:id', '/repo/abc/diff')).toBe(false)
    expect(routeMatches('/settings/*', '/settings/theme')).toBe(true)
    expect(routeMatches('/', '/')).toBe(true)
    expect(routeMatches('/a?tab=1', '/a')).toBe(true)
  })
})
