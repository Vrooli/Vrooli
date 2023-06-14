import { GqlModelType } from "@local/shared";
import { ObjectMap } from "../models/base";

/**
 * Used in mutate.shape.pre of non-versioned objects which have translations
 * that come with search embeddings.
 * @returns object with embeddingNeedsUpdate flag
 */
export const preShapeEmbeddableTranslatable = ({
    createList,
    updateList,
    objectType,
}: {
    createList: { [x: string]: any }[],
    updateList: { where: { id: string }, data: { [x: string]: any } }[],
    objectType: GqlModelType | `${GqlModelType}`,
}): { embeddingNeedsUpdateMap: Record<string, { [language in string]: boolean }> } => {
    // Initialize map
    const embeddingNeedsUpdateMap: Record<string, { [language in string]: boolean }> = {};
    // Find id field
    const idField = ObjectMap[objectType]!.idField ?? "id";
    // For createList, every language needs to be updated
    for (const create of createList) {
        embeddingNeedsUpdateMap[create[idField as any]] = create.translationsCreate?.reduce((acc, t) => ({ ...acc, [t.language]: true }), {} as Record<string, boolean>) ?? {};
    }
    // Same for updateList
    for (const update of updateList) {
        embeddingNeedsUpdateMap[update.where[idField as any]] = {
            ...update.data.translationsCreate?.reduce((acc, t) => ({ ...acc, [t.language]: true }), {} as Record<string, boolean>),
            ...update.data.translationsUpdate?.reduce((acc, t) => ({ ...acc, [t.language]: true }), {} as Record<string, boolean>),
        };
    }
    return { embeddingNeedsUpdateMap };
};
