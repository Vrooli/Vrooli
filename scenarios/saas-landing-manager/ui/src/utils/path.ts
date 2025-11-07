/**
 * Normalizes a base path by removing trailing slashes and handling empty/root cases
 */
export function normalizeBasePath(value?: string): string {
  if (!value) {
    return '';
  }

  const trimmed = value.trim();
  if (!trimmed || trimmed === '/') {
    return '';
  }

  return trimmed.endsWith('/') ? trimmed.slice(0, -1) : trimmed;
}
