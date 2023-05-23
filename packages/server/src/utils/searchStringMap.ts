import { SearchStringQueryParams } from "../models/types";

type P = SearchStringQueryParams;

/**
 * Query for some translated field 
 */
const transField = <Field extends string>(
    { insensitive, languages }: P,
    field: Field,
) => ({ some: { language: languages ? { in: languages } : undefined, [field]: { ...insensitive } } });

/**
 * Maps any search string fields to their corresponding Prisma query.
 * 
 * Example: SearchStringMap['translationsName'] = ({ translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } })
 */
export const SearchStringMap = {
    description: ({ insensitive }: P) => ({ ...insensitive }),
    descriptionWrapped: (p: P) => ({ description: SearchStringMap.description(p) }),
    details: ({ insensitive }: P) => ({ ...insensitive }),
    detailsWrapped: (p: P) => ({ details: SearchStringMap.details(p) }),
    handle: ({ insensitive }: P) => ({ ...insensitive }),
    handleWrapped: (p: P) => ({ handle: SearchStringMap.handle(p) }),
    label: ({ insensitive }: P) => ({ ...insensitive }),
    labelWrapped: (p: P) => ({ label: SearchStringMap.label(p) }),
    labels: ({ insensitive }: P) => ({ some: { label: { label: { ...insensitive } } } }),
    labelsWrapped: (p: P) => ({ labels: SearchStringMap.labels(p) }),
    labelsOwnerWrapped: ({ insensitive }: P) => ({ labels: { some: { label: { ...insensitive } } } }),
    link: ({ insensitive }: P) => ({ ...insensitive }),
    linkWrapped: (p: P) => ({ link: SearchStringMap.link(p) }),
    message: ({ insensitive }: P) => ({ ...insensitive }),
    messageWrapped: (p: P) => ({ message: SearchStringMap.message(p) }),
    name: ({ insensitive }: P) => ({ ...insensitive }),
    nameWrapped: (p: P) => ({ name: SearchStringMap.name(p) }),
    reason: ({ insensitive }: P) => ({ ...insensitive }),
    reasonWrapped: (p: P) => ({ reason: SearchStringMap.reason(p) }),
    title: ({ insensitive }: P) => ({ ...insensitive }),
    titleWrapped: (p: P) => ({ title: SearchStringMap.title(p) }),
    transBio: (p: P) => transField(p, "bio"),
    transBioWrapped: (p: P) => ({ translations: transField(p, "bio") }),
    transDescription: (p: P) => transField(p, "description"),
    transDescriptionWrapped: (p: P) => ({ translations: transField(p, "description") }),
    transName: (p: P) => transField(p, "name"),
    transNameWrapped: (p: P) => ({ translations: transField(p, "name") }),
    transQuestionText: (p: P) => transField(p, "questionText"),
    transQuestionTextWrapped: (p: P) => ({ translations: transField(p, "questionText") }),
    transSummary: (p: P) => transField(p, "summary"),
    transSummaryWrapped: (p: P) => ({ translations: transField(p, "summary") }),
    transResponseWrapped: (p: P) => ({ translations: transField(p, "response") }),
    transText: (p: P) => transField(p, "text"),
    transTextWrapped: (p: P) => ({ translations: transField(p, "text") }),
    tag: ({ insensitive }: P) => ({ ...insensitive }),
    tagWrapped: (p: P) => ({ tag: SearchStringMap.tag(p) }),
    tags: ({ insensitive }: P) => ({ some: { tag: { tag: { ...insensitive } } } }),
    tagsWrapped: (p: P) => ({ tags: SearchStringMap.tags(p) }),
};
