import { describe, expect, it } from 'vitest'
import {
  canonicalJson,
  computeReportDigest,
  createObservedEvidence,
  createPacketObservedBlock,
  deriveAdvisoryFindings,
  deriveCorrectionCandidates,
  validateReport,
} from '../../experience/contracts.js'
import type { ExperienceFrictionReport } from '../../experience/types.js'

function report(overrides: Partial<ExperienceFrictionReport> = {}): ExperienceFrictionReport {
  return {
    schemaVersion: 'experience-friction-report/v1',
    scenario: 'demo',
    generatedAt: '2026-11-24T00:00:00.000Z',
    window: { from: '2026-11-24T00:00:00.000Z', to: '2026-11-24T00:05:00.000Z', sessions: 2 },
    builds: ['sha256:build-a'],
    sessionRefs: [
      { id: 'sess-a', build: 'sha256:build-a', startedAt: '2026-11-24T00:00:00.000Z', endedAt: '2026-11-24T00:02:00.000Z' },
      { id: 'sess-b', build: 'sha256:build-a', startedAt: '2026-11-24T00:03:00.000Z', endedAt: '2026-11-24T00:05:00.000Z' },
    ],
    totals: { sessions: 2, events: 10, actions: 4, activeMs: 5000, awayMs: 0, returns: 0, medianSessionMs: 2500, routes: 2 },
    routes: [
      { route: '/a', visits: 2, dwellMs: 6000, medianDwellMs: 3000, firstActionMs: 500, actionsPerVisit: 1, scrollDepth: 0.5, returns: 0, awayMs: 0, medianAwayMs: null },
      { route: '/slow', visits: 2, dwellMs: 40000, medianDwellMs: 20000, firstActionMs: null, actionsPerVisit: 0, scrollDepth: null, returns: 2, awayMs: 40000, medianAwayMs: 20000 },
    ],
    controls: [],
    flows: [{ id: 'f1', steps: ['/a', 'save'], count: 4, medianDurationMs: 1200, totalDurationMs: 4800 }],
    friction: [
      {
        kind: 'dead_click',
        count: 3,
        routes: [{ route: '/a', count: 3 }],
        elements: [{ fp: 'x', testId: 'save', name: 'Save', count: 3 }],
      },
    ],
    fields: [{ name: 'email', type: 'email', focuses: 5, medianDurationMs: 1000, editRate: 0.2, abandonmentRate: 0.4 }],
    journeys: [],
    undeclaredFlows: [],
    capabilities: [],
    errors: [],
    recommendations: [
      { id: 'dead-clicks', severity: 'high', title: 'Dead clicks on save', evidence: '3 clicks produced no change.', suggestion: 'Check the affordance.' },
    ],
    notes: [],
    ...overrides,
  }
}

describe('canonicalJson / computeReportDigest', () => {
  it('is stable across key order and changes when the body changes', () => {
    expect(canonicalJson({ b: 1, a: 2 })).toBe('{"a":2,"b":1}')
    const first = computeReportDigest(report())
    const second = computeReportDigest(report())
    expect(first).toBe(second)
    expect(first).toMatch(/^sha256:[a-f0-9]{64}$/)

    const changed = computeReportDigest(report({ scenario: 'other' }))
    expect(changed).not.toBe(first)
  })

  it('ignores an existing digest field', () => {
    const base = report()
    const digest = computeReportDigest(base)
    expect(computeReportDigest({ ...base, reportDigest: digest })).toBe(digest)
  })

  it('does not trust a forged digest when building evidence', () => {
    const base = report()
    const forged = { ...base, reportDigest: `sha256:${'0'.repeat(64)}` }
    expect(createObservedEvidence(forged).reportDigest).toBe(computeReportDigest(base))
  })
})

