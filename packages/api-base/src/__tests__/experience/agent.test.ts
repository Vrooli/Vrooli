import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { installExperienceTelemetry, type AgentController } from '../../experience/agent.js'

interface CapturedBody {
  v: number
  sessionId: string
  build?: string
  events: Array<Record<string, unknown>>
}

let controller: AgentController | null = null
let bodies: CapturedBody[] = []
let fetchMock: ReturnType<typeof vi.fn>

beforeAll(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date('2026-01-01T00:00:00Z'))
  document.body.innerHTML = `
    <main data-experience-region="workspace">
      <button data-testid="save" aria-label="Save changes">Save</button>
      <input data-testid="password" type="password" name="password" aria-label="Password" />
      <a data-testid="docs" href="/docs">Docs</a>
    </main>
  `

  fetchMock = vi.fn(async (_url: string, init?: { body?: string }) => {
    bodies.push(JSON.parse(String(init?.body)) as CapturedBody)
    return { ok: true } as Response
  })
  vi.stubGlobal('fetch', fetchMock)
  ;(window as unknown as { fetch: typeof fetch }).fetch = fetchMock as unknown as typeof fetch

  controller = installExperienceTelemetry({
    endpoint: '/collect',
    scenario: 'demo',
    sample: 1,
    captureLabels: true,
    performance: false,
    maxEvents: 1000,
  })
})

afterAll(() => {
  controller?.flush(false)
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('experience agent', () => {
  it('installs a debug controller on the window', () => {
    expect(controller).not.toBeNull()
    expect((window as unknown as { __vrooliExperience?: AgentController }).__vrooliExperience).toBe(controller)
    expect(controller!.status().sessionId).toMatch(/^[A-Za-z0-9_-]{8,64}$/)
  })

  it('captures semantic actions, redacted fields, navigation keys, and friction', () => {
    const button = document.querySelector('[data-testid="save"]') as HTMLButtonElement
    button.dispatchEvent(new MouseEvent('click', { bubbles: true, detail: 1, clientX: 12, clientY: 24 }))
    vi.advanceTimersByTime(50)

    const password = document.querySelector('[data-testid="password"]') as HTMLInputElement
    password.dispatchEvent(new FocusEvent('focusin', { bubbles: true }))
    password.value = 'super-secret-value'
    password.dispatchEvent(new Event('input', { bubbles: true }))
    vi.advanceTimersByTime(1000)
    password.dispatchEvent(new FocusEvent('focusout', { bubbles: true }))

    document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true }))
    document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'a', bubbles: true }))
    document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', ctrlKey: true, bubbles: true }))

    // Let the dead-click heuristic observe a quiet window.
    vi.advanceTimersByTime(900)
    controller!.flush(false)

    expect(bodies).toHaveLength(1)
    const events = bodies[0]!.events
    const types = events.map((event) => event.type)
    expect(types).toContain('session_start')
    expect(types).toContain('route')
    expect(types).toContain('action')
    expect(types).toContain('field')
    expect(types).toContain('key')
    expect(types).toContain('friction')
    expect(events.length).toBe(bodies[0]!.events.length)

    const click = events.find((event) => event.type === 'action') as Record<string, any>
    expect(click.action).toBe('click')
    expect(click.element.testId).toBe('save')
    expect(click.element.name).toBe('Save changes')
    expect(click.element.region).toBe('workspace')

    const blur = events.find((event) => event.type === 'field' && event.phase === 'blur') as Record<string, any>
    expect(blur.field.name).toBe('[redacted]')
    expect(blur.field.sensitive).toBe(true)
    expect(blur.field.durationMs).toBeGreaterThanOrEqual(1000)
    expect(blur.field.edited).toBe(true)

    const keys = events.filter((event) => event.type === 'key').map((event) => event.key)
    expect(keys).toContain('Tab')
    expect(keys).toContain('Mod+Z')
    expect(keys).not.toContain('a')

    const kinds = events.filter((event) => event.type === 'friction').map((event) => event.kind)
    expect(kinds).toContain('undo')
    expect(kinds).toContain('dead_click')

    expect(JSON.stringify(bodies[0])).not.toContain('super-secret-value')
  })

  it('never records printable keystrokes', () => {
    const joined = JSON.stringify(bodies)
    expect(joined).not.toContain('"key":"a"')
    expect(joined).not.toContain('super-secret-value')
  })

  it('refuses to install when the operator disabled it or sampling excludes the session', () => {
    localStorage.setItem('vrooli.experience.disabled', '1')
    expect(installExperienceTelemetry({ endpoint: '/collect', scenario: 'demo' })).toBeNull()
    localStorage.removeItem('vrooli.experience.disabled')

    expect(installExperienceTelemetry({ endpoint: '/collect', scenario: 'demo', sample: 0 })).toBeNull()
  })
})
