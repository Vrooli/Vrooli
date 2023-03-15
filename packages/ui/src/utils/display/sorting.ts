import { CommonKey } from '@shared/translations';
import i18next from 'i18next';

export type LabelledSortOption<SortBy> = { label: string, value: SortBy };

/**
 * Converts sort options to label/value pairs
 * @param sortValues The sort values to convert
 * @returns Array of { label, value }
 */
export function labelledSortOptions<SortBy>(sortValues: { [x in keyof CommonKey]?: any }): LabelledSortOption<SortBy>[] {
    if (!sortValues) return [];
    return Object.keys(sortValues).map((key) => ({
        label: (i18next.t(key as CommonKey, key)) as unknown as string,
        value: key as unknown as SortBy,
    }))
}