describe('createObservedEvidence', () => {
  it('is fresh when sessions exist and the build matches', () => {
    const evidence = createObservedEvidence(report(), { expectedBuild: 'sha256:build-a' })
    expect(evidence.state).toBe('fresh')
    expect(evidence.build).toBe('sha256:build-a')
    expect(evidence.reportDigest).toMatch(/^sha256:/)
  })

  it('is stale when the expected build was not observed', () => {
    const evidence = createObservedEvidence(report(), { expectedBuild: 'sha256:build-b' })
    expect(evidence.state).toBe('stale')
    expect(evidence.expectedBuild).toBe('sha256:build-b')
  })

  it('fails closed when a session build is missing or sessions span builds', () => {
    expect(createObservedEvidence(report({ sessionRefs: [{ id: 's1', build: null, startedAt: '2026-11-24T00:00:00Z', endedAt: '2026-11-24T00:01:00Z' }] })).state).toBe('stale')
    expect(createObservedEvidence(report({
      builds: ['sha256:build-a', 'sha256:build-b'],
      sessionRefs: [
        { id: 's1', build: 'sha256:build-a', startedAt: '2026-11-24T00:00:00Z', endedAt: '2026-11-24T00:01:00Z' },
        { id: 's2', build: 'sha256:build-b', startedAt: '2026-11-24T00:01:00Z', endedAt: '2026-11-24T00:02:00Z' },
      ],
    })).state).toBe('stale')
  })

  it('is absent when there are no sessions', () => {
    const empty = report({
      window: { from: null, to: null, sessions: 0 },
      builds: [],
      sessionRefs: [],
      totals: { sessions: 0, events: 0, actions: 0, activeMs: 0, medianSessionMs: 0, routes: 0 },
    })
    expect(createObservedEvidence(empty).state).toBe('absent')
    expect(createObservedEvidence(empty, { expectedBuild: 'x' }).state).toBe('absent')
  })
})

describe('advisory findings and correction candidates', () => {
  it('marks findings advisory and binds element, route and digest', () => {
    const findings = deriveAdvisoryFindings(report())
    expect(findings).toHaveLength(1)
    expect(findings[0]).toMatchObject({
      id: 'adv-dead-clicks',
      kind: 'dead_click',
      severity: 'high',
      advisory: true,
      route: '/a',
      element: { fp: 'x', testId: 'save' },
    })
    expect(findings[0]!.reportDigest).toMatch(/^sha256:/)
  })

  it('emits review candidates, never corrections', () => {
    const candidates = deriveCorrectionCandidates(report())
    const kinds = candidates.map((candidate) => candidate.kind)
    expect(kinds).toContain('dead_click')
    expect(kinds).toContain('field_abandonment')
    expect(kinds).toContain('dwell_without_action')
    expect(kinds).toContain('leave_and_return')
    candidates.forEach((candidate) => {
      expect(candidate.status).toBe('candidate')
      expect(candidate.source).toBe('observed')
      expect(candidate.advisory).toBe(true)
      expect(candidate.reportDigest).toMatch(/^sha256:/)
    })
  })

  it('deduplicates candidates by kind, route and element', () => {
    const candidates = deriveCorrectionCandidates(report())
    const keys = candidates.map((candidate) => `${candidate.kind}|${candidate.route || ''}|${candidate.element?.fp || candidate.element?.name || ''}`)
    expect(new Set(keys).size).toBe(keys.length)
  })
})

describe('packet block', () => {
  it('summarises the top flow and keeps the advisory caveat', () => {
    const block = createPacketObservedBlock(report())
    expect(block.schemaVersion).toBe('experience-packet-observed/v1')
    expect(block.topFlow?.count).toBe(4)
    expect(block.headline).toContain('/a -> save')
    expect(block.timeCostMs).toBe(4800)
    expect(block.caveat).toContain('Advisory only')
    expect(block.reportDigest).toMatch(/^sha256:/)
    expect(block.sessions).toHaveLength(2)
  })
})

describe('validateReport', () => {
  it('accepts a self-consistent report and rejects a tampered digest', () => {
    const base = report()
    const valid = { ...base, reportDigest: computeReportDigest(base) }
    const check = validateReport(valid)
    expect(check.ok).toBe(true)
    if (check.ok) expect(check.digestValid).toBe(true)

    const tampered = { ...valid, scenario: 'tampered' }
    const bad = validateReport(tampered)
    expect(bad.ok).toBe(true)
    if (bad.ok) expect(bad.digestValid).toBe(false)
  })

  it('rejects malformed shapes', () => {
    expect(validateReport(null).ok).toBe(false)
    expect(validateReport({ schemaVersion: 'wrong' }).ok).toBe(false)
    expect(validateReport({ ...report(), sessionRefs: undefined }).ok).toBe(false)
  })
})
