import { SearchStringQueryParams } from "../models/types";

type P = SearchStringQueryParams;

/**
 * Maps any search string fields to their corresponding Prisma query.
 * 
 * Example: SearchStringMap['translationsName'] = ({ translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } })
 */
export const SearchStringMap = {
    details: ({ insensitive }: P) => ({ details: { ...insensitive } }),
    label: ({ insensitive }: P) => ({ label: { ...insensitive } }),
    link: ({ insensitive }: P) => ({ link: { ...insensitive } }),
    reason: ({ insensitive }: P) => ({ reason: { ...insensitive } }),
    translationsBio: ({ insensitive, languages }: P) => ({ translations: { some: { language: languages ? { in: languages } : undefined, bio: { ...insensitive } } } }),
    translationsDescription: ({ insensitive, languages }: P) => ({ translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } }),
    translationsName: ({ insensitive, languages }: P) => ({ translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } }),
    translationsSummary: ({ insensitive, languages }: P) => ({ translations: { some: { language: languages ? { in: languages } : undefined, summary: { ...insensitive } } } }),
    translationsText: ({ insensitive, languages }: P) => ({ translations: { some: { language: languages ? { in: languages } : undefined, text: { ...insensitive } } } }),
    tags: ({ insensitive }: P) => ({ tags: { some: { tag: { tag: { ...insensitive } } } } }),
}