import { SearchStringQueryParams } from "../models/types";

type P = SearchStringQueryParams;

/**
 * Base query for a string field
 */
const base = ({ insensitive }: P) => ({ ...insensitive });

/**
 * Query for some translated field 
 */
const transField = <Field extends string>(
    { insensitive, languages }: P,
    field: Field
) => ({ some: { language: languages ? { in: languages } : undefined, [field]: { ...insensitive } } });

/**
 * Wraps a query with a field name
 */
const wrap = <
    Field extends string,
    Wrapped extends { [x: string]: any }
>(
    field: Field,
    wrapped: Wrapped
): { [x in Field]: Wrapped } => ({ [field]: wrapped }) as any;
 
/**
 * Maps any search string fields to their corresponding Prisma query.
 * 
 * Example: SearchStringMap['translationsName'] = ({ translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } })
 */
export const SearchStringMap = {
    description: (p: P) => base(p),
    descriptionWrapped: (p: P) => wrap('description', SearchStringMap.description(p)),
    details: (p: P) => base(p),
    detailsWrapped: (p: P) => wrap('details', SearchStringMap.details(p)),
    handle: (p: P) => base(p),
    handleWrapped: (p: P) => wrap('handle', SearchStringMap.handle(p)),
    label: (p: P) => base(p),
    labelWrapped: (p: P) => wrap('label', SearchStringMap.label(p)),
    labels: ({ insensitive }: P) => ({ some: { label: { label: { ...insensitive } } } }),
    labelsWrapped: (p: P) => wrap('labels', SearchStringMap.labels(p)),
    link: (p: P) => base(p),
    linkWrapped: (p: P) => wrap('link', SearchStringMap.link(p)),
    message: (p: P) => base(p),
    messageWrapped: (p: P) => wrap('message', SearchStringMap.message(p)),
    name: (p: P) => base(p),
    nameWrapped: (p: P) => wrap('name', SearchStringMap.name(p)),
    reason: (p: P) => base(p),
    reasonWrapped: (p: P) => wrap('reason', SearchStringMap.reason(p)),
    title: (p: P) => base(p),
    titleWrapped: (p: P) => wrap('title', SearchStringMap.title(p)),
    transBio: (p: P) => transField(p, 'bio'),
    transBioWrapped: (p: P) => wrap('translations', transField(p, 'bio')),
    transDescription: (p: P) => transField(p, 'description'),
    transDescriptionWrapped: (p: P) => wrap('translations', transField(p, 'description')),
    transName: (p: P) => transField(p, 'name'),
    transNameWrapped: (p: P) => wrap('translations', transField(p, 'name')),
    transQuestionText: (p: P) => transField(p, 'questionText'),
    transQuestionTextWrapped: (p: P) => wrap('translations', transField(p, 'questionText')),
    transSummary: (p: P) => transField(p, 'summary'),
    transSummaryWrapped: (p: P) => wrap('translations', transField(p, 'summary')),
    transResponseWrapped: (p: P) => wrap('translations', transField(p, 'response')),
    transText: (p: P) => transField(p, 'text'),
    transTextWrapped: (p: P) => wrap('translations', transField(p, 'text')),
    tag: (p: P) => base(p),
    tagWrapped: (p: P) => wrap('tag', SearchStringMap.tag(p)),
    tags: ({ insensitive }: P) => ({ some: { tag: { tag: { ...insensitive } } } }),
    tagsWrapped: (p: P) => wrap('tags', SearchStringMap.tags(p)),
}