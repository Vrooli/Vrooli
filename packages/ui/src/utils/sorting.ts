/* eslint-disable @typescript-eslint/no-redeclare */
import { ValueOf } from '@local/shared';

/**
 * Converts GraphQL sort values to User-Friendly labels
 */
 export const SortValueToLabelMap = {
    'AlphabeticalAsc': 'A-Z',
    'AlphabeticalDesc': 'Z-A',
    'CommentsDesc': 'Most Comments',
    'StarsDesc': 'Most Stars',
    'ForksDesc': 'Most Forks',
}
export type SortValueToLabelMap = ValueOf<typeof SortValueToLabelMap>;

export type LabelledSortOption<SortBy> = { label: string, value: SortBy };

export function labelledSortOptions<SortBy>(sortValues: any): LabelledSortOption<SortBy>[] {
    return Object.keys(sortValues).map((key) => ({
        label: SortValueToLabelMap[key],
        value: key as unknown as SortBy,
    }))
}