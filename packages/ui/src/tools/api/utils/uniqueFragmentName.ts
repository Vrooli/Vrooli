import { SelectionType } from "../types";

/**
 * Format for unique fragment name
 */
export const uniqueFragmentName = (typename: string, actualSelectionType: SelectionType) => `${typename}_${actualSelectionType}`;
