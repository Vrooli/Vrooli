#!/usr/bin/env node
/**
 * `vrooli-experience` — inspect, prune, and analyze local experience sessions.
 *
 * The CLI reads the same append-only JSONL files the scenario server writes.
 * It has no dependency on a running scenario or on the repository layout.
 */

import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import { analyzeExperience, loadDeclaredSpec, renderFrictionMarkdown } from './report.js'
import {
  createObservedEvidence,
  createPacketObservedBlock,
  deriveAdvisoryFindings,
  deriveCorrectionCandidates,
  validateReport,
} from './contracts.js'
import { experienceDataDir, experienceScenarioName, readExperiencePolicy } from './gate.js'
import { pruneSessions, readSessions, sessionStatus } from './store.js'
import type { ExperienceObservedFlow } from './types.js'

interface Args {
  command: string
  sessions?: string
  scenario?: string
  spec?: string
  out?: string
  build?: string
  minCount?: number
  maxFlows?: number
  days?: number
  max?: number
  json: boolean
  quiet: boolean
}

function usage(): string {
  return [
    'Usage: vrooli-experience <command> [options]',
    '',
    'This is a dedicated binary, not a `vrooli` subcommand.',
    '',
    'Commands:',
    '  report     Analyze recorded sessions into a friction report (default)',
    '  status     Show session count, size, and retention settings',
    '  prune      Delete expired or excess session files',
    '  path       Print the default session directory',
    '',
    'Options:',
    '  --sessions <dir>   Session directory (or its parent)',
    '  --scenario <name>  Scenario name; selects the default directory',
    '  --spec <dir>       Scenario root or experience/ directory for journey diff',
    '  --out <dir>        Report output directory',
    '  --build <id>       Expected build identity (mismatch marks evidence stale)',
    '  --min-count <n>    Minimum repetitions for a flow (default 3)',
    '  --max-flows <n>    Maximum flows in the report (default 12)',
    '  --days <n>         Retention age for prune (default 30)',
    '  --max <n>          Maximum session files for prune (default 500)',
    '  --json             Print the report JSON to stdout instead of a summary',
    '  --quiet            Only print output paths',
    '  -h, --help         Show this help',
    '',
    'Examples:',
    '  vrooli-experience report --scenario git-control-tower',
    '  vrooli-experience report --sessions ~/.vrooli/experience/sessions/git-control-tower \\',
    '    --spec scenarios/git-control-tower/experience --out /tmp/report',
  ].join('\n')
}

function parseArgs(argv: string[]): Args {
  const args: Args = { command: 'report', json: false, quiet: false }
  const valueOptions = new Set(['--sessions', '--scenario', '--spec', '--out', '--build', '--min-count', '--max-flows', '--days', '--max'])
  let index = 0
  if (argv[0] && !argv[0].startsWith('-')) {
    args.command = argv[0]
    index = 1
  }
  for (; index < argv.length; index += 1) {
    const token = argv[index]!
    if (token === '-h' || token === '--help') {
      args.command = 'help'
      continue
    }
    if (token === '--json') {
      args.json = true
      continue
    }
    if (token === '--quiet') {
      args.quiet = true
      continue
    }
    if (valueOptions.has(token)) {
      const value = argv[index + 1]
      if (value === undefined) throw new Error(`${token} requires a value`)
      index += 1
      switch (token) {
        case '--sessions': args.sessions = value; break
        case '--scenario': args.scenario = value; break
        case '--spec': args.spec = value; break
        case '--out': args.out = value; break
        case '--build': args.build = value; break
        case '--min-count': args.minCount = Number.parseInt(value, 10); break
        case '--max-flows': args.maxFlows = Number.parseInt(value, 10); break
        case '--days': args.days = Number.parseInt(value, 10); break
        case '--max': args.max = Number.parseInt(value, 10); break
        default: break
      }
      continue
    }
    throw new Error(`Unknown argument: ${token}`)
  }
  return args
}

function resolveScenario(args: Args): string {
  return experienceScenarioName(args.scenario || path.basename(process.cwd()))
}

function resolveSessionsDir(args: Args, scenario: string): string {
  if (args.sessions) return path.resolve(args.sessions)
  return experienceDataDir(scenario)
}

function resolveSpecDir(args: Args, scenario: string): string | undefined {
  if (args.spec) return path.resolve(args.spec)
  const local = path.resolve(process.cwd(), 'scenarios', scenario, 'experience')
  return fs.existsSync(path.join(local, 'index.json')) ? local : undefined
}

function writeFile(file: string, content: string): void {
  fs.mkdirSync(path.dirname(file), { recursive: true })
  fs.writeFileSync(file, content, 'utf-8')
}

