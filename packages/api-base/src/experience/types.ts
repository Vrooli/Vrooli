/**
 * Experience telemetry contract.
 *
 * Local-only interaction telemetry for scenario UIs. The event vocabulary is
 * deliberately close to Browser Automation Studio's recorded actions so a
 * session can be read by the same reasoning that reads a BAS recording, and so
 * a frequently observed flow can later be promoted into a workflow.
 *
 * Nothing here records input values, clipboard contents, IP addresses, or user
 * agents. Identity is the semantic element (test id / role / accessible name),
 * not the person.
 */

/** Wire version for the browser -> server batch envelope. */
export const EXPERIENCE_PROTOCOL_VERSION = 1

/** Attribute that marks the injected agent script and makes injection idempotent. */
export const EXPERIENCE_AGENT_ATTRIBUTE = 'data-vrooli-experience'

/** Default paths. Kept in one place so the injector, router, and docs agree. */
export const EXPERIENCE_DEFAULTS = {
  assetPath: '/__vrooli/experience/agent.js',
  ingestPath: '/__vrooli/experience/events',
  statusPath: '/__vrooli/experience/status',
  maxBodyBytes: 256 * 1024,
  maxEventsPerBatch: 250,
  maxEventsPerSession: 5000,
  maxAgeDays: 30,
  maxSessions: 500,
  sample: 1,
  flushIntervalMs: 2500,
  batchSize: 20,
} as const

/** A stable reference to the element a user acted on. */
export interface ExperienceElementRef {
  /** Value of the nearest `data-testid`, when present. */
  testId?: string
  /** Explicit or implicit ARIA role. */
  role?: string
  /** Accessible-name approximation, capped in length. Never an input value. */
  name?: string
  /** Lower-case tag name. */
  tag: string
  /** Structural fallback selector, capped in length. */
  path: string
  /** Short grouping fingerprint over role/name/testId/path. */
  fp: string
  /** Nearest declared experience region, when annotated. */
  region?: string
  /** Nearest declared experience surface, when annotated. */
  surface?: string
  /** Whether the element was disabled at interaction time. */
  disabled?: boolean
  /** Viewport-relative bounding box, rounded to whole pixels. */
  rect?: { x: number; y: number; width: number; height: number }
  /** Whether any part of the element was inside the viewport. */
  inViewport?: boolean
}

export interface ExperiencePointerRef {
  x: number
  y: number
}

/** Pointer-derived hesitation features. Raw movement is never stored. */
export interface ExperiencePointerFeatures {
  /** Distance travelled in the second before the action. */
  travelPx: number
  /** Direction reversals over 90 degrees with a meaningful segment. */
  reversals: number
  /** Pixels per millisecond over the sampled window. */
  speedPxPerMs: number
}

interface ExperienceEventBase {
  /** Event type discriminator. */
  type: string
  /** Epoch milliseconds at capture time. */
  ts: number
  /** Monotonic per-session sequence number. */
  seq: number
}

export interface ExperienceSessionStartEvent extends ExperienceEventBase {
  type: 'session_start'
  route: string
  viewport: { width: number; height: number; dpr: number }
  reducedMotion: boolean
  theme: 'light' | 'dark' | 'unknown'
  /** Frame position. Same-origin frames share sessionStorage and therefore the session id. */
  frame: { depth: number; top: boolean }
}

export interface ExperienceSessionEndEvent extends ExperienceEventBase {
  type: 'session_end'
  activeMs: number
  reason: 'hidden' | 'unload' | 'disabled' | 'manual'
}

export interface ExperienceRouteEvent extends ExperienceEventBase {
  type: 'route'
  route: string
  from?: string
  via: 'load' | 'push' | 'replace' | 'pop' | 'hash'
  /** Deepest scroll position reached on the route being left, 0..1. */
  scrollDepth?: number
}

export interface ExperienceActionEvent extends ExperienceEventBase {
  type: 'action'
  action: 'click' | 'keyboard' | 'submit' | 'change'
  element: ExperienceElementRef
  cursor?: ExperiencePointerRef
  /** Milliseconds the pointer rested on the element before activating it. */
  hoverMs?: number
  pointer?: ExperiencePointerFeatures
  /** Milliseconds since the previous event; filled in by analysis, not capture. */
  gapMs?: number
}

