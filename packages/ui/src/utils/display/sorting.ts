import i18next from 'i18next';
import { CommonKey } from 'types';

export type LabelledSortOption<SortBy> = { label: string, value: SortBy };

/**
 * Converts sort options to label/value pairs
 * @param sortValues The sort values to convert
 * @returns Array of { label, value }
 */
export function labelledSortOptions<SortBy>(sortValues: { [x in keyof CommonKey]?: any }, lng: string): LabelledSortOption<SortBy>[] {
    if (!sortValues) return [];
    return Object.keys(sortValues).map((key) => ({
        label: (i18next.t(`common:${key as CommonKey}`, { lng }) ?? key) as unknown as string,
        value: key as unknown as SortBy,
    }))
}