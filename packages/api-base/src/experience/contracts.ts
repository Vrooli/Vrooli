/**
 * Consumer contracts for observed interaction evidence.
 *
 * The report is the human-facing artifact. These functions produce the machine
 * contracts other owners consume:
 *
 * - `ExperienceObservedBehavior` — the evidence envelope, with an explicit
 *   fresh/stale/absent state so a missing or mismatched session can never be
 *   read as fresh evidence.
 * - `ExperienceAdvisoryFinding[]` — ranked findings that are useful but never a
 *   gate or a phase verdict.
 * - `ExperienceCorrectionCandidate[]` — review candidates for the EX1 corpus.
 *   They are not gold labels and must not be written as corrections directly.
 * - `ExperiencePacketObservedBlock` — a small block a redesign packet can show
 *   beside a captured before/after.
 *
 * The digest makes the evidence reproducible and ties every consumer artifact
 * back to the exact report body.
 */

import { createHash } from 'node:crypto'
import {
  type ExperienceAdvisoryFinding,
  type ExperienceAdvisoryKind,
  type ExperienceCorrectionCandidate,
  type ExperienceFrictionReport,
  type ExperienceObservedBehavior,
  type ExperiencePacketObservedBlock,
} from './types.js'

/** Deterministic JSON: sorted keys, undefined omitted, arrays preserved. */
export function canonicalJson(value: unknown): string {
  if (value === null) return 'null'
  if (typeof value === 'number') return Number.isFinite(value) ? String(value) : 'null'
  if (typeof value === 'boolean') return value ? 'true' : 'false'
  if (typeof value === 'string') return JSON.stringify(value)
  if (Array.isArray(value)) {
    return `[${value.map((item) => canonicalJson(item === undefined ? null : item)).join(',')}]`
  }
  if (typeof value === 'object') {
    const record = value as Record<string, unknown>
    const keys = Object.keys(record).filter((key) => record[key] !== undefined).sort()
    return `{${keys.map((key) => `${JSON.stringify(key)}:${canonicalJson(record[key])}`).join(',')}}`
  }
  return 'null'
}

/** sha256 over the report body, excluding the digest field itself. */
export function computeReportDigest(report: ExperienceFrictionReport): string {
  const { reportDigest: _ignored, ...body } = report
  return `sha256:${createHash('sha256').update(canonicalJson(body)).digest('hex')}`
}

function withDigest(report: ExperienceFrictionReport): ExperienceFrictionReport {
  const computed = computeReportDigest(report)
  if (report.reportDigest === computed) return report
  // Recompute rather than trust a stale or tampered digest field.
  return { ...report, reportDigest: computed }
}

function elementOf(report: ExperienceFrictionReport, kind: string): { fp?: string; testId?: string; name?: string; role?: string } | undefined {
  const friction = report.friction.find((item) => item.kind === kind)
  const element = friction?.elements[0]
  if (!element) return undefined
  return {
    fp: element.fp,
    ...(element.testId ? { testId: element.testId } : {}),
    ...(element.name ? { name: element.name } : {}),
  }
}

function routeOf(report: ExperienceFrictionReport, kind: string): string | undefined {
  return report.friction.find((item) => item.kind === kind)?.routes[0]?.route
}

const RECOMMENDATION_KIND: Record<string, ExperienceAdvisoryKind> = {
  'dead-clicks': 'dead_click',
  'rage-clicks': 'rage_click',
  'repeated-actions': 'repeated_action',
  'leave-and-return': 'leave_and_return',
  'capability-unobserved': 'capability_unobserved',
  'dwell-without-action': 'dwell_without_action',
  'field-abandonment': 'field_abandonment',
  'undeclared-flow': 'undeclared_flow',
  'declared-unobserved': 'declared_unobserved',
  'repeated-flow': 'repeated_flow',
  'runtime-errors': 'runtime_error',
}

/**
 * Convert report recommendations into advisory findings. The report already
 * holds the evidence; this only adds the element/route binding and the digest
 * so a consumer can cite it without parsing prose.
 */
export function deriveAdvisoryFindings(input: ExperienceFrictionReport): ExperienceAdvisoryFinding[] {
  const report = withDigest(input)
  const digest = report.reportDigest!
  const findings: ExperienceAdvisoryFinding[] = []

  report.recommendations.forEach((recommendation) => {
    const kind = RECOMMENDATION_KIND[recommendation.id]
    if (!kind) return
    const element = elementOf(report, kind)
    const route = routeOf(report, kind)
    findings.push({
      schemaVersion: 'experience-advisory-finding/v1',
      id: `adv-${recommendation.id}`,
      severity: recommendation.severity,
      kind,
      title: recommendation.title,
      evidence: recommendation.evidence,
      suggestion: recommendation.suggestion,
      ...(route ? { route } : {}),
      ...(element ? { element } : {}),
      reportDigest: digest,
      advisory: true,
    })
  })

  return findings
}