export interface ExperienceKeyEvent extends ExperienceEventBase {
  type: 'key'
  /** Navigation/shortcut category only. Printable characters are never sent. */
  key: string
  element?: ExperienceElementRef
}

export interface ExperienceFieldEvent extends ExperienceEventBase {
  type: 'field'
  phase: 'focus' | 'blur'
  field: {
    type: string
    name: string
    sensitive: boolean
    /** Present on blur. */
    durationMs?: number
    /** Whether the field's value changed. Values are never captured. */
    edited?: boolean
  }
}

export type ExperienceFrictionKind =
  | 'dead_click'
  | 'rage_click'
  | 'repeated_action'
  | 'leave_and_return'
  | 'dismiss'
  | 'undo'
  | 'backtrack'
  | 'truncated'

export interface ExperienceFrictionEvent extends ExperienceEventBase {
  type: 'friction'
  kind: ExperienceFrictionKind
  element?: ExperienceElementRef
  route?: string
  detail?: Record<string, unknown>
}

export interface ExperienceErrorEvent extends ExperienceEventBase {
  type: 'error'
  kind: 'error' | 'unhandledrejection'
  message: string
  source?: string
  line?: number
}

export interface ExperiencePerformanceEvent extends ExperienceEventBase {
  type: 'performance'
  route: string
  metrics: {
    ttfbMs?: number
    domContentLoadedMs?: number
    loadMs?: number
    lcpMs?: number
    cls?: number
    longTasks?: number
    longTaskMs?: number
  }
}

export interface ExperienceVisibilityEvent extends ExperienceEventBase {
  type: 'visibility'
  state: 'visible' | 'hidden'
}

export type ExperienceEvent =
  | ExperienceSessionStartEvent
  | ExperienceSessionEndEvent
  | ExperienceRouteEvent
  | ExperienceActionEvent
  | ExperienceKeyEvent
  | ExperienceFieldEvent
  | ExperienceFrictionEvent
  | ExperienceErrorEvent
  | ExperiencePerformanceEvent
  | ExperienceVisibilityEvent

/**
 * One append-only line in a session file. The server stamps `scenario` and
 * `receivedAt`; the browser cannot spoof either.
 */
export interface ExperienceBatch {
  v: typeof EXPERIENCE_PROTOCOL_VERSION
  sessionId: string
  scenario: string
  receivedAt: number
  build?: string
  events: ExperienceEvent[]
}

/** Operator-facing configuration for local experience telemetry. */
export interface ExperienceTelemetryOptions {
  /** Force on/off. Defaults to the environment rule in `resolveExperienceTelemetry`. */
  enabled?: boolean
  /** Exact served build identity. Defaults to VROOLI_BUILD_IDENTITY. */
  build?: string
  /** Session directory. Defaults to ~/.vrooli/experience/sessions/<scenario>. */
  dataDir?: string
  /** Agent bundle to serve. Defaults to this package's dist copy; tests override it. */
  assetFile?: string
  /** URL path for the browser agent module. */
  assetPath?: string
  /** URL path for batched event ingestion. */
  ingestPath?: string
  /** URL path for a local status probe. */
  statusPath?: string
  /** Fraction of sessions to record, 0..1. */
  sample?: number
  /** Include accessible control names in captured events. */
  captureLabels?: boolean
  /** Collect navigation/timing/long-task metrics. */
  performance?: boolean
  /** Maximum accepted request body for ingestion. */
  maxBodyBytes?: number
  /** Per-session event cap enforced by the browser. */
  maxEventsPerSession?: number
  /** Delete session files older than this. */
  maxAgeDays?: number
  /** Keep at most this many session files per scenario. */
  maxSessions?: number
  /** Record requests whose Host is not loopback (for phone/tunnel testing). */
  allowRemoteHosts?: boolean
  /** Log injection and ingestion decisions. */
  verbose?: boolean
  /** Programmatic policy override; wins over the policy file. */
  policy?: ExperiencePolicy
  /** Policy file path; `false` disables policy-file lookup. */
  policyFile?: string | false
}

