/**
 * Shared types for preview dialog and overlay functionality
 */

/**
 * Preview overlay state that can be displayed over the iframe
 */
export type PreviewOverlayState = {
  type: 'restart' | 'waiting' | 'error';
  message: string;
} | null;

/**
 * Preview location state passed through router navigation
 */
export interface PreviewLocationState {
  autoSelected?: boolean;
  [key: string]: unknown;
}

/**
 * Type guard to validate PreviewLocationState from unknown router state.
 * Ensures runtime type safety when accessing location.state.
 */
export const isPreviewLocationState = (state: unknown): state is PreviewLocationState => {
  if (state === null || state === undefined) {
    return false;
  }

  if (typeof state !== 'object') {
    return false;
  }

  const obj = state as Record<string, unknown>;

  // If autoSelected exists, it must be a boolean
  if ('autoSelected' in obj && typeof obj.autoSelected !== 'boolean') {
    return false;
  }

  return true;
}