function runReport(args: Args, scenario: string): number {
  const sessionsDir = resolveSessionsDir(args, scenario)
  const specDir = resolveSpecDir(args, scenario)
  const status = sessionStatus(sessionsDir)
  if (status.sessions === 0) {
    process.stderr.write(`No sessions found in ${sessionsDir}\n`)
    process.stderr.write('Open a scenario locally and interact with it, then run this again.\n')
    return 0
  }

  const { batches, malformed, files } = readSessions(sessionsDir)
  if (batches.length === 0) {
    process.stderr.write(`Found ${files} file(s) but no parseable batches in ${sessionsDir}\n`)
    return 0
  }

  // Capability declarations live in the same per-scenario policy the server
  // reads. Prefer the scenario's own file so a report run from the repo root
  // sees the same declarations the UI saw.
  const scenarioPolicy = path.resolve(process.cwd(), 'scenarios', scenario, '.vrooli', 'experience.json')
  const policy = fs.existsSync(scenarioPolicy) ? readExperiencePolicy(scenarioPolicy) : readExperiencePolicy()
  const capabilities = policy?.capabilities
  const { report, observedFlows } = analyzeExperience(batches, {
    scenario,
    ...(specDir ? { specDir } : {}),
    ...(capabilities ? { capabilities } : {}),
    ...(args.minCount !== undefined ? { minFlowCount: args.minCount } : {}),
    ...(args.maxFlows !== undefined ? { maxFlows: args.maxFlows } : {}),
  })

  if (args.json) {
    process.stdout.write(`${JSON.stringify(report, null, 2)}\n`)
  }

  const observedBehavior = createObservedEvidence(report, { ...(args.build ? { expectedBuild: args.build } : {}) })
  const advisoryFindings = deriveAdvisoryFindings(report)
  const correctionCandidates = deriveCorrectionCandidates(report)
  const packetObserved = createPacketObservedBlock(report)
  const validation = validateReport(report)
  if (!validation.ok || !validation.digestValid) {
    process.stderr.write(
      `Warning: generated report failed its own validation (${validation.ok ? 'digest mismatch' : validation.errors.join('; ')})\n`,
    )
  }

  const outDir = path.resolve(args.out || path.join(os.homedir(), '.vrooli', 'experience', 'reports', scenario))
  const jsonPath = path.join(outDir, 'friction-report.json')
  const markdownPath = path.join(outDir, 'friction-report.md')
  const flowsPath = path.join(outDir, 'observed-flows.json')
  const advisoryPath = path.join(outDir, 'advisory-findings.json')
  const candidatesPath = path.join(outDir, 'correction-candidates.json')
  const behaviorPath = path.join(outDir, 'observed-behavior.json')
  const packetPath = path.join(outDir, 'packet-observed.json')
  writeFile(jsonPath, `${JSON.stringify(report, null, 2)}\n`)
  writeFile(markdownPath, renderFrictionMarkdown(report))
  writeFile(flowsPath, `${JSON.stringify({ flows: observedFlows as ExperienceObservedFlow[] }, null, 2)}\n`)
  writeFile(advisoryPath, `${JSON.stringify({ schemaVersion: 'experience-advisory-findings/v1', findings: advisoryFindings }, null, 2)}\n`)
  writeFile(candidatesPath, `${JSON.stringify({ schemaVersion: 'experience-correction-candidates/v1', candidates: correctionCandidates }, null, 2)}\n`)
  writeFile(behaviorPath, `${JSON.stringify(observedBehavior, null, 2)}\n`)
  writeFile(packetPath, `${JSON.stringify(packetObserved, null, 2)}\n`)

  if (!args.quiet) {
    const declared = loadDeclaredSpec(specDir).journeys.length
    process.stdout.write(`${report.totals.sessions} sessions, ${report.totals.actions} actions, ${report.recommendations.length} recommendation(s)\n`)
    process.stdout.write(`Evidence: ${observedBehavior.state}; ${advisoryFindings.length} advisory finding(s), ${correctionCandidates.length} correction candidate(s)\n`)
    if (malformed > 0) process.stdout.write(`${malformed} malformed line(s) skipped\n`)
    if (declared > 0) process.stdout.write(`${declared} declared journey(s) compared\n`)
  }
  process.stdout.write(`${jsonPath}\n${markdownPath}\n${flowsPath}\n${advisoryPath}\n${candidatesPath}\n${behaviorPath}\n${packetPath}\n`)
  return 0
}

function main(argv: string[]): number {
  let args: Args
  try {
    args = parseArgs(argv)
  } catch (error) {
    process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n\n${usage()}\n`)
    return 2
  }

  if (args.command === 'help') {
    process.stdout.write(`${usage()}\n`)
    return 0
  }

  const scenario = resolveScenario(args)
  const sessionsDir = resolveSessionsDir(args, scenario)

  switch (args.command) {
    case 'report':
      return runReport(args, scenario)
    case 'status': {
      const status = sessionStatus(sessionsDir)
      process.stdout.write(`${JSON.stringify({
        scenario,
        sessionDir: status.sessionDir,
        sessions: status.sessions,
        bytes: status.bytes,
        oldest: status.oldest ? new Date(status.oldest).toISOString() : null,
        newest: status.newest ? new Date(status.newest).toISOString() : null,
      }, null, 2)}\n`)
      return 0
    }
    case 'prune': {
      const result = pruneSessions(sessionsDir, {
        maxAgeDays: args.days ?? 30,
        maxSessions: args.max ?? 500,
      })
      process.stdout.write(`removed ${result.removed} session(s), kept ${result.kept}\n`)
      return 0
    }
    case 'path':
      process.stdout.write(`${sessionsDir}\n`)
      return 0
    default:
      process.stderr.write(`Unknown command: ${args.command}\n\n${usage()}\n`)
      return 2
  }
}

process.exitCode = main(process.argv.slice(2))