/**
 * Convert observed friction into review candidates for the correction corpus.
 * A candidate is a question ("was this friction a defect?"), never an answer.
 */
export function deriveCorrectionCandidates(input: ExperienceFrictionReport): ExperienceCorrectionCandidate[] {
  const report = withDigest(input)
  const digest = report.reportDigest!
  const candidates: ExperienceCorrectionCandidate[] = []
  const seen = new Set<string>()

  const add = (candidate: Omit<ExperienceCorrectionCandidate, 'schemaVersion' | 'id' | 'reportDigest' | 'status' | 'source' | 'advisory'>): void => {
    const key = `${candidate.kind}|${candidate.route || ''}|${candidate.element?.fp || candidate.element?.name || ''}`
    if (seen.has(key)) return
    seen.add(key)
    candidates.push({
      schemaVersion: 'experience-correction-candidate/v1',
      id: `cand-${createHash('sha1').update(key).digest('hex').slice(0, 12)}`,
      reportDigest: digest,
      status: 'candidate',
      source: 'observed',
      advisory: true,
      ...candidate,
    })
  }

  const frictionKinds: Array<{ kind: ExperienceAdvisoryKind; severity: 'high' | 'medium' | 'low'; label: string }> = [
    { kind: 'dead_click', severity: 'high', label: 'Dead click' },
    { kind: 'rage_click', severity: 'medium', label: 'Rage click' },
    { kind: 'repeated_action', severity: 'medium', label: 'Repeated action' },
  ]

  frictionKinds.forEach(({ kind, severity, label }) => {
    const metric = report.friction.find((item) => item.kind === kind)
    if (!metric) return
    metric.elements.slice(0, 5).forEach((element) => {
      const route = metric.routes[0]?.route
      add({
        kind,
        severity,
        ...(route ? { route } : {}),
        element: {
          fp: element.fp,
          ...(element.testId ? { testId: element.testId } : {}),
          ...(element.name ? { name: element.name } : {}),
        },
        evidence: `${label} on ${element.testId || element.name || element.fp} occurred ${element.count} time(s)${route ? ` on ${route}` : ''}.`,
      })
    })
  })

  report.routes
    .filter((route) => route.returns >= 2 && route.medianAwayMs !== null && route.medianAwayMs >= 15000)
    .slice(0, 5)
    .forEach((route) => {
      add({
        kind: 'leave_and_return',
        severity: 'high',
        route: route.route,
        evidence: `${route.returns} hide/return cycle(s) on ${route.route}, median ${Math.round((route.medianAwayMs ?? 0) / 1000)}s away.`,
      })
    })

  report.fields
    .filter((field) => field.focuses >= 4 && field.abandonmentRate >= 0.25)
    .slice(0, 5)
    .forEach((field) => {
      add({
        kind: 'field_abandonment',
        severity: 'medium',
        element: { name: field.name },
        evidence: `${Math.round(field.abandonmentRate * 100)}% of ${field.focuses} focuses on "${field.name}" ended without input.`,
      })
    })

  report.routes
    .filter((route) => route.visits >= 2 && route.medianDwellMs >= 15000 && route.actionsPerVisit < 1)
    .slice(0, 5)
    .forEach((route) => {
      add({
        kind: 'dwell_without_action',
        severity: 'medium',
        route: route.route,
        evidence: `Median dwell ${Math.round(route.medianDwellMs / 1000)}s across ${route.visits} visits with ${route.actionsPerVisit} actions per visit.`,
      })
    })

  return candidates.slice(0, 50)
}

export interface ObservedEvidenceOptions {
  /** Build the consumer is discussing; a mismatch produces `stale`. */
  expectedBuild?: string
}

/**
 * Build the evidence envelope. `absent`/`stale` are first-class: a consumer
 * must never promote them to fresh evidence or a passing verdict.
 */
