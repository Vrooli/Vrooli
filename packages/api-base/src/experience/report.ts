/**
 * Offline analysis of recorded experience sessions.
 *
 * The analyzer is deliberately dependency-free and deterministic. It answers
 * "where does time go, what do I repeat, where do I stall, and what never gets
 * used", then emits machine-readable findings a person or an agent can turn
 * into a redesign. It does not guess intent; hesitation is an inference from
 * timing and pointer reversals and is labelled as such.
 */

import * as fs from 'node:fs'
import * as path from 'node:path'
import { computeReportDigest } from './contracts.js'
import {
  type ExperienceActionEvent,
  type ExperienceBatch,
  type ExperienceCapabilityDeclaration,
  type ExperienceCapabilityMetric,
  type ExperienceControlMetric,
  type ExperienceElementRef,
  type ExperienceFieldMetric,
  type ExperienceFlowMetric,
  type ExperienceFrictionMetric,
  type ExperienceFrictionReport,
  type ExperienceJourneyDiff,
  type ExperienceObservedFlow,
  type ExperienceRecommendation,
  type ExperienceRouteMetric,
  type ExperienceSession,
  type ExperienceTotals,
} from './types.js'

interface FlowToken {
  label: string
  display: string
  ts: number
  route: string
  element?: ExperienceElementRef
}

interface FlowAccumulator {
  steps: string[]
  count: number
  durations: number[]
}

interface DeclaredJourney {
  id: string
  title: string
  pages: string[]
}

interface DeclaredSpec {
  routesByPage: Map<string, string[]>
  journeys: DeclaredJourney[]
}

export interface AnalyzeOptions {
  scenario: string
  /** Directory containing experience/index.json, or a scenario root. */
  specDir?: string
  /**
   * Capabilities the operator is expected to use (from the scenario policy).
   * One with no observed clicks is reported as unobserved, so an alternative
   * path that exists but is not discovered becomes visible.
   */
  capabilities?: ExperienceCapabilityDeclaration[]
  minFlowCount?: number
  maxFlows?: number
  now?: number
}

export interface AnalyzeResult {
  report: ExperienceFrictionReport
  observedFlows: ExperienceObservedFlow[]
}

function round(value: number): number {
  return Math.round(value)
}

function median(values: number[]): number | null {
  if (values.length === 0) return null
  const sorted = [...values].sort((a, b) => a - b)
  const middle = Math.floor(sorted.length / 2)
  if (sorted.length % 2 === 1) return sorted[middle]!
  return (sorted[middle - 1]! + sorted[middle]!) / 2
}

function ratio(numerator: number, denominator: number): number {
  return denominator === 0 ? 0 : numerator / denominator
}