/**
 * Per-scenario policy, read from `.vrooli/experience.json` (or
 * `VROOLI_EXPERIENCE_POLICY`). Use it to opt a scenario out or tune sampling
 * without changing every start command.
 */
export interface ExperiencePolicy {
  enabled?: boolean
  sample?: number
  captureLabels?: boolean
  performance?: boolean
  allowRemoteHosts?: boolean
  maxAgeDays?: number
  maxSessions?: number
  /**
   * Capabilities the scenario expects an operator to use. The analyzer reports
   * one that is never exercised, so an alternative path that exists but is not
   * discovered becomes visible instead of silently unused.
   */
  capabilities?: ExperienceCapabilityDeclaration[]
}

export interface ExperienceCapabilityDeclaration {
  id: string
  label?: string
  /** Match by the control's `data-testid`. */
  testId?: string
  /** Match by the accessible name. */
  name?: string
}

export interface ExperienceCapabilityMetric {
  id: string
  label?: string
  testId?: string
  name?: string
  /** Number of observed clicks on the matching control. */
  exercised: number
  state: 'exercised' | 'unobserved' | 'insufficient-evidence'
}

/** Fully resolved telemetry settings for a running server. */
export interface ResolvedExperienceTelemetry {
  enabled: boolean
  source: string
  scenario: string
  build?: string
  dataDir: string
  assetFile?: string
  assetPath: string
  ingestPath: string
  statusPath: string
  sample: number
  captureLabels: boolean
  performance: boolean
  maxBodyBytes: number
  maxEventsPerBatch: number
  maxEventsPerSession: number
  maxAgeDays: number
  maxSessions: number
  allowRemoteHosts: boolean
  verbose: boolean
}

/** Shape the browser posts. Scenario identity is server-owned. */
export interface ExperienceBatchRequest {
  v: number
  sessionId: string
  build?: string
  events: ExperienceEvent[]
}

export interface ExperienceSession {
  id: string
  scenario: string
  build?: string
  startedAt: number
  endedAt: number
  events: ExperienceEvent[]
}

export interface ExperienceTotals {
  sessions: number
  events: number
  actions: number
  /** Foreground time: session span minus time the tab was hidden. */
  activeMs: number
  /** Time the tab spent hidden. */
  awayMs: number
  /** Number of hidden -> visible transitions. */
  returns: number
  medianSessionMs: number
  routes: number
}

export interface ExperienceRouteMetric {
  route: string
  visits: number
  dwellMs: number
  medianDwellMs: number
  firstActionMs: number | null
  actionsPerVisit: number
  scrollDepth: number | null
  /** Times the operator hid the tab and came back while this route was current. */
  returns: number
  /** Total hidden time attributed to this route. */
  awayMs: number
  /** Median hidden time per return, or null when the route had no returns. */
  medianAwayMs: number | null
}

export interface ExperienceControlMetric {
  fp: string
  testId?: string
  name?: string
  role?: string
  count: number
  medianHoverMs: number | null
  medianGapMs: number | null
  friction: number
}

export interface ExperienceFlowMetric {
  id: string
  steps: string[]
  count: number
  medianDurationMs: number
  totalDurationMs: number
}

export interface ExperienceFrictionMetric {
  kind: string
  count: number
  routes: Array<{ route: string; count: number }>
  elements: Array<{ fp: string; testId?: string; name?: string; count: number }>
}

export interface ExperienceFieldMetric {
  name: string
  type: string
  focuses: number
  medianDurationMs: number
  editRate: number
  abandonmentRate: number
}

export interface ExperienceJourneyDiff {
  journeyId: string
  title: string
  declaredPages: string[]
  sessionsMatched: number
  coverage: number
  medianDurationMs: number | null
  friction: number
}

export interface ExperienceRecommendation {
  id: string
  severity: 'high' | 'medium' | 'low'
  title: string
  evidence: string
  suggestion: string
}

