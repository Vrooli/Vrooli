/**
 * Shared form utility functions
 */

/**
 * Check if a form has been modified from its original state.
 * Uses JSON.stringify for deep equality comparison.
 *
 * @param current - Current form values
 * @param original - Original form values
 * @returns True if the form has unsaved changes
 */
export function isFormDirty<T>(current: T, original: T): boolean {
  return JSON.stringify(current) !== JSON.stringify(original);
}

/**
 * Check if a form has been modified, with normalization applied before comparison.
 * Useful when forms need trimming, default value handling, or other transformations
 * before comparison.
 *
 * @param current - Current form values
 * @param original - Original form values
 * @param normalizer - Function to normalize form values before comparison
 * @returns True if the normalized forms differ
 */
export function isFormDirtyNormalized<T, N>(
  current: T,
  original: T,
  normalizer: (form: T) => N
): boolean {
  return JSON.stringify(normalizer(current)) !== JSON.stringify(normalizer(original));
}
