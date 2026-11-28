import { describe, expect, it } from 'vitest'
import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import {
  experienceScenarioName,
  isLoopbackHost,
  readExperiencePolicy,
  resolveExperienceTelemetry,
  shouldInstrumentRequest,
} from '../../experience/gate.js'

const env = { NODE_ENV: 'development' } as NodeJS.ProcessEnv

describe('isLoopbackHost', () => {
  it('accepts loopback forms and rejects the tunnel', () => {
    expect(isLoopbackHost('localhost:3000')).toBe(true)
    expect(isLoopbackHost('127.0.0.1:3000')).toBe(true)
    expect(isLoopbackHost('[::1]:3000')).toBe(true)
    expect(isLoopbackHost('app.localhost')).toBe(true)
    expect(isLoopbackHost('abc.trycloudflare.com')).toBe(false)
    expect(isLoopbackHost(undefined)).toBe(false)
  })
})

describe('resolveExperienceTelemetry', () => {
  it('defaults on for a local run', () => {
    const resolved = resolveExperienceTelemetry(undefined, env, 'git-control-tower')
    expect(resolved?.enabled).toBe(true)
    expect(resolved?.scenario).toBe('git-control-tower')
    expect(resolved?.dataDir).toContain('git-control-tower')
  })

  it('defaults off under tests and CI', () => {
    expect(resolveExperienceTelemetry(undefined, { NODE_ENV: 'test' } as NodeJS.ProcessEnv, 'x')).toBeNull()
    expect(resolveExperienceTelemetry(undefined, { CI: 'true' } as NodeJS.ProcessEnv, 'x')).toBeNull()
    expect(resolveExperienceTelemetry(undefined, { VITEST: 'true' } as NodeJS.ProcessEnv, 'x')).toBeNull()
  })

  it('honours the environment switch and explicit options', () => {
    expect(resolveExperienceTelemetry(undefined, { ...env, VROOLI_EXPERIENCE_TELEMETRY: '0' } as NodeJS.ProcessEnv, 'x')).toBeNull()
    expect(resolveExperienceTelemetry(undefined, { ...env, VROOLI_EXPERIENCE_TELEMETRY: '1' } as NodeJS.ProcessEnv, 'x')?.source).toBe('env')
    expect(resolveExperienceTelemetry(true, { NODE_ENV: 'test' } as NodeJS.ProcessEnv, 'x')?.source).toBe('option')
    expect(resolveExperienceTelemetry({ enabled: false }, env, 'x')).toBeNull()
  })

  it('binds the served build identity into the resolved telemetry config', () => {
    expect(resolveExperienceTelemetry(undefined, { ...env, VROOLI_BUILD_IDENTITY: 'sha256:served-build' } as NodeJS.ProcessEnv, 'x')?.build)
      .toBe('sha256:served-build')
    expect(resolveExperienceTelemetry({ enabled: true, build: 'sha256:explicit-build' }, { ...env, VROOLI_BUILD_IDENTITY: 'sha256:served-build' } as NodeJS.ProcessEnv, 'x')?.build)
      .toBe('sha256:explicit-build')
  })

  it('sanitizes directory names and clamps sampling', () => {
    expect(experienceScenarioName('My Scenario / ../x')).toBe('my-scenario-x')
    const resolved = resolveExperienceTelemetry({ enabled: true, sample: 4 }, env, 'x')
    expect(resolved?.sample).toBe(1)
  })

  it('defaults retention and limits', () => {
    const resolved = resolveExperienceTelemetry({ enabled: true }, env, 'x')!
    expect(resolved.maxAgeDays).toBe(30)
    expect(resolved.maxSessions).toBe(500)
    expect(resolved.maxEventsPerSession).toBe(5000)
    expect(resolved.allowRemoteHosts).toBe(false)
  })

  it('honours a directory override from the environment', () => {
    const resolved = resolveExperienceTelemetry({ enabled: true }, { ...env, VROOLI_EXPERIENCE_DIR: '/tmp/experience-dir' } as NodeJS.ProcessEnv, 'x')!
    expect(resolved.dataDir).toBe('/tmp/experience-dir')
  })

  it('reads a per-scenario policy file and applies its precedence', () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'experience-policy-'))
    const file = path.join(dir, 'experience.json')
    fs.writeFileSync(file, JSON.stringify({ enabled: false, sample: 0.25, maxAgeDays: 7 }))
    try {
      expect(readExperiencePolicy(file)).toMatchObject({ enabled: false, sample: 0.25, maxAgeDays: 7 })
      expect(resolveExperienceTelemetry({ policyFile: file }, env, 'x')).toBeNull()

      fs.writeFileSync(file, JSON.stringify({ enabled: true, captureLabels: false }))
      const enabled = resolveExperienceTelemetry({ policyFile: file }, { ...env, NODE_ENV: 'test' } as NodeJS.ProcessEnv, 'x')
      expect(enabled?.source).toBe('policy')
      expect(enabled?.captureLabels).toBe(false)

      // Capability declarations are read from the same file.
      fs.writeFileSync(file, JSON.stringify({
        capabilities: [
          { id: 'stage-queue', label: 'Queue files', testId: 'stage-during-suggestion-queue' },
          { id: 'broken' },
          { label: 'no id' },
        ],
      }))
      expect(readExperiencePolicy(file)?.capabilities).toEqual([
        { id: 'stage-queue', label: 'Queue files', testId: 'stage-during-suggestion-queue' },
      ])

      // Explicit options win over the file.
      expect(resolveExperienceTelemetry({ policyFile: file, enabled: false }, env, 'x')).toBeNull()
    } finally {
      fs.rmSync(dir, { recursive: true, force: true })
    }
  })

  it('ignores a malformed policy file', () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'experience-policy-bad-'))
    const file = path.join(dir, 'experience.json')
    fs.writeFileSync(file, '{oops')
    try {
      expect(readExperiencePolicy(file)).toBeNull()
    } finally {
      fs.rmSync(dir, { recursive: true, force: true })
    }
  })
})

