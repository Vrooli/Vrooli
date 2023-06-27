import { GqlModelType } from "@local/shared";
import { getLogic } from "../getters";

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
    updateList: { [x: string]: any }[],
    objectType: GqlModelType | `${GqlModelType}`,
}): { embeddingNeedsUpdateMap: Record<string, { [language in string]: boolean }> } => {
    // Initialize map
    const embeddingNeedsUpdateMap: Record<string, { [language in string]: boolean }> = {};
    // Get id field
    const { idField } = getLogic(["idField"], objectType, ["en"], "preShapeEmbeddableTranslatable");
    // For createList, every language needs to be updated
    for (const create of createList) {
        embeddingNeedsUpdateMap[create[idField as any]] = create.translationsCreate?.reduce((acc, t) => ({ ...acc, [t.language]: true }), {} as Record<string, boolean>) ?? {};
    }
    // Same for updateList
    for (const update of updateList) {
        embeddingNeedsUpdateMap[update[idField as any]] = {
            ...update.translationsCreate?.reduce((acc, t) => ({ ...acc, [t.language]: true }), {} as Record<string, boolean>),
            ...update.translationsUpdate?.reduce((acc, t) => ({ ...acc, [t.language]: true }), {} as Record<string, boolean>),
        };
    }
    return { embeddingNeedsUpdateMap };
};
