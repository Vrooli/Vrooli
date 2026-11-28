/**
 * Enablement and request gating for experience telemetry.
 *
 * Default posture: on for loopback requests in a local run, off in tests, CI,
 * and for any non-loopback host (a tunnel or a LAN device). An operator can
 * force it with `VROOLI_EXPERIENCE_TELEMETRY=1`, and a browser or automation
 * client can opt out per request.
 */

import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import {
  EXPERIENCE_DEFAULTS,
  type ExperienceCapabilityDeclaration,
  type ExperiencePolicy,
  type ExperienceTelemetryOptions,
  type ResolvedExperienceTelemetry,
} from './types.js'

const TRUE_VALUES = new Set(['1', 'true', 'on', 'yes'])
const FALSE_VALUES = new Set(['0', 'false', 'off', 'no'])

export interface ExperienceRequestLike {
  method: string
  path: string
  headers: Record<string, string | string[] | undefined>
}

function header(headers: ExperienceRequestLike['headers'], name: string): string | undefined {
  const value = headers[name]
  const first = Array.isArray(value) ? value[0] : value
  const trimmed = first?.trim()
  return trimmed ? trimmed : undefined
}

function booleanFromEnv(value: string | undefined): boolean | undefined {
  if (value === undefined) return undefined
  const normalized = value.trim().toLowerCase()
  if (TRUE_VALUES.has(normalized)) return true
  if (FALSE_VALUES.has(normalized)) return false
  return undefined
}

/** Sanitize a scenario/service name for use as a directory segment. */
export function experienceScenarioName(value: string | undefined): string {
  const cleaned = (value || 'scenario')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9._-]+/g, '-')
    .replace(/\.{2,}/g, '-')
    .replace(/-{2,}/g, '-')
    .replace(/^[.-]+|[.-]+$/g, '')
  return cleaned.slice(0, 64) || 'scenario'
}

/** Default session directory for a scenario, outside the repository. */
export function experienceDataDir(scenario: string): string {
  return path.join(os.homedir(), '.vrooli', 'experience', 'sessions', experienceScenarioName(scenario))
}

function optionalBoolean(value: unknown): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined
}

function optionalNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function optionalCapabilities(value: unknown): ExperienceCapabilityDeclaration[] | undefined {
  if (!Array.isArray(value)) return undefined
  const parsed: ExperienceCapabilityDeclaration[] = []
  value.forEach((item) => {
    if (!item || typeof item !== 'object') return
    const record = item as Record<string, unknown>
    if (typeof record.id !== 'string' || !record.id.trim()) return
    // A declaration with no matcher can never be exercised; drop it rather than
    // reporting a permanent false "unobserved".
    if (typeof record.testId !== 'string' && typeof record.name !== 'string') return
    parsed.push({
      id: record.id.trim(),
      ...(typeof record.label === 'string' && record.label ? { label: record.label } : {}),
      ...(typeof record.testId === 'string' && record.testId ? { testId: record.testId } : {}),
      ...(typeof record.name === 'string' && record.name ? { name: record.name } : {}),
    })
  })
  return parsed.length > 0 ? parsed : undefined
}

/**
 * Read a per-scenario policy file. Precedence of locations:
 * explicit path, `VROOLI_EXPERIENCE_POLICY`, `<cwd>/.vrooli/experience.json`,
 * `<cwd>/../.vrooli/experience.json`, `<cwd>/../../.vrooli/experience.json`.
 *
 * The file is optional and malformed content is ignored: a bad policy file must
 * never prevent the UI from starting.
 */
