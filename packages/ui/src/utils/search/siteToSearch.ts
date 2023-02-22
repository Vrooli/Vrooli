import i18next from 'i18next';
import { Session } from "@shared/consts";
import { getUserLanguages, normalizeText, removeEmojis, removePunctuation } from 'utils/display';
import { FilterOptionsState } from '@mui/material';
import { CommonKey } from '@shared/translations';

/**
 * A search item before it is translated into the user's language.
 */
export interface PreSearchItem {
    /**
     * Key for the label
     */
    label: CommonKey;
    /**
     * Arguments for the label
     */
    labelArgs?: { [key: string]: string | number };
    /**
     * Keys (and possibly arguments) for the keywords
     */
    keywords?: readonly (CommonKey | ({ key: CommonKey; } & { [key: string]: string | number }))[];
    /**
     * The link/value that will be used when the user selects the item.
     */
    value: string;
}

/**
 * A translated search item for shortcuts, settings items, and any other non-user-generated 
 * data that can be searched
 */
export interface SearchItem {
    /**
     * What the user will see in the search results.
     */
    label: string;
    /**
     * Words/phrases that the user can use to search for this item. Includes label, which is 
     * shaped to help with searching (lowercase, no accents, etc.)
     */
    keywords?: string[];
    /**
     * Keywords that are not shaped. Useful if you want to display this next to the label.
     */
    unshapedKeywords?: string[];
    /**
     * The link/value that will be used when the user selects the item.
     */
    value: string;
}

/**
 * Shapes a string to help with searching. Removes punctuation, accents, and emojis.
 * @param text The text to shape.
 * @returns The shaped text.
 */
export const shapeSearchText = (text: string) => {
    if (!text) {
        console.warn('No text provided to shapeSearchText');
        return '';
    }
    // Remove extra whitespace
    let shaped = text.trim();
    // Normalize the search term
    shaped = normalizeText(shaped);
    // Remove punctuation
    shaped = removePunctuation(shaped);
    // Remove emojis
    shaped = removeEmojis(shaped);
    // Lowercase
    shaped = shaped.toLowerCase();
    return shaped;
}

/**
 * Converts a list of PreSearchItems into a list of SearchItems.
 * @param items The list of PreSearchItems to convert.
 * @param session The current session.
 */
export const translateSearchItems = (items: PreSearchItem[], session: Session): SearchItem[] => {
    const lng = getUserLanguages(session)[0];
    return items.map(item => {
        const label = i18next.t(`common:${item.label}`, { ...(item.labelArgs ?? {}), lng });
        let keywords = [shapeSearchText(label)];
        let unshapedKeywords = [label];
        for (const keyword of item.keywords ?? []) {
            if (typeof keyword === 'string') {
                const keywordText = i18next.t(`common:${keyword}`, { lng });
                keywords.push(shapeSearchText(keywordText));
                unshapedKeywords.push(keywordText);
            } else {
                const keywordText = i18next.t(`common:${keyword.key}`, { ...keyword, lng });
                keywords.push(shapeSearchText(keywordText));
                unshapedKeywords.push(keywordText);
            }
        }
        return {
            label,
            keywords,
            unshapedKeywords,
            value: item.value,
        };
    });
}

/**
 * Finds matches for the given search term in the given list of search items
 * @param items The list of translated search items to search.
 * @param state The state of the autocomplete search bar
 * @returns A list of matches.
 */
export const findSearchResults = (items: SearchItem[], { inputValue }: FilterOptionsState<SearchItem>): SearchItem[] => {
    console.log('findSearchResults start', { ...items }, inputValue)
    // Shape the search term
    const shapedTerm = shapeSearchText(inputValue);
    // Filter out items which don't contain the shaped search term in their keywords
    const matches = items.filter(item => item.keywords?.some(keyword => keyword.includes(shapedTerm)));
    // Sort. Exact matches first, then by number of keywords that match, then alphabetically
    return matches.sort((a, b) => {
        const aExact = a.keywords?.some(keyword => keyword === shapedTerm) ?? false;
        const bExact = b.keywords?.some(keyword => keyword === shapedTerm) ?? false;
        if (aExact && !bExact) {
            return -1;
        } else if (!aExact && bExact) {
            return 1;
        } else {
            const aMatchCount = a.keywords?.filter(keyword => keyword.includes(shapedTerm)).length ?? 0;
            const bMatchCount = b.keywords?.filter(keyword => keyword.includes(shapedTerm)).length ?? 0;
            if (aMatchCount > bMatchCount) {
                return -1;
            } else if (aMatchCount < bMatchCount) {
                return 1;
            } else {
                return a.label.localeCompare(b.label);
            }
        }
    });
}