export function createObservedEvidence(
  input: ExperienceFrictionReport,
  options: ObservedEvidenceOptions = {},
): ExperienceObservedBehavior {
  const report = withDigest(input)
  const digest = report.reportDigest!
  const builds = report.builds.filter((build) => Boolean(build))
  const sessions = report.sessionRefs ?? []
  const expectedBuild = options.expectedBuild?.trim() || undefined

  let state: ExperienceObservedBehavior['state'] = 'fresh'
  if (report.totals.sessions === 0 || sessions.length === 0) state = 'absent'
  else if (sessions.some((session) => !session.build) || builds.length !== 1) state = 'stale'
  else if (expectedBuild && !sessions.some((session) => session.build === expectedBuild)) state = 'stale'

  return {
    kind: 'experience-observed-behavior',
    schemaVersion: 'experience-observed-behavior/v1',
    state,
    scenario: report.scenario,
    build: builds.length === 1 ? builds[0]! : null,
    builds,
    sessions,
    ...(expectedBuild ? { expectedBuild } : {}),
    window: report.window,
    reportDigest: digest,
    totals: report.totals,
    flows: report.flows.slice(0, 8),
    routes: report.routes.slice(0, 8),
    friction: report.friction.slice(0, 8),
    journeys: report.journeys,
    recommendations: report.recommendations,
  }
}

/**
 * A compact block a redesign packet can show beside the capture. It carries no
 * fidelity label and makes no claim about pixels.
 */
export function createPacketObservedBlock(input: ExperienceFrictionReport): ExperiencePacketObservedBlock {
  const report = withDigest(input)
  const digest = report.reportDigest!
  const topFlow = report.flows[0] ?? null
  const topFriction = report.friction[0] ?? null
  const builds = report.builds.filter(Boolean)

  const parts: string[] = []
  if (topFlow) {
    parts.push(`${topFlow.steps.join(' -> ')}: ${topFlow.count}x, median ${Math.round(topFlow.medianDurationMs / 1000)}s`)
  }
  if (topFriction) {
    parts.push(`${topFriction.count} ${topFriction.kind.replace(/_/g, ' ')} signal(s)`)
  }
  if (report.totals.returns > 0) {
    parts.push(`${report.totals.returns} return(s) after ${Math.round(report.totals.awayMs / 1000)}s away`)
  }
  parts.push(`${report.totals.sessions} observed session(s)`)

  return {
    schemaVersion: 'experience-packet-observed/v1',
    scenario: report.scenario,
    generatedAt: report.generatedAt,
    build: builds.length === 1 ? builds[0]! : null,
    sessions: (report.sessionRefs ?? []).map(({ id, build }) => ({ id, build })),
    window: report.window,
    headline: parts.join('; '),
    topFlow,
    topFriction,
    timeCostMs: topFlow ? topFlow.totalDurationMs : report.totals.activeMs,
    reportDigest: digest,
    caveat: 'Observed locally from the operator\'s own use. Advisory only: not a capture, not a verdict, and not proof that the flow is correct.',
  }
}

/** Lightweight structural validation for a report read from disk. */
export function validateReport(value: unknown): { ok: true; report: ExperienceFrictionReport; digestValid: boolean } | { ok: false; errors: string[] } {
  if (!value || typeof value !== 'object') return { ok: false, errors: ['report must be an object'] }
  const report = value as Partial<ExperienceFrictionReport>
  const errors: string[] = []
  if (report.schemaVersion !== 'experience-friction-report/v1') errors.push('unsupported schemaVersion')
  if (typeof report.scenario !== 'string' || !report.scenario) errors.push('scenario is required')
  if (typeof report.generatedAt !== 'string' || !report.generatedAt) errors.push('generatedAt is required')
  if (!report.totals || typeof report.totals.sessions !== 'number') errors.push('totals.sessions is required')
  if (!Array.isArray(report.routes)) errors.push('routes must be an array')
  if (!Array.isArray(report.flows)) errors.push('flows must be an array')
  if (!Array.isArray(report.friction)) errors.push('friction must be an array')
  if (!Array.isArray(report.recommendations)) errors.push('recommendations must be an array')
  if (!Array.isArray(report.builds)) errors.push('builds must be an array')
  if (!Array.isArray(report.sessionRefs)) errors.push('sessionRefs must be an array')
  else {
    const refs = report.sessionRefs as unknown[]
    if (refs.some((item) => !item || typeof item !== 'object' || typeof (item as Record<string, unknown>).id !== 'string' || !/^[a-zA-Z0-9_-]{1,128}$/.test(String((item as Record<string, unknown>).id || '')) || !((item as Record<string, unknown>).build === null || typeof (item as Record<string, unknown>).build === 'string') || typeof (item as Record<string, unknown>).startedAt !== 'string' || typeof (item as Record<string, unknown>).endedAt !== 'string')) errors.push('sessionRefs contain a malformed session binding')
    if (report.totals && refs.length !== report.totals.sessions) errors.push('sessionRefs count does not match totals.sessions')
  }
  if (errors.length > 0) return { ok: false, errors }
  const digestValid = report.reportDigest ? report.reportDigest === computeReportDigest(report as ExperienceFrictionReport) : false
  return { ok: true, report: report as ExperienceFrictionReport, digestValid }
}
