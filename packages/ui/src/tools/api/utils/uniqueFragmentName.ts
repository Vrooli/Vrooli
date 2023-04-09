/**
 * Format for unique fragment name
 */
export const uniqueFragmentName = (typename: string, actualSelectionType: 'common' | 'full' | 'list' | 'nav') => `${typename}_${actualSelectionType}`;