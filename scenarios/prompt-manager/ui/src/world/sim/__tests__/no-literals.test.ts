/**
 * The sim carries no behaviour numbers: every timing, speed, weight or
 * distance is a tuning lever. This scan allows structural constants only.
 */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const SIM_ROOT = resolve(import.meta.dirname, '..')
const EXEMPT_FILES = new Set(['rng.ts', 'hash.ts', 'seatMath.ts'])
// Small integers are indices, ordinal ranks and array sizes; the only
// fraction allowed is one half. Anything else must be a lever.
const ALLOWED = new Set(['0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0.5', '1e-9'])

function walk(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    if (statSync(full).isDirectory()) {
      if (entry === '__tests__' || entry === '__lint__') continue
      walk(full, out)
    } else if (entry.endsWith('.ts') && !EXEMPT_FILES.has(entry)) out.push(full)
  }
  return out
}

function stripCommentsAndStrings(source: string): string {
  return source
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/\/\/.*$/gm, '')
    .replace(/`(?:\\.|[^`\\])*`/g, '""')
    .replace(/'(?:\\.|[^'\\])*'/g, '""')
    .replace(/"(?:\\.|[^"\\])*"/g, '""')
}

describe('sim has no behaviour literals', () => {
  it('every numeric literal outside rng/hash is a small integer or one half', () => {
    const offenders: string[] = []
    for (const file of walk(SIM_ROOT)) {
      const code = stripCommentsAndStrings(readFileSync(file, 'utf8'))
      const lines = code.split('\n')
      lines.forEach((line, index) => {
        for (const match of line.matchAll(/(?<![\w.])(\d+(?:\.\d+)?(?:e-?\d+)?)(?![\w.])/g)) {
          const literal = match[1] ?? ''
          if (ALLOWED.has(literal)) continue
          offenders.push(`${file.replace(SIM_ROOT, 'sim')}:${index + 1}: ${literal}`)
        }
      })
    }
    expect(offenders).toEqual([])
  })
})