export function readExperiencePolicy(explicitPath?: string): ExperiencePolicy | null {
  const candidates = explicitPath
    ? [explicitPath]
    : [
      process.env.VROOLI_EXPERIENCE_POLICY,
      path.resolve(process.cwd(), '.vrooli', 'experience.json'),
      path.resolve(process.cwd(), '..', '.vrooli', 'experience.json'),
      path.resolve(process.cwd(), '..', '..', '.vrooli', 'experience.json'),
    ].filter((candidate): candidate is string => Boolean(candidate))

  for (const candidate of candidates) {
    try {
      if (!fs.statSync(candidate).isFile()) continue
      const parsed = JSON.parse(fs.readFileSync(candidate, 'utf-8')) as Record<string, unknown>
      if (!parsed || typeof parsed !== 'object') continue
      return {
        ...(optionalBoolean(parsed.enabled) !== undefined ? { enabled: optionalBoolean(parsed.enabled)! } : {}),
        ...(optionalNumber(parsed.sample) !== undefined ? { sample: optionalNumber(parsed.sample)! } : {}),
        ...(optionalBoolean(parsed.captureLabels) !== undefined ? { captureLabels: optionalBoolean(parsed.captureLabels)! } : {}),
        ...(optionalBoolean(parsed.performance) !== undefined ? { performance: optionalBoolean(parsed.performance)! } : {}),
        ...(optionalBoolean(parsed.allowRemoteHosts) !== undefined ? { allowRemoteHosts: optionalBoolean(parsed.allowRemoteHosts)! } : {}),
        ...(optionalNumber(parsed.maxAgeDays) !== undefined ? { maxAgeDays: optionalNumber(parsed.maxAgeDays)! } : {}),
        ...(optionalNumber(parsed.maxSessions) !== undefined ? { maxSessions: optionalNumber(parsed.maxSessions)! } : {}),
        ...(optionalCapabilities(parsed.capabilities) !== undefined ? { capabilities: optionalCapabilities(parsed.capabilities)! } : {}),
      }
    } catch {
      // Try the next location; a broken policy file is not a startup failure.
    }
  }
  return null
}

/**
 * True when a Host header names the local machine. Deliberately excludes the
 * tunnel: the operator's own browsing records, a public visitor's does not.
 */
export function isLoopbackHost(host: string | undefined): boolean {
  if (!host) return false
  let value = host.trim().toLowerCase()
  if (value.startsWith('[')) {
    const end = value.indexOf(']')
    if (end === -1) return false
    value = value.slice(1, end)
  } else {
    const firstColon = value.indexOf(':')
    if (firstColon !== -1 && firstColon === value.lastIndexOf(':')) {
      value = value.slice(0, firstColon)
    }
  }
  return value === 'localhost'
    || value === '127.0.0.1'
    || value === '::1'
    || value === '0.0.0.0'
    || value.endsWith('.localhost')
}

/**
 * Resolve the effective configuration, or null when telemetry is off.
 *
 * Precedence: explicit option > `VROOLI_EXPERIENCE_TELEMETRY` > environment
 * default. The environment default is off under tests and CI so a test run
 * never writes the operator's session history.
 */
