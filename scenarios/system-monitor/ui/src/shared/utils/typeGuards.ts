/**
 * Shared type-narrowing helpers for safely reading unknown payloads.
 */

export const extractString = (value: unknown): string | undefined =>
  typeof value === 'string' ? value : undefined;

export const extractBoolean = (value: unknown): boolean | undefined =>
  typeof value === 'boolean' ? value : undefined;

export const extractNumber = (value: unknown): number | undefined =>
  typeof value === 'number' ? value : undefined;
