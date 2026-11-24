/**
 * Experience telemetry browser agent.
 *
 * Self-contained, framework-free ES module served by `createScenarioServer`.
 * It deliberately has no runtime imports so it can be loaded by any scenario
 * UI regardless of its React, bundler, Node, or TypeScript version.
 *
 * What it records:
 * - semantic actions (click / keyboard / submit / change) with a stable element
 *   reference, so a flow can be expressed in test-id terms instead of pixels;
 * - timing (route dwell, first action, field focus duration);
 * - pointer-derived hesitation features (travel, reversals, hover), never raw
 *   mouse trails;
 * - friction primitives (dead clicks, rage clicks, repeated actions, undo,
 *   dismiss, backtracking) that are cheap to detect and hard to fake;
 * - errors and a small set of performance metrics.
 *
 * What it never records:
 * - input values, clipboard contents, or printable keystrokes;
 * - IP addresses, user agents, or any identifier that survives the tab.
 *
 * This file must remain dependency-free. `inject.ts` owns the attribute name;
 * keep `MARKER` in sync (asserted by a unit test).
 */

type Theme = 'light' | 'dark' | 'unknown'

export interface AgentConfig {
  endpoint: string
  scenario: string
  build?: string
  sample: number
  captureLabels: boolean
  performance: boolean
  maxEvents: number
  debug: boolean
}

interface AgentStatus {
  version: string
  sessionId: string
  enabled: boolean
  endpoint: string
  queued: number
  sent: number
  startedAt: number
}

export interface AgentController {
  status(): AgentStatus
  flush(useBeacon?: boolean): void
  disable(): void
}

const MARKER = 'data-vrooli-experience'
const AGENT_VERSION = '1'
const STORAGE_KEY = 'vrooli.experience.disabled'
const SESSION_KEY = 'vrooli.experience.session'
const MAX_PATH = 240
const MAX_NAME = 80
const MAX_MESSAGE = 200
const MAX_MOUSE_SAMPLES = 120

const NAV_KEYS = new Set([
  'Tab', 'Enter', 'Escape', 'Backspace', 'Delete', 'Insert',
  'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight',
  'Home', 'End', 'PageUp', 'PageDown', ' ', 'Spacebar',
])

const SENSITIVE_NAME = /pass(word)?|secret|token|otp|one[-_]?time|cvv|cvc|card|ssn|social|routing|iban|account|license|email|phone|address|birth/i
const SENSITIVE_AUTOCOMPLETE = /cc-|current-password|new-password|one-time-code/i
const SENSITIVE_TYPE = new Set(['password', 'email', 'tel'])
const INTERACTIVE_ROLES = new Set([
  'button', 'link', 'tab', 'menuitem', 'menuitemcheckbox', 'menuitemradio', 'option',
  'checkbox', 'radio', 'switch', 'textbox', 'searchbox', 'combobox', 'slider', 'spinbutton',
])

