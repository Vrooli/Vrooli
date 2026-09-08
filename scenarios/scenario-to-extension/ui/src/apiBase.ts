import { resolveApiBase } from '@vrooli/api-base'

/** Canonical UI source contract for proxy and tunnel safe API resolution. */
export function resolveScenarioApiBase(): string {
  return resolveApiBase({ appendSuffix: true })
}
