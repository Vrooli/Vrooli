/**
 * The parsed, validated tuning file. Importing this module throws when
 * world.tuning.json violates its schema, so a bad lever never reaches a frame.
 */
import raw from './world.tuning.json'
import { WorldTuningSchema, type WorldTuning } from './tuning.schema'

export function parseTuning(input: unknown): WorldTuning {
  const result = WorldTuningSchema.safeParse(input)
  if (!result.success) {
    const issues = result.error.issues
      .map((issue) => `${issue.path.join('.') || '<root>'}: ${issue.message}`)
      .join('\n')
    throw new Error(`world.tuning.json is invalid:\n${issues}`)
  }
  return result.data
}

export const tuning: WorldTuning = parseTuning(raw)

/**
 * A deep partial override merged over the shipped tuning. Used by the dev-only
 * levers panel; production code never calls this.
 */
export type TuningOverride = DeepPartial<WorldTuning>

type DeepPartial<T> = T extends readonly unknown[]
  ? T
  : T extends object
    ? { [K in keyof T]?: DeepPartial<T[K]> }
    : T

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function mergeDeep(base: unknown, override: unknown): unknown {
  if (!isPlainObject(base) || !isPlainObject(override)) return override === undefined ? base : override
  const out: Record<string, unknown> = { ...base }
  for (const [key, value] of Object.entries(override)) {
    if (value === undefined) continue
    out[key] = mergeDeep(base[key], value)
  }
  return out
}

/** Merge an override over the shipped tuning and re-validate the result. */
export function withTuningOverride(override: TuningOverride, base: WorldTuning = tuning): WorldTuning {
  return parseTuning(mergeDeep(base, override))
}