describe('shouldInstrumentRequest', () => {
  const resolved = resolveExperienceTelemetry({ enabled: true }, env, 'x')!

  const request = (overrides: Partial<{ method: string; path: string; headers: Record<string, string> }> = {}) => ({
    method: overrides.method ?? 'GET',
    path: overrides.path ?? '/',
    headers: { host: 'localhost:3000', accept: 'text/html', ...(overrides.headers || {}) },
  })

  it('instruments loopback documents', () => {
    expect(shouldInstrumentRequest(request(), resolved)).toBe(true)
  })

  it('refuses non-loopback hosts unless explicitly allowed', () => {
    expect(shouldInstrumentRequest(request({ headers: { host: 'abc.trycloudflare.com' } }), resolved)).toBe(false)
    const remote = resolveExperienceTelemetry({ enabled: true, allowRemoteHosts: true }, env, 'x')!
    expect(shouldInstrumentRequest(request({ headers: { host: 'abc.trycloudflare.com' } }), remote)).toBe(true)
  })

  it('refuses writes, telemetry routes, and automation opt-outs', () => {
    expect(shouldInstrumentRequest(request({ method: 'POST' }), resolved)).toBe(false)
    expect(shouldInstrumentRequest(request({ path: resolved.ingestPath }), resolved)).toBe(false)
    expect(shouldInstrumentRequest(request({ headers: { 'x-vrooli-experience': 'off' } }), resolved)).toBe(false)
    expect(shouldInstrumentRequest(request({ headers: { 'x-vrooli-experience-opt-out': '1' } }), resolved)).toBe(false)
  })

  it('refuses non-document fetches and non-HTML accepts', () => {
    expect(shouldInstrumentRequest(request({ headers: { 'sec-fetch-dest': 'empty' } }), resolved)).toBe(false)
    expect(shouldInstrumentRequest(request({ headers: { accept: 'application/json' } }), resolved)).toBe(false)
  })

  it('is inert when disabled', () => {
    expect(shouldInstrumentRequest(request(), null)).toBe(false)
  })
})
