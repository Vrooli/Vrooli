import { GqlModelType } from "@local/shared";
import { WithIdField } from "../types";
import { preShapeEmbeddableTranslatable } from "./preShapeEmbeddableTranslatable";

/**
 * Used in mutate.shape.pre of version objects. Has one purpose:
 * 1. Calculate embeddingNeedsUpdate flag for translations relationship
 * @returns object with embeddingNeedsUpdate flag
 */
export const preShapeVersion = <IdField extends string = "id">({
    Create,
    Update,
    objectType,
}: {
    Create: {
        input: WithIdField<IdField> & {
            translationsCreate?: { language: string }[] | null | undefined,
        }
    }[],
    Update: {
        input: WithIdField<IdField> & {
            translationsCreate?: { language: string }[] | null | undefined,
            translationsUpdate?: { language: string }[] | null | undefined,
            translationsDelete?: string[] | null | undefined,
        }
    }[],
    objectType: GqlModelType | `${GqlModelType}`,
}): {
    embeddingNeedsUpdateMap: Record<string, { [language in string]: boolean }>,
} => {
    const { embeddingNeedsUpdateMap } = preShapeEmbeddableTranslatable<IdField>({ Create, Update, objectType });
    return { embeddingNeedsUpdateMap };
};