export interface ExperienceFrictionReport {
  schemaVersion: 'experience-friction-report/v1'
  scenario: string
  generatedAt: string
  window: { from: string | null; to: string | null; sessions: number }
  /** Distinct served build identities observed in the window. */
  builds: string[]
  /** Per-session provenance only; never includes raw interaction events. */
  sessionRefs?: Array<{ id: string; build: string | null; startedAt: string; endedAt: string }>
  /** sha256 over the canonical report body (this field excluded). */
  reportDigest?: string
  totals: ExperienceTotals
  routes: ExperienceRouteMetric[]
  controls: ExperienceControlMetric[]
  flows: ExperienceFlowMetric[]
  friction: ExperienceFrictionMetric[]
  fields: ExperienceFieldMetric[]
  journeys: ExperienceJourneyDiff[]
  undeclaredFlows: ExperienceFlowMetric[]
  capabilities: ExperienceCapabilityMetric[]
  errors: Array<{ message: string; count: number; routes: string[] }>
  recommendations: ExperienceRecommendation[]
  notes: string[]
}

export type ExperienceAdvisoryKind =
  | 'dead_click'
  | 'rage_click'
  | 'repeated_action'
  | 'leave_and_return'
  | 'dwell_without_action'
  | 'field_abandonment'
  | 'undeclared_flow'
  | 'declared_unobserved'
  | 'repeated_flow'
  | 'capability_unobserved'
  | 'runtime_error'

export interface ExperienceAdvisoryFinding {
  schemaVersion: 'experience-advisory-finding/v1'
  id: string
  severity: 'high' | 'medium' | 'low'
  kind: ExperienceAdvisoryKind
  title: string
  evidence: string
  suggestion: string
  route?: string
  element?: { fp?: string; testId?: string; name?: string; role?: string }
  reportDigest: string
  /** Always true; consumers must never treat this as a gate. */
  advisory: true
}

export interface ExperienceCorrectionCandidate {
  schemaVersion: 'experience-correction-candidate/v1'
  id: string
  kind: ExperienceAdvisoryKind
  severity: 'high' | 'medium' | 'low'
  route?: string
  element?: { fp?: string; testId?: string; name?: string; role?: string }
  evidence: string
  reportDigest: string
  status: 'candidate'
  source: 'observed'
  advisory: true
}

export interface ExperienceObservedBehavior {
  kind: 'experience-observed-behavior'
  schemaVersion: 'experience-observed-behavior/v1'
  /**
   * `absent` = no sessions; `stale` = the expected build was not observed;
   * `fresh` = all bound sessions share one non-empty build and it matches the
   * expected build. Mixed or missing session builds are stale.
   */
  state: 'fresh' | 'stale' | 'absent'
  scenario: string
  build: string | null
  builds: string[]
  /** Session ids and build identities supporting this aggregate. */
  sessions: Array<{ id: string; build: string | null; startedAt: string; endedAt: string }>
  expectedBuild?: string
  window: { from: string | null; to: string | null; sessions: number }
  reportDigest: string
  totals: ExperienceTotals
  flows: ExperienceFlowMetric[]
  routes: ExperienceRouteMetric[]
  friction: ExperienceFrictionMetric[]
  journeys: ExperienceJourneyDiff[]
  recommendations: ExperienceRecommendation[]
}

export interface ExperiencePacketObservedBlock {
  schemaVersion: 'experience-packet-observed/v1'
  scenario: string
  generatedAt: string
  build: string | null
  sessions: Array<{ id: string; build: string | null }>
  window: { from: string | null; to: string | null; sessions: number }
  headline: string
  topFlow: ExperienceFlowMetric | null
  topFriction: ExperienceFrictionMetric | null
  timeCostMs: number
  reportDigest: string
  caveat: string
}

/** A candidate flow, deliberately not a valid experience-journey document. */
export interface ExperienceObservedFlow {
  kind: 'experience-observed-flow'
  schemaVersion: 'experience-observed-flow/v1'
  scenario: string
  generatedAt: string
  flow: {
    id: string
    steps: Array<{ route: string; element?: string; testId?: string }>
    count: number
    medianDurationMs: number
  }
}
