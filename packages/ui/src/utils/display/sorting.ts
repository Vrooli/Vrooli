/* eslint-disable @typescript-eslint/no-redeclare */
import { ValueOf } from '@shared/consts';

/**
 * Converts GraphQL sort values to User-Friendly labels
 */
 export const SortValueToLabelMap = {
    'CommentsAsc': 'Comments ↑',
    'CommentsDesc': 'Comments ↓',
    'LastViewedAsc': 'Viewed ↑',
    'LastViewedDesc': 'Viewed ↓',
    'StarsAsc': 'Stars ↑',
    'StarsDesc': 'Stars ↓',
    'VotesAsc': 'Votes ↑',
    'VotesDesc': 'Votes ↓',
    'ForksAsc': 'Forks ↑',
    'ForksDesc': 'Forks ↓',
    'DateCreatedAsc': 'Old',
    'DateCreatedDesc': 'New',
    'DateStartedAsc': 'Started ↑',
    'DateStartedDesc': 'Started ↓',
    'DateUpdatedAsc': 'Recent ↑',
    'DateUpdatedDesc': 'Recent ↓',
}
export type SortValueToLabelMap = ValueOf<typeof SortValueToLabelMap>;

export type LabelledSortOption<SortBy> = { label: string, value: SortBy };

/**
 * Converts sort options to label/value pairs
 * @param sortValues The sort values to convert
 * @returns Array of { label, value }
 */
export function labelledSortOptions<SortBy>(sortValues: any): LabelledSortOption<SortBy>[] {
    return Object.keys(sortValues).map((key) => ({
        label: SortValueToLabelMap[key],
        value: key as unknown as SortBy,
    }))
}