export function resolveExperienceTelemetry(
  input: boolean | ExperienceTelemetryOptions | undefined,
  env: NodeJS.ProcessEnv,
  scenario: string,
): ResolvedExperienceTelemetry | null {
  const options: ExperienceTelemetryOptions = typeof input === 'object' && input !== null ? input : {}
  const policy = options.policy ?? (options.policyFile === false ? null : readExperiencePolicy(options.policyFile))
  const explicit = typeof input === 'boolean' ? input : options.enabled
  const fromEnv = booleanFromEnv(env.VROOLI_EXPERIENCE_TELEMETRY)

  let enabled: boolean
  let source: string
  if (explicit !== undefined) {
    enabled = explicit
    source = 'option'
  } else if (fromEnv !== undefined) {
    enabled = fromEnv
    source = 'env'
  } else if (policy?.enabled !== undefined) {
    enabled = policy.enabled
    source = 'policy'
  } else if (env.NODE_ENV === 'test' || env.VITEST || env.CI) {
    enabled = false
    source = 'environment-default'
  } else {
    enabled = true
    source = 'environment-default'
  }

  if (!enabled) return null

  const sample = options.sample ?? policy?.sample ?? EXPERIENCE_DEFAULTS.sample
  const dataDir = options.dataDir || env.VROOLI_EXPERIENCE_DIR || experienceDataDir(scenario)
  const resolved: ResolvedExperienceTelemetry = {
    enabled: true,
    source,
    scenario: experienceScenarioName(scenario),
    ...(options.build || env.VROOLI_BUILD_IDENTITY ? { build: options.build || env.VROOLI_BUILD_IDENTITY } : {}),
    dataDir,
    assetPath: options.assetPath || EXPERIENCE_DEFAULTS.assetPath,
    ingestPath: options.ingestPath || EXPERIENCE_DEFAULTS.ingestPath,
    statusPath: options.statusPath || EXPERIENCE_DEFAULTS.statusPath,
    sample: Number.isFinite(sample) ? Math.min(1, Math.max(0, sample)) : 1,
    captureLabels: options.captureLabels ?? policy?.captureLabels ?? true,
    performance: options.performance ?? policy?.performance ?? true,
    maxBodyBytes: options.maxBodyBytes ?? EXPERIENCE_DEFAULTS.maxBodyBytes,
    maxEventsPerBatch: EXPERIENCE_DEFAULTS.maxEventsPerBatch,
    maxEventsPerSession: options.maxEventsPerSession ?? EXPERIENCE_DEFAULTS.maxEventsPerSession,
    maxAgeDays: options.maxAgeDays ?? policy?.maxAgeDays ?? EXPERIENCE_DEFAULTS.maxAgeDays,
    maxSessions: options.maxSessions ?? policy?.maxSessions ?? EXPERIENCE_DEFAULTS.maxSessions,
    allowRemoteHosts: options.allowRemoteHosts ?? booleanFromEnv(env.VROOLI_EXPERIENCE_ALLOW_REMOTE) ?? policy?.allowRemoteHosts ?? false,
    verbose: options.verbose ?? false,
    ...(options.assetFile ? { assetFile: options.assetFile } : {}),
  }
  return resolved
}

/** Per-request opt-out used by automation and by privacy-conscious tabs. */
function optedOut(headers: ExperienceRequestLike['headers']): boolean {
  const marker = header(headers, 'x-vrooli-experience')
  if (marker && FALSE_VALUES.has(marker.toLowerCase())) return true
  const optOut = header(headers, 'x-vrooli-experience-opt-out')
  return Boolean(optOut && TRUE_VALUES.has(optOut.toLowerCase()))
}

/**
 * Decide whether this response should receive the agent. Gating is per request
 * because Host, Accept, and Fetch Metadata are per request.
 */
export function shouldInstrumentRequest(
  req: ExperienceRequestLike,
  resolved: ResolvedExperienceTelemetry | null,
): boolean {
  if (!resolved) return false
  if (req.method !== 'GET' && req.method !== 'HEAD') return false
  if (optedOut(req.headers)) return false

  const pathname = req.path.split('?')[0] || '/'
  if (pathname === resolved.assetPath || pathname === resolved.ingestPath || pathname === resolved.statusPath) return false

  if (!resolved.allowRemoteHosts && !isLoopbackHost(header(req.headers, 'host'))) return false

  const destination = header(req.headers, 'sec-fetch-dest')
  if (destination && destination !== 'document' && destination !== 'iframe' && destination !== 'frame') return false

  const accept = header(req.headers, 'accept')
  if (accept && !accept.includes('text/html') && !accept.includes('*/*')) return false

  return true
}

/** Options that are safe to publish to the browser. */
export function experienceInjectionConfig(resolved: ResolvedExperienceTelemetry): {
  assetPath: string
  ingestPath: string
  scenario: string
  sample: number
  captureLabels: boolean
  performance: boolean
  maxEvents: number
  build?: string
} {
  return {
    assetPath: resolved.assetPath,
    ingestPath: resolved.ingestPath,
    scenario: resolved.scenario,
    ...(resolved.build ? { build: resolved.build } : {}),
    sample: resolved.sample,
    captureLabels: resolved.captureLabels,
    performance: resolved.performance,
    maxEvents: resolved.maxEventsPerSession,
  }
}
