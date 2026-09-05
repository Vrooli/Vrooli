import { useMemo, useState } from 'react'
import { selectors } from '@/constants/selectors'
import { WorldTuningSchema, type TuningOverride, type WorldTuning } from '../config'
import { collectLevers } from '../config/tuningDocs'

/** Groups the dev panel exposes; the rest are build-time or need a reload. */
const LIVE_GROUPS = ['sim', 'layout', 'camera', 'labels', 'actor', 'quality'] as const

interface LeversPanelProps {
  tuning: WorldTuning
  override: TuningOverride
  onChange: (override: TuningOverride) => void
  onReset: () => void
}

function setPath(target: Record<string, unknown>, path: string[], value: unknown): void {
  let cursor = target
  path.slice(0, -1).forEach((key) => {
    const next = cursor[key]
    if (typeof next !== 'object' || next === null) {
      cursor[key] = {}
    }
    cursor = cursor[key] as Record<string, unknown>
  })
  const last = path[path.length - 1]
  if (last !== undefined) cursor[last] = value
}

function readPath(source: unknown, path: string[]): unknown {
  let cursor: unknown = source
  for (const key of path) {
    if (typeof cursor !== 'object' || cursor === null) return undefined
    cursor = (cursor as Record<string, unknown>)[key]
  }
  return cursor
}

/**
 * Dev-only live editor for numeric and boolean levers. Every edit is
 * re-validated by the schema; invalid values are refused with the message
 * shown inline. Never shipped in production bundles.
 */
export function LeversPanel({ tuning, override, onChange, onReset }: LeversPanelProps) {
  const [group, setGroup] = useState<(typeof LIVE_GROUPS)[number]>('sim')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const rows = useMemo(() => collectLevers(undefined, tuning).filter((r) => r.path.startsWith(`${group}.`)), [group, tuning])

  const commit = (path: string, raw: string, kind: 'number' | 'boolean') => {
    const keys = path.split('.')
    const value = kind === 'boolean' ? raw === 'true' : Number(raw)
    const next = structuredClone(override) as Record<string, unknown>
    setPath(next, keys, value)
    const result = WorldTuningSchema.safeParse(mergeForCheck(tuning, next))
    if (!result.success) {
      const issue = result.error.issues.find((i) => i.path.join('.') === path) ?? result.error.issues[0]
      setErrors((prev) => ({ ...prev, [path]: issue?.message ?? 'invalid' }))
      return
    }
    setErrors((prev) => {
      const { [path]: _dropped, ...rest } = prev
      return rest
    })
    onChange(next as TuningOverride)
  }

  return (
    <div className="space-y-2 text-xs" data-testid={selectors.world.settings.levers}>
      <div className="flex flex-wrap items-center gap-1">
        {LIVE_GROUPS.map((id) => (
          <button key={id} type="button" onClick={() => setGroup(id)} aria-pressed={group === id} className={group === id ? 'rounded bg-primary px-2 py-0.5 text-primary-foreground' : 'rounded border border-border px-2 py-0.5 hover:bg-muted'}>
            {id}
          </button>
        ))}
        <button type="button" onClick={onReset} className="ml-auto rounded border border-border px-2 py-0.5 hover:bg-muted" data-testid={selectors.world.settings.leversReset}>
          Reset
        </button>
      </div>
      <div className="max-h-64 overflow-auto rounded border border-border">
        <table className="w-full">
          <tbody>
            {rows.map((row) => {
              const keys = row.path.split('.')
              const current = readPath(tuning, keys)
              const kind = typeof current === 'boolean' ? 'boolean' : typeof current === 'number' ? 'number' : null
              if (!kind) return null
              return (
                <tr key={row.path} className="border-b border-border/60 last:border-0">
                  <td className="px-2 py-1 align-top">
                    <label htmlFor={`lever-${row.path}`} className="font-mono" title={row.description}>
                      {row.path.slice(group.length + 1)}
                    </label>
                    {errors[row.path] && <p className="text-red-600 dark:text-red-400">{errors[row.path]}</p>}
                  </td>
                  <td className="px-2 py-1 text-right">
                    {kind === 'boolean' ? (
                      <input id={`lever-${row.path}`} type="checkbox" checked={current === true} onChange={(e) => commit(row.path, String(e.target.checked), 'boolean')} />
                    ) : (
                      <input
                        id={`lever-${row.path}`}
                        type="number"
                        step="any"
                        defaultValue={String(current)}
                        onBlur={(e) => commit(row.path, e.target.value, 'number')}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') commit(row.path, (e.target as HTMLInputElement).value, 'number')
                        }}
                        className="w-24 rounded border border-border bg-background px-1 text-right tabular-nums"
                      />
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function mergeForCheck(base: WorldTuning, override: Record<string, unknown>): unknown {
  const merge = (a: unknown, b: unknown): unknown => {
    if (typeof a !== 'object' || a === null || Array.isArray(a) || typeof b !== 'object' || b === null) return b === undefined ? a : b
    const out: Record<string, unknown> = { ...(a as Record<string, unknown>) }
    for (const [k, v] of Object.entries(b as Record<string, unknown>)) out[k] = merge(out[k], v)
    return out
  }
  return merge(base, override)
}