function now(): number {
  return Date.now()
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function round(value: number): number {
  return Math.round(value)
}

function collapse(value: string | null | undefined): string {
  return (value || '').replace(/\s+/g, ' ').trim()
}

function hash(input: string): string {
  let h = 0x811c9dc5
  for (let i = 0; i < input.length; i += 1) {
    h ^= input.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  return (h >>> 0).toString(36)
}

function cssEscape(value: string): string {
  if (typeof CSS !== 'undefined' && typeof CSS.escape === 'function') {
    return CSS.escape(value)
  }
  return value.replace(/["\\\]]/g, '\\$&')
}

function safeRandomId(): string {
  try {
    if (typeof crypto !== 'undefined' && typeof (crypto as Crypto & { randomUUID?: () => string }).randomUUID === 'function') {
      return (crypto as Crypto & { randomUUID: () => string }).randomUUID().replace(/-/g, '').slice(0, 24)
    }
  } catch {
    // fall through
  }
  return `${Math.random().toString(36).slice(2)}${now().toString(36)}`.slice(0, 24)
}

function readSessionId(): string {
  try {
    const existing = sessionStorage.getItem(SESSION_KEY)
    if (existing && /^[A-Za-z0-9_-]{8,64}$/.test(existing)) return existing
    const created = safeRandomId()
    sessionStorage.setItem(SESSION_KEY, created)
    return created
  } catch {
    return safeRandomId()
  }
}

function currentTheme(): Theme {
  try {
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) return 'dark'
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) return 'light'
  } catch {
    // ignore
  }
  return 'unknown'
}

function reducedMotion(): boolean {
  try {
    return Boolean(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches)
  } catch {
    return false
  }
}

function frameInfo(): { depth: number; top: boolean } {
  let depth = 0
  try {
    let current: Window = window
    while (current.parent && current.parent !== current && depth < 8) {
      depth += 1
      current = current.parent
    }
  } catch {
    // Cross-origin access to parent is not needed; the WindowProxy comparison works.
  }
  return { depth, top: depth === 0 }
}

function sanitizeRoute(url: string): string {
  try {
    const parsed = new URL(url, window.location.origin)
    const params = new URLSearchParams(parsed.search)
    const redact = /token|code|key|secret|password|session|auth|sig/i
    params.forEach((_value, name) => {
      if (redact.test(name)) params.set(name, '[redacted]')
    })
    const search = params.toString()
    return `${parsed.pathname}${search ? `?${search}` : ''}${parsed.hash && parsed.hash.length <= 80 ? parsed.hash : ''}`.slice(0, 300)
  } catch {
    return String(url).slice(0, 300)
  }
}

function elementName(el: Element): string {
  const aria = el.getAttribute('aria-label')
  if (aria) return collapse(aria).slice(0, MAX_NAME)

  const labelledBy = el.getAttribute('aria-labelledby')
  if (labelledBy) {
    const parts: string[] = []
    labelledBy.split(/\s+/).forEach((id) => {
      const target = document.getElementById(id)
      if (target) parts.push(collapse(target.textContent))
    })
    const joined = collapse(parts.join(' '))
    if (joined) return joined.slice(0, MAX_NAME)
  }

  if (el instanceof HTMLInputElement) {
    const labels = el.labels ? Array.from(el.labels) : []
    const labelText = collapse(labels.map((label) => label.textContent).join(' '))
    if (labelText) return labelText.slice(0, MAX_NAME)
    if (el.placeholder) return collapse(el.placeholder).slice(0, MAX_NAME)
  }

  const title = el.getAttribute('title')
  if (title) return collapse(title).slice(0, MAX_NAME)

  const alt = el.getAttribute('alt')
  if (alt) return collapse(alt).slice(0, MAX_NAME)

  return collapse(el.textContent).slice(0, MAX_NAME)
}

function implicitRole(el: Element): string {
  const explicit = el.getAttribute('role')
  if (explicit) return explicit
  const tag = el.tagName.toLowerCase()
  if (tag === 'button') return 'button'
  if (tag === 'a' && el.hasAttribute('href')) return 'link'
  if (tag === 'input') {
    const type = (el as HTMLInputElement).type
    if (type === 'checkbox') return 'checkbox'
    if (type === 'radio') return 'radio'
    if (type === 'submit' || type === 'button') return 'button'
    if (type === 'search') return 'searchbox'
    return 'textbox'
  }
  if (tag === 'textarea') return 'textbox'
  if (tag === 'select') return 'combobox'
  if (tag === 'summary') return 'button'
  if (tag === 'dialog') return 'dialog'
  if (/^h[1-6]$/.test(tag)) return 'heading'
  return ''
}

function nthOfType(el: Element): number {
  const parent = el.parentElement
  if (!parent) return 1
  let index = 0
  const siblings = parent.children
  for (let i = 0; i < siblings.length; i += 1) {
    const candidate = siblings[i]
    if (candidate && candidate.tagName === el.tagName) {
      index += 1
      if (candidate === el) return index
    }
  }
  return index
}

function buildPath(el: Element): string {
  const parts: string[] = []
  let node: Element | null = el
  let depth = 0
  while (node && node !== document.body && depth < 5) {
    const tag = node.tagName.toLowerCase()
    const testId = node.getAttribute('data-testid')
    if (testId) {
      parts.unshift(`${tag}[data-testid="${cssEscape(testId)}"]`)
      break
    }
    if (node.id) {
      parts.unshift(`${tag}#${cssEscape(node.id)}`)
      break
    }
    const index = nthOfType(node)
    parts.unshift(index > 1 ? `${tag}:nth-of-type(${index})` : tag)
    node = node.parentElement
    depth += 1
  }
  return parts.join('>').slice(0, MAX_PATH) || el.tagName.toLowerCase()
}

function isInteractiveElement(el: Element): boolean {
  const tag = el.tagName.toLowerCase()
  if (tag === 'button' || tag === 'a' || tag === 'input' || tag === 'select' || tag === 'textarea' || tag === 'summary' || tag === 'option') {
    return true
  }
  if (el.hasAttribute('data-testid')) return true
  const role = el.getAttribute('role')
  return Boolean(role && INTERACTIVE_ROLES.has(role))
}

interface ElementRef {
  testId?: string
  role?: string
  name?: string
  tag: string
  path: string
  fp: string
  region?: string
  surface?: string
  disabled?: boolean
  rect?: { x: number; y: number; width: number; height: number }
  inViewport?: boolean
}

function describeElement(el: Element, labels: boolean): ElementRef {
  const tag = el.tagName.toLowerCase()
  const testId = el.getAttribute('data-testid') || undefined
  const role = implicitRole(el) || undefined
  const name = labels && isInteractiveElement(el) ? elementName(el) || undefined : undefined
  const path = buildPath(el)

  let region: string | undefined
  let surface: string | undefined
  try {
    const regionEl = el.closest('[data-experience-region]')
    region = regionEl?.getAttribute('data-experience-region') || undefined
    if (!region) {
      const landmark = el.closest('[role="main"],[role="region"],[role="navigation"],[role="complementary"]')
      region = landmark?.getAttribute('aria-label') || landmark?.getAttribute('role') || undefined
    }
    surface = el.closest('[data-experience-surface]')?.getAttribute('data-experience-surface') || undefined
  } catch {
    // ignore
  }

  let rect: ElementRef['rect']
  let inViewport: boolean | undefined
  try {
    const box = el.getBoundingClientRect()
    rect = { x: round(box.x), y: round(box.y), width: round(box.width), height: round(box.height) }
    inViewport = box.bottom > 0 && box.right > 0 && box.top < window.innerHeight && box.left < window.innerWidth
  } catch {
    // ignore
  }

  const disabled = el instanceof HTMLButtonElement || el instanceof HTMLInputElement || el instanceof HTMLSelectElement || el instanceof HTMLTextAreaElement
    ? el.disabled
    : el.getAttribute('aria-disabled') === 'true'

  return {
    ...(testId ? { testId } : {}),
    ...(role ? { role } : {}),
    ...(name ? { name } : {}),
    tag,
    path,
    fp: hash(`${tag}|${role || ''}|${testId || ''}|${name || ''}|${path}`),
    ...(region ? { region } : {}),
    ...(surface ? { surface } : {}),
    ...(disabled ? { disabled } : {}),
    ...(rect ? { rect } : {}),
    ...(inViewport !== undefined ? { inViewport } : {}),
  }
}

function isSensitiveField(el: Element): boolean {
  if (el instanceof HTMLInputElement) {
    if (SENSITIVE_TYPE.has(el.type)) return true
    if (SENSITIVE_AUTOCOMPLETE.test(el.getAttribute('autocomplete') || '')) return true
  }
  const hint = [
    el.getAttribute('name'),
    el.getAttribute('id'),
    el.getAttribute('data-testid'),
    el.getAttribute('aria-label'),
  ].filter(Boolean).join(' ')
  return SENSITIVE_NAME.test(hint)
}

function fieldDescriptor(el: Element): { type: string; name: string; sensitive: boolean } {
  const tag = el.tagName.toLowerCase()
  const type = el instanceof HTMLInputElement ? el.type : tag === 'textarea' ? 'textarea' : tag === 'select' ? 'select' : 'contenteditable'
  const sensitive = isSensitiveField(el)
  if (sensitive) return { type, name: '[redacted]', sensitive: true }
  const label = el.getAttribute('aria-label') || el.getAttribute('name') || el.getAttribute('data-testid') || (el instanceof HTMLElement ? elementName(el) : '') || type
  return { type, name: collapse(label).slice(0, MAX_NAME) || type, sensitive: false }
}

interface PointerSample { x: number; y: number; t: number }

function readConfig(tag: HTMLScriptElement): Partial<AgentConfig> {
  const dataset = tag.dataset
  const sample = Number.parseFloat(dataset.sample || '1')
  const maxEvents = Number.parseInt(dataset.maxEvents || '', 10)
  return {
    endpoint: dataset.endpoint || undefined,
    scenario: dataset.scenario || undefined,
    build: dataset.build || undefined,
    sample: Number.isFinite(sample) ? clamp(sample, 0, 1) : 1,
    captureLabels: dataset.labels !== '0',
    performance: dataset.performance !== '0',
    maxEvents: Number.isFinite(maxEvents) && maxEvents > 0 ? maxEvents : 5000,
    debug: dataset.debug === '1',
  } as Partial<AgentConfig>
}

/**
 * Install the agent. Returns null when telemetry is disabled, sampled out, or
 * the document is not a browser document.
 */
export function installExperienceTelemetry(overrides: Partial<AgentConfig> = {}): AgentController | null {
  if (typeof window === 'undefined' || typeof document === 'undefined') return null

  let disabledByUser = false
  try {
    disabledByUser = localStorage.getItem(STORAGE_KEY) === '1'
  } catch {
    // Storage can be unavailable in sandboxed frames; treat as enabled.
  }
  if (disabledByUser) return null

  const tag = document.querySelector(`script[${MARKER}]`) as HTMLScriptElement | null
  const fromTag = tag ? readConfig(tag) : {}
  const config: AgentConfig = {
    endpoint: overrides.endpoint ?? fromTag.endpoint ?? '/__vrooli/experience/events',
    scenario: overrides.scenario ?? fromTag.scenario ?? 'unknown',
    build: overrides.build ?? fromTag.build,
    sample: overrides.sample ?? fromTag.sample ?? 1,
    captureLabels: overrides.captureLabels ?? fromTag.captureLabels ?? true,
    performance: overrides.performance ?? fromTag.performance ?? true,
    maxEvents: overrides.maxEvents ?? fromTag.maxEvents ?? 5000,
    debug: overrides.debug ?? fromTag.debug ?? false,
  }

  const sessionId = readSessionId()
  if (config.sample < 1) {
    const bucket = Number.parseInt(hash(sessionId).slice(-4), 36) % 1000
    if (bucket >= config.sample * 1000) return null
  }

  let enabled = true
  let seqNumber = 0
  let sent = 0
  let dropped = 0
  let emitted = 0
  let queue: unknown[] = []
  let timer: ReturnType<typeof setTimeout> | null = null
  let flushFailures = 0
  let truncationReported = false
  const startedAt = now()
  const points: PointerSample[] = []
  const hoverStart = new WeakMap<Element, number>()
  const fieldStart = new WeakMap<Element, { at: number; edited: boolean }>()
  const recentClicks: Array<{ fp: string; at: number; x: number; y: number }> = []
  const repeated = new Map<string, number[]>()
  const routeHistory: Array<{ route: string; at: number }> = []
  let previousActionAt = 0
  let lastPointer: { x: number; y: number; at: number } | null = null
  let route = sanitizeRoute(window.location.href)
  let scrollMax = 0
  let hiddenMs = 0
  let hiddenAt = 0
  let longTasks = 0
  let longTaskMs = 0
  let cls = 0
  let lcp: number | undefined

  function log(message: string, detail?: unknown): void {
    if (!config.debug) return
    try {
      console.debug(`[vrooli-experience] ${message}`, detail)
    } catch {
      // ignore
    }
  }

  function emit(draft: Record<string, unknown> & { type: string }): void {
    if (!enabled) return
    if (emitted >= config.maxEvents) {
      if (!truncationReported && draft.type === 'friction') {
        truncationReported = true
        emitted += 1
        queue.push({ ...draft, kind: 'truncated', ts: now(), seq: seqNumber++ })
      }
      dropped += 1
      return
    }
    emitted += 1
    queue.push({ ...draft, ts: now(), seq: seqNumber++ })
    schedule()
  }

  function schedule(): void {
    if (timer !== null) return
    timer = setTimeout(() => {
      timer = null
      flush(false)
    }, 2500)
  }

  function post(body: string, useBeacon: boolean, requeue: () => void): boolean {
    try {
      if (useBeacon && typeof navigator.sendBeacon === 'function') {
        const blob = new Blob([body], { type: 'application/json' })
        if (navigator.sendBeacon(config.endpoint, blob)) return true
      }
      void fetch(config.endpoint, {
        method: 'POST',
        keepalive: true,
        headers: { 'content-type': 'application/json' },
        body,
      }).then((response) => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`)
        flushFailures = 0
      }).catch(() => {
        flushFailures += 1
        requeue()
      })
      return true
    } catch {
      flushFailures += 1
      return false
    }
  }

  function flush(useBeacon: boolean): void {
    if (timer !== null) {
      clearTimeout(timer)
      timer = null
    }
    if (queue.length === 0) return
    const events = queue
    queue = []
    const body = JSON.stringify({
      v: 1,
      sessionId,
      ...(config.build ? { build: config.build } : {}),
      events,
    })
    const delivered = post(body, useBeacon, () => {
      if (flushFailures <= 2) queue = [...events, ...queue]
    })
    if (delivered) {
      sent += events.length
    } else if (flushFailures <= 2) {
      queue = [...events, ...queue]
    } else {
      log('dropping batch after repeated delivery failures', { events: events.length })
    }
  }

  function scrollDepth(): number {
    try {
      const max = Math.max(0, document.documentElement.scrollHeight - window.innerHeight)
      return max === 0 ? 0 : clamp(scrollMax / max, 0, 1)
    } catch {
      return 0
    }
  }

  function commitRoute(via: 'load' | 'push' | 'replace' | 'pop' | 'hash', nextRoute: string, fromRoute: string): void {
    const depth = scrollDepth()
    route = nextRoute
    scrollMax = 0
    routeHistory.push({ route: nextRoute, at: now() })
    if (routeHistory.length > 30) routeHistory.shift()
    emit({ type: 'route', route: nextRoute, from: fromRoute, via, scrollDepth: depth })
    const prior = routeHistory.filter((entry) => entry.route === nextRoute).length
    if (prior > 1) {
      const last = routeHistory.slice(0, -1).reverse().find((entry) => entry.route === nextRoute)
      if (last && now() - last.at < 10000) {
        emit({ type: 'friction', kind: 'backtrack', route: nextRoute, detail: { sinceMs: now() - last.at } })
      }
    }
  }

  function onRoute(via: 'push' | 'replace' | 'pop' | 'hash'): void {
    try {
      const next = sanitizeRoute(window.location.href)
      if (next === route) return
      const from = route
      commitRoute(via, next, from)
    } catch (error) {
      log('route handler failed', error)
    }
  }

  function pointerFeatures(): { travelPx: number; reversals: number; speedPxPerMs: number } | undefined {
    if (points.length < 2) return undefined
    let travel = 0
    let reversals = 0
    let previousAngle: number | null = null
    let previous = points[0]
    for (let i = 1; i < points.length; i += 1) {
      const current = points[i]
      if (!current || !previous) continue
      const dx = current.x - previous.x
      const dy = current.y - previous.y
      const distance = Math.sqrt(dx * dx + dy * dy)
      travel += distance
      if (distance >= 8) {
        const angle = Math.atan2(dy, dx)
        if (previousAngle !== null) {
          let delta = Math.abs(angle - previousAngle)
          if (delta > Math.PI) delta = Math.PI * 2 - delta
          if (delta > Math.PI / 2) reversals += 1
        }
        previousAngle = angle
      }
      previous = current
    }
    const span = Math.max(1, points[points.length - 1]!.t - points[0]!.t)
    return { travelPx: round(travel), reversals, speedPxPerMs: Number((travel / span).toFixed(3)) }
  }

  function interactiveTarget(target: EventTarget | null): Element | null {
    if (!(target instanceof Element)) return null
    if (target.closest('[data-experience-ignore]')) return null
    return target.closest('button,a,input,select,textarea,summary,[role],[data-testid],[contenteditable="true"]') || target
  }

  function recordAction(action: 'click' | 'keyboard' | 'submit' | 'change', el: Element, extra: Record<string, unknown> = {}): void {
    const element = describeElement(el, config.captureLabels)
    emit({ type: 'action', action, element, ...extra })
    previousActionAt = now()
  }

  function checkDeadClick(el: Element, at: number, routeAtClick: string): void {
    const resourcesBefore = countResources()
    let mutations = 0
    let routeChanged = false
    let observer: MutationObserver | null = null
    try {
      observer = new MutationObserver((records) => {
        mutations += records.length
      })
      observer.observe(document.documentElement, { subtree: true, childList: true, attributes: true, characterData: true })
    } catch {
      observer = null
    }
    setTimeout(() => {
      try {
        observer?.disconnect()
        if (route !== routeAtClick) routeChanged = true
        const resourceDelta = countResources() - resourcesBefore
        log('dead-click check', { mutations, resourceDelta, routeChanged, connected: el.isConnected })
        if (!routeChanged && mutations === 0 && resourceDelta <= 0 && el.isConnected) {
          emit({
            type: 'friction',
            kind: 'dead_click',
            element: describeElement(el, config.captureLabels),
            route,
            detail: { waitedMs: now() - at },
          })
        }
      } catch (error) {
        log('dead-click check failed', error)
      }
    }, 700)
  }

  const resourceSeen = new Set<string>()
  let resourceCount = 0
  function countResources(): number {
    try {
      if (typeof performance === 'undefined' || typeof performance.getEntriesByType !== 'function') return resourceCount
      const entries = performance.getEntriesByType('resource') as PerformanceResourceTiming[]
      for (const entry of entries) {
        const key = `${entry.name}|${entry.startTime}`
        if (!resourceSeen.has(key)) {
          resourceSeen.add(key)
          resourceCount += 1
        }
      }
    } catch {
      // ignore
    }
    return resourceCount
  }

  function recordFriction(kind: string, el?: Element | null, detail?: Record<string, unknown>): void {
    emit({
      type: 'friction',
      kind,
      ...(el ? { element: describeElement(el, config.captureLabels) } : {}),
      route,
      ...(detail ? { detail } : {}),
    })
  }

  function onPointerDown(event: PointerEvent): void {
    try {
      lastPointer = { x: round(event.clientX), y: round(event.clientY), at: now() }
    } catch {
      // ignore
    }
  }

  function onClick(event: MouseEvent): void {
    try {
      const el = interactiveTarget(event.target)
      if (!el) return
      const target = el as Element
      if ((target === document.body || target === document.documentElement) && !target.getAttribute('data-testid')) return
      const at = now()
      const enter = hoverStart.get(target)
      const hoverMs = enter ? Math.max(0, at - enter) : undefined
      const keyboard = event.detail === 0
      const pointer = pointerFeatures()
      const cursor = lastPointer && at - lastPointer.at < 1500 ? { x: lastPointer.x, y: lastPointer.y } : { x: round(event.clientX), y: round(event.clientY) }
      recordAction(keyboard ? 'keyboard' : 'click', target, {
        ...(keyboard ? {} : { cursor }),
        ...(hoverMs !== undefined ? { hoverMs } : {}),
        ...(pointer ? { pointer } : {}),
      })

      const ref = describeElement(target, false)
      recentClicks.push({ fp: ref.fp, at, x: round(event.clientX), y: round(event.clientY) })
      while (recentClicks.length > 0 && at - recentClicks[0]!.at > 1500) recentClicks.shift()
      const sameTarget = recentClicks.filter((entry) => entry.fp === ref.fp)
      if (sameTarget.length >= 3 && Math.abs(sameTarget[0]!.x - sameTarget[sameTarget.length - 1]!.x) < 40 && Math.abs(sameTarget[0]!.y - sameTarget[sameTarget.length - 1]!.y) < 40) {
        recordFriction('rage_click', target, { clicks: sameTarget.length, windowMs: 1500 })
        recentClicks.length = 0
      }

      const stamps = repeated.get(ref.fp) || []
      stamps.push(at)
      const recent = stamps.filter((stamp) => at - stamp <= 10000)
      repeated.set(ref.fp, recent)
      if (recent.length === 4) {
        recordFriction('repeated_action', target, { count: recent.length, windowMs: 10000 })
      }

      checkDeadClick(target, at, route)
    } catch (error) {
      log('click handler failed', error)
    }
  }

  function navigationKey(event: KeyboardEvent): string | null {
    const mod = event.metaKey || event.ctrlKey
    if (mod && /^[a-z]$/i.test(event.key)) return `Mod+${event.key.toUpperCase()}`
    if (event.altKey && /^[a-z]$/i.test(event.key)) return `Alt+${event.key.toUpperCase()}`
    if (event.shiftKey && event.key === 'Tab') return 'Shift+Tab'
    if (NAV_KEYS.has(event.key)) return event.key === ' ' ? 'Space' : event.key
    return null
  }

  function onKeyDown(event: KeyboardEvent): void {
    try {
      const key = navigationKey(event)
      if (!key) return
      const el = interactiveTarget(event.target)
      emit({
        type: 'key',
        key,
        ...(el ? { element: describeElement(el, config.captureLabels) } : {}),
      })
      if (key === 'Escape') {
        const since = previousActionAt ? now() - previousActionAt : undefined
        recordFriction('dismiss', el, since !== undefined ? { sincePreviousMs: since } : undefined)
      } else if (key === 'Mod+Z') {
        recordFriction('undo', el)
      }
    } catch (error) {
      log('key handler failed', error)
    }
  }

  function onFocusIn(event: FocusEvent): void {
    try {
      const el = event.target
      if (!(el instanceof Element)) return
      if (!(el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement || el instanceof HTMLSelectElement || el.getAttribute('contenteditable') === 'true')) return
      if (el instanceof HTMLInputElement && el.type === 'hidden') return
      fieldStart.set(el, { at: now(), edited: false })
      emit({ type: 'field', phase: 'focus', field: fieldDescriptor(el) })
    } catch (error) {
      log('focus handler failed', error)
    }
  }

  function onInput(event: Event): void {
    try {
      const el = event.target
      if (!(el instanceof Element)) return
      const record = fieldStart.get(el)
      if (record) record.edited = true
    } catch {
      // ignore
    }
  }

  function onFocusOut(event: FocusEvent): void {
    try {
      const el = event.target
      if (!(el instanceof Element)) return
      const record = fieldStart.get(el)
      if (!record) return
      fieldStart.delete(el)
      emit({
        type: 'field',
        phase: 'blur',
        field: {
          ...fieldDescriptor(el),
          durationMs: Math.max(0, now() - record.at),
          edited: record.edited,
        },
      })
    } catch (error) {
      log('blur handler failed', error)
    }
  }

  function onSubmit(event: Event): void {
    try {
      const form = event.target
      if (form instanceof Element) recordAction('submit', form)
    } catch (error) {
      log('submit handler failed', error)
    }
  }

  function onChange(event: Event): void {
    try {
      const el = interactiveTarget(event.target)
      if (!el) return
      if (el instanceof HTMLInputElement && el.type === 'hidden') return
      recordAction('change', el)
    } catch (error) {
      log('change handler failed', error)
    }
  }

  function onPointerMove(event: PointerEvent): void {
    try {
      points.push({ x: round(event.clientX), y: round(event.clientY), t: now() })
      while (points.length > MAX_MOUSE_SAMPLES) points.shift()
      const cutoff = now() - 1000
      while (points.length > 0 && points[0]!.t < cutoff) points.shift()
    } catch {
      // ignore
    }
  }

  function onPointerOver(event: PointerEvent): void {
    try {
      const target = interactiveTarget(event.target)
      if (target) hoverStart.set(target, now())
    } catch {
      // ignore
    }
  }

  function onPointerOut(event: PointerEvent): void {
    try {
      const target = interactiveTarget(event.target)
      if (target) hoverStart.delete(target)
    } catch {
      // ignore
    }
  }

  function onScroll(): void {
    try {
      scrollMax = Math.max(scrollMax, window.scrollY || document.documentElement.scrollTop || 0)
    } catch {
      // ignore
    }
  }

  function onError(event: ErrorEvent): void {
    try {
      emit({
        type: 'error',
        kind: 'error',
        message: collapse(event.message).slice(0, MAX_MESSAGE) || 'error',
        ...(event.filename ? { source: collapse(event.filename).slice(0, 200) } : {}),
        ...(typeof event.lineno === 'number' ? { line: event.lineno } : {}),
      })
    } catch {
      // ignore
    }
  }

  function onRejection(event: PromiseRejectionEvent): void {
    try {
      const reason = event.reason
      const message = typeof reason === 'string' ? reason : reason && typeof reason.message === 'string' ? reason.message : String(reason)
      emit({ type: 'error', kind: 'unhandledrejection', message: collapse(message).slice(0, MAX_MESSAGE) || 'rejection' })
    } catch {
      // ignore
    }
  }

  function emitPerformance(): void {
    if (!config.performance) return
    const metrics: Record<string, number> = {}
    try {
      const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming | undefined
      if (navigation) {
        metrics.ttfbMs = round(navigation.responseStart)
        metrics.domContentLoadedMs = round(navigation.domContentLoadedEventEnd)
        metrics.loadMs = round(navigation.loadEventEnd)
      }
    } catch {
      // ignore
    }
    if (lcp !== undefined) metrics.lcpMs = round(lcp)
    if (cls > 0) metrics.cls = Number(cls.toFixed(4))
    if (longTasks > 0) {
      metrics.longTasks = longTasks
      metrics.longTaskMs = round(longTaskMs)
    }
    if (Object.keys(metrics).length === 0) return
    emit({ type: 'performance', route, metrics })
    longTasks = 0
    longTaskMs = 0
  }

  function observePerformance(): void {
    if (!config.performance || typeof PerformanceObserver === 'undefined') return
    try {
      const longTaskObserver = new PerformanceObserver((list) => {
        list.getEntries().forEach((entry) => {
          longTasks += 1
          longTaskMs += entry.duration
        })
      })
      longTaskObserver.observe({ entryTypes: ['longtask'] } as PerformanceObserverInit)
    } catch {
      // longtask is not supported everywhere
    }
    try {
      const lcpObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries()
        const last = entries[entries.length - 1]
        if (last) lcp = last.startTime
      })
      lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true } as PerformanceObserverInit)
    } catch {
      // ignore
    }
    try {
      const clsObserver = new PerformanceObserver((list) => {
        list.getEntries().forEach((entry) => {
          const shift = entry as PerformanceEntry & { value?: number; hadRecentInput?: boolean }
          if (!shift.hadRecentInput && typeof shift.value === 'number') cls += shift.value
        })
      })
      clsObserver.observe({ type: 'layout-shift', buffered: true } as PerformanceObserverInit)
    } catch {
      // ignore
    }
  }

  function onVisibility(): void {
    try {
      const visible = document.visibilityState === 'visible'
      if (!visible) {
        hiddenAt = now()
        emitPerformance()
        emit({ type: 'visibility', state: 'hidden' })
        flush(true)
      } else {
        if (hiddenAt > 0) hiddenMs += now() - hiddenAt
        hiddenAt = 0
        emit({ type: 'visibility', state: 'visible' })
      }
    } catch (error) {
      log('visibility handler failed', error)
    }
  }

  function onPageHide(): void {
    try {
      emitPerformance()
      emit({ type: 'session_end', activeMs: Math.max(0, now() - startedAt - hiddenMs), reason: 'unload' })
      flush(true)
    } catch {
      // ignore
    }
  }

  function patchHistory(): void {
    const wrap = (method: 'pushState' | 'replaceState'): void => {
      const original = history[method]
      if (typeof original !== 'function') return
      history[method] = function patched(this: History, ...args: Parameters<History['pushState']>): void {
        const result = original.apply(this, args)
        onRoute(method === 'pushState' ? 'push' : 'replace')
        return result
      } as History['pushState']
    }
    try {
      wrap('pushState')
      wrap('replaceState')
    } catch {
      // ignore
    }
  }

  function listen(): void {
    const capture = { capture: true, passive: true } as AddEventListenerOptions
    window.addEventListener('pointerdown', onPointerDown as EventListener, capture)
    window.addEventListener('pointermove', onPointerMove as EventListener, capture)
    window.addEventListener('pointerover', onPointerOver as EventListener, capture)
    window.addEventListener('pointerout', onPointerOut as EventListener, capture)
    window.addEventListener('scroll', onScroll, capture)
    document.addEventListener('click', onClick as EventListener, true)
    document.addEventListener('keydown', onKeyDown as EventListener, true)
    document.addEventListener('focusin', onFocusIn as EventListener, true)
    document.addEventListener('focusout', onFocusOut as EventListener, true)
    document.addEventListener('input', onInput as EventListener, true)
    document.addEventListener('change', onChange as EventListener, true)
    document.addEventListener('submit', onSubmit as EventListener, true)
    window.addEventListener('error', onError as EventListener, true)
    window.addEventListener('unhandledrejection', onRejection as EventListener, true)
    window.addEventListener('popstate', () => onRoute('pop'))
    window.addEventListener('hashchange', () => onRoute('hash'))
    document.addEventListener('visibilitychange', onVisibility)
    window.addEventListener('pagehide', onPageHide)
  }

  patchHistory()
  listen()
  observePerformance()

  emit({
    type: 'session_start',
    route,
    viewport: { width: window.innerWidth, height: window.innerHeight, dpr: Number(window.devicePixelRatio || 1) },
    reducedMotion: reducedMotion(),
    theme: currentTheme(),
    frame: frameInfo(),
  })
  emit({ type: 'route', route, via: 'load', scrollDepth: 0 })
  routeHistory.push({ route, at: now() })

  try {
    if (document.readyState === 'complete') {
      setTimeout(emitPerformance, 2500)
    } else {
      window.addEventListener('load', () => setTimeout(emitPerformance, 2500), { once: true })
    }
  } catch {
    // ignore
  }

  const controller: AgentController = {
    status: () => ({
      version: AGENT_VERSION,
      sessionId,
      enabled,
      endpoint: config.endpoint,
      queued: queue.length,
      sent,
      startedAt,
    }),
    flush: (useBeacon = false) => flush(useBeacon),
    disable: () => {
      try {
        localStorage.setItem(STORAGE_KEY, '1')
      } catch {
        // ignore
      }
      emit({ type: 'session_end', activeMs: Math.max(0, now() - startedAt - hiddenMs), reason: 'disabled' })
      flush(true)
      enabled = false
      log('disabled by user')
    },
  }

  try {
    ;(window as unknown as { __vrooliExperience?: AgentController }).__vrooliExperience = controller
  } catch {
    // ignore
  }

  log('installed', { sessionId, endpoint: config.endpoint, sample: config.sample, dropped })
  return controller
}

// Auto-install when the server injected this module. Tests import the module
// directly and call the installer, so this branch stays inert there.
if (typeof window !== 'undefined' && typeof document !== 'undefined') {
  try {
    if (document.querySelector(`script[${MARKER}]`)) {
      installExperienceTelemetry()
    }
  } catch {
    // Never let telemetry break the host application.
  }
}
