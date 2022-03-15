/* eslint-disable @typescript-eslint/no-redeclare */
import { ValueOf } from '@local/shared';

/**
 * Converts GraphQL sort values to User-Friendly labels
 */
 export const SortValueToLabelMap = {
    'CommentsAsc': 'Least Comments',
    'CommentsDesc': 'Comments',
    'StarsAsc': 'Least Stars',
    'StarsDesc': 'Stars',
    'ForksAsc': 'Least Forks',
    'ForksDesc': 'Forks',
    'DateCreatedAsc': 'Oldest',
    'DateCreatedDesc': 'Newest',
    'DateUpdatedAsc': 'Least Recent',
    'DateUpdatedDesc': 'Most Recent',
}
export type SortValueToLabelMap = ValueOf<typeof SortValueToLabelMap>;

export type LabelledSortOption<SortBy> = { label: string, value: SortBy };

export function labelledSortOptions<SortBy>(sortValues: any): LabelledSortOption<SortBy>[] {
    return Object.keys(sortValues).map((key) => ({
        label: SortValueToLabelMap[key],
        value: key as unknown as SortBy,
    }))
}