function hash(input: string): string {
  let h = 0x811c9dc5
  for (let i = 0; i < input.length; i += 1) {
    h ^= input.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  return (h >>> 0).toString(36)
}

/** Group append-only batches into sessions and de-duplicate retried events. */
export function buildSessions(batches: ExperienceBatch[]): ExperienceSession[] {
  const byId = new Map<string, ExperienceSession>()
  const seen = new Map<string, Set<string>>()

  batches.forEach((batch) => {
    let session = byId.get(batch.sessionId)
    if (!session) {
      session = {
        id: batch.sessionId,
        scenario: batch.scenario,
        ...(batch.build ? { build: batch.build } : {}),
        startedAt: Number.POSITIVE_INFINITY,
        endedAt: 0,
        events: [],
      }
      byId.set(batch.sessionId, session)
      seen.set(batch.sessionId, new Set())
    }
    const keys = seen.get(batch.sessionId)!
    batch.events.forEach((event) => {
      const key = `${event.ts}:${event.seq}:${event.type}`
      if (keys.has(key)) return
      keys.add(key)
      session!.events.push(event)
      if (event.ts < session!.startedAt) session!.startedAt = event.ts
      if (event.ts > session!.endedAt) session!.endedAt = event.ts
    })
  })

  return Array.from(byId.values())
    .filter((session) => Number.isFinite(session.startedAt) && session.events.length > 0)
    .map((session) => ({ ...session, events: [...session.events].sort((a, b) => a.ts - b.ts || a.seq - b.seq) }))
    .sort((a, b) => a.startedAt - b.startedAt)
}

function readJson(file: string): unknown {
  try {
    return JSON.parse(fs.readFileSync(file, 'utf-8'))
  } catch {
    return null
  }
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' ? value as Record<string, unknown> : null
}

function stringArray(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : []
}

/**
 * Read a scenario's experience spec. Accepts the experience directory or its
 * parent. Missing or malformed files degrade to "no declared journeys" rather
 * than failing the report.
 */
export function loadDeclaredSpec(specDir: string | undefined): DeclaredSpec {
  const spec: DeclaredSpec = { routesByPage: new Map(), journeys: [] }
  if (!specDir) return spec

  const candidates = [specDir, path.join(specDir, 'experience')]
  const root = candidates.find((candidate) => fs.existsSync(path.join(candidate, 'index.json')))
  if (!root) return spec

  const index = asRecord(readJson(path.join(root, 'index.json')))
  const pages = Array.isArray(index?.pages) ? index!.pages : []
  pages.forEach((entry) => {
    const ref = asRecord(entry)
    if (!ref || typeof ref.id !== 'string' || typeof ref.path !== 'string') return
    const page = asRecord(readJson(path.join(root, ref.path)))
    const identity = asRecord(page?.page)
    const routes = stringArray(identity?.routes)
    if (routes.length > 0) spec.routesByPage.set(ref.id, routes)
  })

  const journeyFiles = new Set<string>()
  const indexJourneys = Array.isArray(index?.journeys) ? index!.journeys : []
  indexJourneys.forEach((entry) => {
    const ref = asRecord(entry)
    if (ref && typeof ref.path === 'string') journeyFiles.add(ref.path)
  })
  try {
    fs.readdirSync(path.join(root, 'journeys'))
      .filter((name) => name.endsWith('.json'))
      .forEach((name) => journeyFiles.add(path.join('journeys', name)))
  } catch {
    // No journeys directory.
  }

  journeyFiles.forEach((relative) => {
    const document = asRecord(readJson(path.join(root, relative)))
    const journey = asRecord(document?.journey)
    if (!journey || typeof journey.id !== 'string') return
    const steps = Array.isArray(document?.steps) ? document!.steps : []
    const pages = steps
      .map((step) => asRecord(step)?.page)
      .filter((page): page is string => typeof page === 'string')
    if (pages.length === 0) return
    spec.journeys.push({
      id: journey.id,
      title: typeof journey.title === 'string' ? journey.title : journey.id,
      pages,
    })
  })

  return spec
}

/** Match a declared route pattern (`/repo/:id`, `*`) against an observed path. */
export function routeMatches(pattern: string, route: string): boolean {
  const cleanPattern = pattern.split('?')[0] || '/'
  const cleanRoute = route.split('?')[0] || '/'
  if (cleanPattern === cleanRoute || cleanPattern === '*') return true
  const patternParts = cleanPattern.split('/').filter(Boolean)
  const routeParts = cleanRoute.split('/').filter(Boolean)
  if (patternParts.length !== routeParts.length) return false
  for (let i = 0; i < patternParts.length; i += 1) {
    const expected = patternParts[i]!
    const actual = routeParts[i]!
    if (expected === '*') return true
    if (expected.startsWith(':')) continue
    if (expected !== actual) return false
  }
  return true
}

function displayElement(element: ExperienceElementRef | undefined): string {
  if (!element) return 'unknown'
  return element.testId || element.name || element.role || element.fp
}

function sessionsForJourney(
  pages: string[],
  routesByPage: Map<string, string[]>,
  session: ExperienceSession,
): { matched: boolean; start: number; end: number } | null {
  const routeTimes: Array<{ route: string; ts: number }> = []
  session.events.forEach((event) => {
    if (event.type === 'route') routeTimes.push({ route: (event as { route: string }).route, ts: event.ts })
  })
  if (routeTimes.length === 0) return null

  let cursor = 0
  let start = 0
  let end = 0
  for (const page of pages) {
    const patterns = routesByPage.get(page)
    if (!patterns || patterns.length === 0) return null
    let found = false
    for (; cursor < routeTimes.length; cursor += 1) {
      const candidate = routeTimes[cursor]!
      if (patterns.some((pattern) => routeMatches(pattern, candidate.route))) {
        found = true
        if (start === 0) start = candidate.ts
        end = candidate.ts
        cursor += 1
        break
      }
    }
    if (!found) return null
  }
  return { matched: true, start, end }
}

export function analyzeExperience(batches: ExperienceBatch[], options: AnalyzeOptions): AnalyzeResult {
  const sessions = buildSessions(batches)
  const spec = loadDeclaredSpec(options.specDir)
  const minFlowCount = options.minFlowCount ?? 3
  const maxFlows = options.maxFlows ?? 12
  const generatedAt = new Date(options.now ?? Date.now()).toISOString()

  const routeStats = new Map<string, { visits: number; dwell: number[]; firstAction: number[]; actions: number; scroll: number[]; returns: number; away: number[] }>()
  const controlStats = new Map<string, { testId?: string; name?: string; role?: string; count: number; hover: number[]; gap: number[] }>()
  const frictionByKind = new Map<string, { count: number; routes: Map<string, number>; elements: Map<string, { testId?: string; name?: string; count: number }> }>()
  const fieldStats = new Map<string, { type: string; focuses: number; durations: number[]; edited: number; abandoned: number }>()
  const errorStats = new Map<string, { message: string; count: number; routes: Set<string> }>()
  const flows = new Map<string, FlowAccumulator>()
  let actionCount = 0
  let eventCount = 0
  let awayTotalMs = 0
  let returnsTotal = 0
  const foregroundMs: number[] = []

  const sessionsForFlows = sessions.map((session) => {
    const tokens: FlowToken[] = []
    let route = ''
    let routeEnteredAt = session.startedAt
    let routeActions = 0
    let firstActionRecorded = false
    let previousEventAt: number | null = null
    let previousTokenLabel: string | null = null
    let hiddenAt: number | null = null
    let sessionAwayMs = 0

    const closeRoute = (endTs: number, scrollDepth: number | undefined): void => {
      if (!route) return
      const entry = routeStats.get(route) || { visits: 0, dwell: [], firstAction: [], actions: 0, scroll: [], returns: 0, away: [] }
      entry.visits += 1
      entry.dwell.push(Math.max(0, endTs - routeEnteredAt))
      entry.actions += routeActions
      if (scrollDepth !== undefined && Number.isFinite(scrollDepth)) entry.scroll.push(scrollDepth)
      routeStats.set(route, entry)
    }

    session.events.forEach((event) => {
      eventCount += 1
      const currentRoute = route || '/'
      if (event.type === 'route') {
        const routeEvent = event as { route: string; scrollDepth?: number }
        const nextRoute = routeEvent.route || '/'
        if (nextRoute !== route) {
          closeRoute(event.ts, routeEvent.scrollDepth)
          route = nextRoute
          routeEnteredAt = event.ts
          routeActions = 0
          firstActionRecorded = false
        }
        const label = `route:${route}`
        if (previousTokenLabel !== label) {
          tokens.push({ label, display: route, ts: event.ts, route })
          previousTokenLabel = label
        }
        return
      }

      if (event.type === 'action') {
        const action = event as ExperienceActionEvent
        actionCount += 1
        routeActions += 1
        const element = action.element
        if (!firstActionRecorded) {
          const entry = routeStats.get(currentRoute) || { visits: 0, dwell: [], firstAction: [], actions: 0, scroll: [], returns: 0, away: [] }
          entry.firstAction.push(Math.max(0, event.ts - routeEnteredAt))
          routeStats.set(currentRoute, entry)
          firstActionRecorded = true
        }

        const control = controlStats.get(element.fp) || {
          ...(element.testId ? { testId: element.testId } : {}),
          ...(element.name ? { name: element.name } : {}),
          ...(element.role ? { role: element.role } : {}),
          count: 0,
          hover: [],
          gap: [],
        }
        control.count += 1
        if (typeof action.hoverMs === 'number' && action.hoverMs >= 0) control.hover.push(action.hoverMs)
        if (previousEventAt !== null) control.gap.push(Math.max(0, event.ts - previousEventAt))
        controlStats.set(element.fp, control)

        const label = `act:${element.fp}`
        if (previousTokenLabel !== label) {
          tokens.push({ label, display: displayElement(element), ts: event.ts, route: currentRoute, element })
          previousTokenLabel = label
        }
        previousEventAt = event.ts
        return
      }

      if (event.type === 'visibility') {
        // A hidden tab is the operator leaving the page; a visible event is the
        // return. Attribute the gap to whichever route was current so a slow
        // dialog can be distinguished from ordinary navigation.
        const visibility = event as { state: 'hidden' | 'visible' }
        if (visibility.state === 'hidden') {
          hiddenAt = event.ts
        } else if (hiddenAt !== null) {
          const away = Math.max(0, event.ts - hiddenAt)
          hiddenAt = null
          sessionAwayMs += away
          awayTotalMs += away
          returnsTotal += 1
          const entry = routeStats.get(currentRoute) || { visits: 0, dwell: [], firstAction: [], actions: 0, scroll: [], returns: 0, away: [] }
          entry.returns += 1
          entry.away.push(away)
          routeStats.set(currentRoute, entry)
        }
        return
      }

      if (event.type === 'friction') {
        const friction = event as { kind: string; element?: ExperienceElementRef; route?: string }
        const kind = friction.kind
        const bucket = frictionByKind.get(kind) || { count: 0, routes: new Map(), elements: new Map() }
        bucket.count += 1
        const frictionRoute = friction.route || currentRoute
        bucket.routes.set(frictionRoute, (bucket.routes.get(frictionRoute) || 0) + 1)
        if (friction.element) {
          const existing = bucket.elements.get(friction.element.fp) || {
            ...(friction.element.testId ? { testId: friction.element.testId } : {}),
            ...(friction.element.name ? { name: friction.element.name } : {}),
            count: 0,
          }
          existing.count += 1
          bucket.elements.set(friction.element.fp, existing)
        }
        frictionByKind.set(kind, bucket)
        return
      }

      if (event.type === 'field') {
        const fieldEvent = event as { phase: 'focus' | 'blur'; field: { type: string; name: string; durationMs?: number; edited?: boolean } }
        const key = `${fieldEvent.field.type}|${fieldEvent.field.name}`
        const bucket = fieldStats.get(key) || { type: fieldEvent.field.type, focuses: 0, durations: [], edited: 0, abandoned: 0 }
        if (fieldEvent.phase === 'focus') {
          bucket.focuses += 1
        } else {
          const duration = typeof fieldEvent.field.durationMs === 'number' ? Math.max(0, fieldEvent.field.durationMs) : 0
          bucket.durations.push(duration)
          if (fieldEvent.field.edited) bucket.edited += 1
          else if (duration >= 2000) bucket.abandoned += 1
        }
        fieldStats.set(key, bucket)
        return
      }

      if (event.type === 'error') {
        const error = event as { message: string }
        const bucket = errorStats.get(error.message) || { message: error.message, count: 0, routes: new Set<string>() }
        bucket.count += 1
        bucket.routes.add(currentRoute)
        errorStats.set(error.message, bucket)
      }
    })

    if (route) closeRoute(session.endedAt, undefined)

    foregroundMs.push(Math.max(0, session.endedAt - session.startedAt - sessionAwayMs))
    return { session, tokens: tokens.slice(0, 120) }
  })

  // n-gram flow mining (2..4 tokens), contiguous, per session.
  sessionsForFlows.forEach(({ tokens }) => {
    for (let size = 2; size <= 4; size += 1) {
      for (let start = 0; start + size <= tokens.length; start += 1) {
        const window = tokens.slice(start, start + size)
        const labels = window.map((token) => token.label)
        const key = labels.join('>')
        const accumulator = flows.get(key) || {
          steps: window.map((token) => token.display),
          count: 0,
          durations: [],
        }
        accumulator.count += 1
        accumulator.durations.push(Math.max(0, window[window.length - 1]!.ts - window[0]!.ts))
        flows.set(key, accumulator)
      }
    }
  })

  const flowMetrics: ExperienceFlowMetric[] = Array.from(flows.entries())
    .map(([key, value]) => ({
      id: hash(key),
      steps: value.steps,
      count: value.count,
      medianDurationMs: round(median(value.durations) ?? 0),
      totalDurationMs: round((median(value.durations) ?? 0) * value.count),
    }))
    .filter((flow) => flow.count >= minFlowCount)
    .sort((a, b) => b.totalDurationMs - a.totalDurationMs)

  // Drop a shorter n-gram when a longer one covers the same occurrences.
  const dedupedFlows: ExperienceFlowMetric[] = []
  flowMetrics.forEach((candidate) => {
    const contained = dedupedFlows.some((kept) => {
      if (kept.count < candidate.count) return false
      if (kept.steps.length <= candidate.steps.length) return false
      for (let start = 0; start + candidate.steps.length <= kept.steps.length; start += 1) {
        if (kept.steps.slice(start, start + candidate.steps.length).join('>') === candidate.steps.join('>')) return true
      }
      return false
    })
    if (!contained) dedupedFlows.push(candidate)
  })

  const topFlows = dedupedFlows.slice(0, maxFlows)

  // Declared journey coverage.
  const journeyDiffs: ExperienceJourneyDiff[] = spec.journeys.map((journey) => {
    let matched = 0
    const durations: number[] = []
    let friction = 0
    sessions.forEach((session) => {
      const result = sessionsForJourney(journey.pages, spec.routesByPage, session)
      if (!result) return
      matched += 1
      durations.push(Math.max(0, result.end - result.start))
      friction += session.events.filter((event) => {
        if (event.type !== 'friction') return false
        return event.ts >= result.start && event.ts <= result.end
      }).length
    })
    return {
      journeyId: journey.id,
      title: journey.title,
      declaredPages: journey.pages,
      sessionsMatched: matched,
      coverage: ratio(matched, sessions.length),
      medianDurationMs: median(durations) === null ? null : round(median(durations)!),
      friction,
    }
  })
  journeyDiffs.sort((a, b) => b.coverage - a.coverage)

  const declaredRouteOrder = new Set<string>()
  spec.journeys.forEach((journey) => {
    for (let i = 0; i + 1 < journey.pages.length; i += 1) {
      const from = spec.routesByPage.get(journey.pages[i]!) || []
      const to = spec.routesByPage.get(journey.pages[i + 1]!) || []
      from.forEach((a) => to.forEach((b) => declaredRouteOrder.add(`${a}>${b}`)))
    }
  })

  const undeclaredFlows = topFlows.filter((flow) => {
    const routeSteps = flow.steps.filter((step) => step.startsWith('/'))
    if (routeSteps.length < 2) return false
    for (let i = 0; i + 1 < routeSteps.length; i += 1) {
      const pair = `${routeSteps[i]}>${routeSteps[i + 1]}`
      if (declaredRouteOrder.has(pair)) return false
    }
    return true
  }).slice(0, 5)

  const routes: ExperienceRouteMetric[] = Array.from(routeStats.entries())
    .map(([route, value]) => ({
      route,
      visits: value.visits,
      dwellMs: round(value.dwell.reduce((sum, item) => sum + item, 0)),
      medianDwellMs: round(median(value.dwell) ?? 0),
      firstActionMs: median(value.firstAction) === null ? null : round(median(value.firstAction)!),
      actionsPerVisit: Number(ratio(value.actions, value.visits).toFixed(2)),
      scrollDepth: median(value.scroll) === null ? null : Number((median(value.scroll)!).toFixed(2)),
      returns: value.returns,
      awayMs: round(value.away.reduce((sum, item) => sum + item, 0)),
      medianAwayMs: median(value.away) === null ? null : round(median(value.away)!),
    }))
    .sort((a, b) => b.dwellMs - a.dwellMs)

  const controls: ExperienceControlMetric[] = Array.from(controlStats.entries())
    .map(([fp, value]) => {
      const frictionBucket = Array.from(frictionByKind.values())
        .reduce((sum, bucket) => sum + (bucket.elements.get(fp)?.count || 0), 0)
      return {
        fp,
        ...(value.testId ? { testId: value.testId } : {}),
        ...(value.name ? { name: value.name } : {}),
        ...(value.role ? { role: value.role } : {}),
        count: value.count,
        medianHoverMs: median(value.hover) === null ? null : round(median(value.hover)!),
        medianGapMs: median(value.gap) === null ? null : round(median(value.gap)!),
        friction: frictionBucket,
      }
    })
    .sort((a, b) => b.count - a.count)

  const capabilities: ExperienceCapabilityMetric[] = (options.capabilities ?? []).map((capability) => {
    const matches = (control: ExperienceControlMetric): boolean => {
      if (capability.testId) return control.testId === capability.testId
      if (capability.name) return control.name === capability.name
      return false
    }
    const exercised = controls.filter(matches).reduce((sum, control) => sum + control.count, 0)
    return {
      id: capability.id,
      ...(capability.label ? { label: capability.label } : {}),
      ...(capability.testId ? { testId: capability.testId } : {}),
      ...(capability.name ? { name: capability.name } : {}),
      exercised,
      // One smoke load is not evidence that a capability is undiscovered.
      state: exercised > 0 ? 'exercised' : actionCount >= 10 ? 'unobserved' : 'insufficient-evidence',
    }
  })

  const friction: ExperienceFrictionMetric[] = Array.from(frictionByKind.entries())
    .map(([kind, value]) => ({
      kind,
      count: value.count,
      routes: Array.from(value.routes.entries())
        .map(([route, count]) => ({ route, count }))
        .sort((a, b) => b.count - a.count)
        .slice(0, 5),
      elements: Array.from(value.elements.entries())
        .map(([fp, item]) => ({ fp, ...item }))
        .sort((a, b) => b.count - a.count)
        .slice(0, 5),
    }))
    .sort((a, b) => b.count - a.count)

  // Leaving and returning is a signal, not a defect: a wait long enough that the
  // operator does something else and comes back. It is derived from visibility
  // events rather than a recorded click, so it is added after the event buckets.
  const returnRoutes = routes
    .filter((route) => route.returns >= 2 && route.medianAwayMs !== null && route.medianAwayMs >= 15000)
    .sort((a, b) => b.awayMs - a.awayMs)
  if (returnRoutes.length > 0) {
    friction.push({
      kind: 'leave_and_return',
      count: returnRoutes.reduce((sum, route) => sum + route.returns, 0),
      routes: returnRoutes.map((route) => ({ route: route.route, count: route.returns })).slice(0, 5),
      elements: [],
    })
    friction.sort((a, b) => b.count - a.count)
  }

  const fields: ExperienceFieldMetric[] = Array.from(fieldStats.entries())
    .map(([key, value]) => {
      const [, name] = key.split('|')
      return {
        name: name || key,
        type: value.type,
        focuses: value.focuses,
        medianDurationMs: round(median(value.durations) ?? 0),
        editRate: Number(ratio(value.edited, value.durations.length).toFixed(2)),
        abandonmentRate: Number(ratio(value.abandoned, value.durations.length).toFixed(2)),
      }
    })
    .sort((a, b) => b.focuses - a.focuses)

  const errors = Array.from(errorStats.values())
    .map((value) => ({ message: value.message, count: value.count, routes: Array.from(value.routes) }))
    .sort((a, b) => b.count - a.count)

  const totals: ExperienceTotals = {
    sessions: sessions.length,
    events: eventCount,
    actions: actionCount,
    activeMs: foregroundMs.reduce((sum, item) => sum + item, 0),
    awayMs: awayTotalMs,
    returns: returnsTotal,
    medianSessionMs: round(median(foregroundMs) ?? 0),
    routes: routeStats.size,
  }

  const recommendations = buildRecommendations({ routes, controls, fields, friction, journeyDiffs, undeclaredFlows, errors, totals, flows: topFlows, capabilities })
  const builds = Array.from(new Set(sessions.map((session) => session.build).filter((build): build is string => Boolean(build)))).sort()
  const notes: string[] = []
  if (sessions.length < 3) {
    notes.push('Fewer than three sessions: treat medians as anecdotal, not as a trend.')
  }
  notes.push('Hesitation is inferred from timing, hover, and pointer reversals. It locates friction; it does not explain it.')
  notes.push('Leaving and returning is inferred from tab visibility. A hidden tab locates a wait; it does not explain why the operator stopped waiting.')
  notes.push('Rank by total time cost (frequency x duration). With one operator, statistical significance is not the point.')
  if (spec.journeys.length === 0) {
    notes.push('No declared journeys were found. Pass --spec <scenario>/experience to compare declared and observed flows.')
  }

  const report: ExperienceFrictionReport = {
    schemaVersion: 'experience-friction-report/v1',
    scenario: options.scenario,
    generatedAt,
    window: {
      from: sessions.length > 0 ? new Date(sessions[0]!.startedAt).toISOString() : null,
      to: sessions.length > 0 ? new Date(sessions[sessions.length - 1]!.endedAt).toISOString() : null,
      sessions: sessions.length,
    },
    builds,
    sessionRefs: sessions.map((session) => ({
      id: session.id,
      build: session.build ?? null,
      startedAt: new Date(session.startedAt).toISOString(),
      endedAt: new Date(session.endedAt).toISOString(),
    })),
    totals,
    routes,
    controls,
    flows: topFlows,
    friction,
    fields,
    journeys: journeyDiffs,
    undeclaredFlows,
    capabilities,
    errors,
    recommendations,
    notes,
  }
  const reportWithDigest: ExperienceFrictionReport = { ...report, reportDigest: computeReportDigest(report) }

  const observedFlows: ExperienceObservedFlow[] = topFlows.map((flow) => ({
    kind: 'experience-observed-flow',
    schemaVersion: 'experience-observed-flow/v1',
    scenario: options.scenario,
    generatedAt,
    flow: {
      id: flow.id,
      steps: flow.steps.map((step) => (step.startsWith('/') ? { route: step } : { route: '', element: step })),
      count: flow.count,
      medianDurationMs: flow.medianDurationMs,
    },
  }))

  return { report: reportWithDigest, observedFlows }
}

function buildRecommendations(input: {
  routes: ExperienceRouteMetric[]
  controls: ExperienceControlMetric[]
  fields: ExperienceFieldMetric[]
  friction: ExperienceFrictionMetric[]
  journeyDiffs: ExperienceJourneyDiff[]
  undeclaredFlows: ExperienceFlowMetric[]
  capabilities: ExperienceCapabilityMetric[]
  errors: Array<{ message: string; count: number; routes: string[] }>
  totals: ExperienceTotals
  flows: ExperienceFlowMetric[]
}): ExperienceRecommendation[] {
  const recommendations: ExperienceRecommendation[] = []
  const friction = (kind: string): ExperienceFrictionMetric | undefined => input.friction.find((item) => item.kind === kind)

  const deadClicks = friction('dead_click')
  if (deadClicks && deadClicks.count >= 2) {
    const first = deadClicks.elements[0]
    recommendations.push({
      id: 'dead-clicks',
      severity: 'high',
      title: `Dead clicks on ${first ? (first.testId || first.name || first.fp) : 'several controls'}`,
      evidence: `${deadClicks.count} clicks produced no route, DOM, or network change.`,
      suggestion: 'Check the affordance: either the control looks clickable but is inert, or the response is slow enough to read as a miss. Prefer an immediate inline state over a silent wait.',
    })
  }

  const rage = friction('rage_click')
  if (rage && rage.count >= 1) {
    recommendations.push({
      id: 'rage-clicks',
      severity: 'medium',
      title: 'Repeated clicking on the same control',
      evidence: `${rage.count} burst(s) of three or more clicks within 1.5s.`,
      suggestion: 'Make the pending state visible and disable repeat submission, or remove the wait entirely.',
    })
  }

  const repeated = friction('repeated_action')
  if (repeated && repeated.count >= 1) {
    recommendations.push({
      id: 'repeated-actions',
      severity: 'medium',
      title: 'The same control is used four or more times in ten seconds',
      evidence: `${repeated.count} repetition burst(s).`,
      suggestion: 'Offer a batch action, a persistent default, or a keyboard shortcut for this loop.',
    })
  }

  const leaveReturn = friction('leave_and_return')
  if (leaveReturn) {
    const topRoute = leaveReturn.routes[0]?.route
    const metric = input.routes.find((route) => route.route === topRoute)
    const awaySeconds = metric?.medianAwayMs ? Math.round(metric.medianAwayMs / 1000) : null
    recommendations.push({
      id: 'leave-and-return',
      severity: 'high',
      title: `The operator leaves ${topRoute ?? 'a route'} and comes back`,
      evidence: `${leaveReturn.count} hide/return cycle(s)${awaySeconds ? `, median ${awaySeconds}s away` : ''}${metric?.awayMs ? `, ${Math.round(metric.awayMs / 1000)}s away in total` : ''}.`,
      suggestion: 'A wait long enough to leave is longer than the work available on screen. Make progress visible, and let unrelated work continue while it runs. Then check whether the alternative is discoverable: a capability that exists but is never used is its own finding.',
    })
  }

  const slow = input.routes.find((route) => route.medianDwellMs >= 15000 && route.actionsPerVisit < 1 && route.visits >= 2)
  if (slow) {
    recommendations.push({
      id: 'dwell-without-action',
      severity: 'medium',
      title: `Long dwell with few actions at ${slow.route}`,
      evidence: `Median dwell ${Math.round(slow.medianDwellMs / 1000)}s across ${slow.visits} visits with ${slow.actionsPerVisit} actions per visit.`,
      suggestion: 'The page is being read, not operated. Check whether the next step is visible or whether the information needed to act is present.',
    })
  }

  const unusedCapabilities = input.capabilities.filter((capability) => capability.state === 'unobserved')
  if (unusedCapabilities.length > 0) {
    recommendations.push({
      id: 'capability-unobserved',
      severity: 'medium',
      title: `Declared but unused: ${unusedCapabilities.map((capability) => capability.label || capability.id).join(', ')}`,
      evidence: `${unusedCapabilities.length} declared capability(ies) recorded no clicks across ${input.totals.actions} actions.`,
      suggestion: 'An available path that is never used is a finding about discovery, not proof the path is wrong. Check whether it is offered at the moment it would help, whether its label says what it does, and whether using it costs the operator anything.',
    })
  }

  const abandonedField = input.fields.find((field) => field.focuses >= 4 && field.abandonmentRate >= 0.25)
  if (abandonedField) {
    recommendations.push({
      id: 'field-abandonment',
      severity: 'medium',
      title: `Field "${abandonedField.name}" is focused but not filled`,
      evidence: `${Math.round(abandonedField.abandonmentRate * 100)}% of ${abandonedField.focuses} focuses ended without input.`,
      suggestion: 'Check validation timing, required-field clarity, and whether the expected format is obvious before typing.',
    })
  }

  if (input.undeclaredFlows.length > 0) {
    const flow = input.undeclaredFlows[0]!
    recommendations.push({
      id: 'undeclared-flow',
      severity: 'high',
      title: 'A frequent flow is not declared anywhere',
      evidence: `${flow.count} occurrences of ${flow.steps.join(' -> ')} (median ${Math.round(flow.medianDurationMs / 1000)}s).`,
      suggestion: 'Either promote the flow into an explicit journey and shorten it, or fix the affordance that makes this detour necessary.',
    })
  }

  const unobserved = input.journeyDiffs.find((journey) => journey.sessionsMatched === 0 && input.totals.sessions >= 3)
  if (unobserved) {
    recommendations.push({
      id: 'declared-unobserved',
      severity: 'medium',
      title: `Declared journey "${unobserved.title}" was never observed`,
      evidence: `0 of ${input.totals.sessions} sessions followed ${unobserved.declaredPages.join(' -> ')}.`,
      suggestion: 'Verify discoverability of the first step, or retire the journey if the work no longer happens that way.',
    })
  }

  const repeatedFlow = input.flows[0]
  if (repeatedFlow && repeatedFlow.count >= 4) {
    recommendations.push({
      id: 'repeated-flow',
      severity: 'medium',
      title: 'A flow is repeated often enough to deserve a shortcut',
      evidence: `${repeatedFlow.count} occurrences of ${repeatedFlow.steps.join(' -> ')}.`,
      suggestion: 'Consider a command-palette action, a bulk control, or a remembered default.',
    })
  }

  if (input.errors.length > 0) {
    const error = input.errors[0]!
    recommendations.push({
      id: 'runtime-errors',
      severity: 'medium',
      title: 'Runtime errors occurred during the recorded window',
      evidence: `${error.count}x "${error.message.slice(0, 120)}" on ${error.routes.join(', ')}.`,
      suggestion: 'Fix the error before tuning the interaction; an error path dominates any layout change.',
    })
  }

  const severityRank = { high: 0, medium: 1, low: 2 } as const
  return recommendations.sort((a, b) => severityRank[a.severity] - severityRank[b.severity]).slice(0, 8)
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${round(ms)}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

/** Render the report as a reviewable Markdown document. */
export function renderFrictionMarkdown(report: ExperienceFrictionReport): string {
  const lines: string[] = []
  lines.push(`# Experience friction: ${report.scenario}`)
  lines.push('')
  lines.push(`Generated ${report.generatedAt}. Window ${report.window.from ?? '-'} to ${report.window.to ?? '-'}.`)
  lines.push('')
  lines.push(`Build: ${report.builds.length === 0 ? 'unknown' : report.builds.join(', ')}. Report digest: ${report.reportDigest ?? '-'}.`)
  lines.push('')
  lines.push(`**${report.totals.sessions} sessions**, ${report.totals.actions} actions, ${formatDuration(report.totals.activeMs)} active${report.totals.awayMs > 0 ? `, ${formatDuration(report.totals.awayMs)} away across ${report.totals.returns} return(s)` : ''}, ${report.totals.routes} routes.`)
  lines.push('')

  lines.push('## What to fix first')
  lines.push('')
  if (report.recommendations.length === 0) {
    lines.push('No recommendation cleared its threshold. Record more sessions or lower the thresholds.')
  } else {
    report.recommendations.forEach((recommendation, index) => {
      lines.push(`${index + 1}. **${recommendation.title}** (${recommendation.severity})`)
      lines.push(`   - Evidence: ${recommendation.evidence}`)
      lines.push(`   - Suggestion: ${recommendation.suggestion}`)
    })
  }
  lines.push('')

  if (report.flows.length > 0) {
    lines.push('## Most expensive repeated flows')
    lines.push('')
    lines.push('| Count | Median | Flow |')
    lines.push('| --- | --- | --- |')
    report.flows.forEach((flow) => {
      lines.push(`| ${flow.count} | ${formatDuration(flow.medianDurationMs)} | ${flow.steps.join(' -> ')} |`)
    })
    lines.push('')
  }

  if (report.routes.length > 0) {
    lines.push('## Where time goes')
    lines.push('')
    lines.push('| Route | Visits | Median dwell | First action | Actions/visit |')
    lines.push('| --- | --- | --- | --- | --- |')
    report.routes.slice(0, 10).forEach((route) => {
      lines.push(`| ${route.route} | ${route.visits} | ${formatDuration(route.medianDwellMs)} | ${route.firstActionMs === null ? '-' : formatDuration(route.firstActionMs)} | ${route.actionsPerVisit} |`)
    })
    lines.push('')
  }

  if (report.capabilities.length > 0) {
    lines.push('## Declared capabilities')
    lines.push('')
    lines.push('| Capability | Observed uses | State |')
    lines.push('| --- | --- | --- |')
    report.capabilities.forEach((capability) => {
      lines.push(`| ${capability.label || capability.id} | ${capability.exercised} | ${capability.state} |`)
    })
    lines.push('')
    lines.push('An unobserved capability is a discovery question, not a defect and not proof the capability is wrong.')
    lines.push('')
  }

  if (report.routes.some((route) => route.returns > 0)) {
    lines.push('## Leaving and returning')
    lines.push('')
    lines.push('| Route | Returns | Median away | Total away |')
    lines.push('| --- | --- | --- | --- |')
    report.routes
      .filter((route) => route.returns > 0)
      .slice(0, 10)
      .forEach((route) => {
        lines.push(`| ${route.route} | ${route.returns} | ${route.medianAwayMs === null ? '-' : formatDuration(route.medianAwayMs)} | ${formatDuration(route.awayMs)} |`)
      })
    lines.push('')
    lines.push('A hidden tab means the operator chose not to wait on screen. It locates a wait; it does not say what caused it.')
    lines.push('')
  }

  if (report.controls.length > 0) {
    lines.push('## Most used controls')
    lines.push('')
    lines.push('| Uses | Control | Median hover | Median gap after | Friction |')
    lines.push('| --- | --- | --- | --- | --- |')
    report.controls.slice(0, 12).forEach((control) => {
      const label = control.testId || control.name || control.role || control.fp
      lines.push(`| ${control.count} | ${label} | ${control.medianHoverMs === null ? '-' : formatDuration(control.medianHoverMs)} | ${control.medianGapMs === null ? '-' : formatDuration(control.medianGapMs)} | ${control.friction} |`)
    })
    lines.push('')
  }

  if (report.friction.length > 0) {
    lines.push('## Friction signals')
    lines.push('')
    report.friction.forEach((item) => {
      const routes = item.routes.map((entry) => `${entry.route} (${entry.count})`).join(', ')
      const elements = item.elements.map((entry) => `${entry.testId || entry.name || entry.fp} (${entry.count})`).join(', ')
      lines.push(`- **${item.kind}**: ${item.count}${routes ? ` on ${routes}` : ''}${elements ? `; ${elements}` : ''}`)
    })
    lines.push('')
  }

  if (report.fields.length > 0) {
    lines.push('## Fields')
    lines.push('')
    lines.push('| Field | Type | Focuses | Median focus | Edit rate | Abandonment |')
    lines.push('| --- | --- | --- | --- | --- | --- |')
    report.fields.slice(0, 10).forEach((field) => {
      lines.push(`| ${field.name} | ${field.type} | ${field.focuses} | ${formatDuration(field.medianDurationMs)} | ${Math.round(field.editRate * 100)}% | ${Math.round(field.abandonmentRate * 100)}% |`)
    })
    lines.push('')
  }

  if (report.journeys.length > 0) {
    lines.push('## Declared journeys')
    lines.push('')
    lines.push('| Journey | Coverage | Median duration | Friction |')
    lines.push('| --- | --- | --- | --- |')
    report.journeys.forEach((journey) => {
      lines.push(`| ${journey.title} | ${journey.sessionsMatched}/${report.totals.sessions} (${Math.round(journey.coverage * 100)}%) | ${journey.medianDurationMs === null ? '-' : formatDuration(journey.medianDurationMs)} | ${journey.friction} |`)
    })
    lines.push('')
  }

  if (report.errors.length > 0) {
    lines.push('## Runtime errors')
    lines.push('')
    report.errors.forEach((error) => {
      lines.push(`- ${error.count}x ${error.message}${error.routes.length > 0 ? ` (${error.routes.join(', ')})` : ''}`)
    })
    lines.push('')
  }

  if (report.notes.length > 0) {
    lines.push('## How to read this')
    lines.push('')
    report.notes.forEach((note) => lines.push(`- ${note}`))
    lines.push('')
  }

  return `${lines.join('\n')}\n`
}
