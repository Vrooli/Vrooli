import { GqlModelType } from "@local/shared";
import { preShapeEmbeddableTranslatable } from "./preShapeEmbeddableTranslatable";

/**
 * Used in mutate.shape.pre of version objects. Has one purpose:
 * 1. Calculate embeddingNeedsUpdate flag for translations relationship
 * @returns object with embeddingNeedsUpdate flag
 */
export const preShapeVersion = ({
    createList,
    updateList,
    objectType,
}: {
    createList: { [x: string]: any }[],
    updateList: { where: { id: string }, data: { [x: string]: any } }[],
    objectType: GqlModelType | `${GqlModelType}`,
}): {
    embeddingNeedsUpdateMap: Record<string, { [language in string]: boolean }>,
} => {
    const { embeddingNeedsUpdateMap } = preShapeEmbeddableTranslatable({ createList, updateList, objectType });
    return { embeddingNeedsUpdateMap };
};
