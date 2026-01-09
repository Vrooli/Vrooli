/**
 * Type guard to check if a value is a plain object (record)
 */
export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/**
 * Extracts the result object from a diagnostics section
 * Handles both nested status.result and direct result patterns
 */
export function getSectionResult(section: unknown): Record<string, unknown> | null {
  if (!isRecord(section)) {
    return null
  }

  const status = section.status
  if (isRecord(status) && isRecord(status.result)) {
    return status.result
  }

  if (isRecord(section.result)) {
    return section.result
  }

  return section
}

/**
 * Safely converts unknown value to number
 */
export function toNumber(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined
}

/**
 * Safely converts unknown value to string
 */
export function toString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}
