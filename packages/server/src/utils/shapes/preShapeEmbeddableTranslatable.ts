import { GqlModelType } from "@local/shared";
import { getLogic } from "../../getters/getLogic";
import { WithIdField } from "../../types";

/**
 * Used in mutate.shape.pre of non-versioned objects which have translations
 * that come with search embeddings.
 * @returns object with embeddingNeedsUpdate flag
 */
export const preShapeEmbeddableTranslatable = <IdField extends string = "id">({
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
}): { embeddingNeedsUpdateMap: Record<string, { [language in string]: boolean }> } => {
    // Initialize map
    const embeddingNeedsUpdateMap: Record<string, { [language in string]: boolean }> = {};
    // Get id field
    const { idField } = getLogic(["idField"], objectType, ["en"], "preShapeEmbeddableTranslatable");
    // For Create, every language needs to be updated
    for (const { input } of Create) {
        embeddingNeedsUpdateMap[input[idField]] = input.translationsCreate?.reduce((acc, t) => ({ ...acc, [t.language]: true }), {} as Record<string, boolean>) ?? {};
    }
    // Same for Update
    for (const { input } of Update) {
        embeddingNeedsUpdateMap[input[idField]] = {
            ...input.translationsCreate?.reduce((acc, t) => ({ ...acc, [t.language]: true }), {} as Record<string, boolean>),
            ...input.translationsUpdate?.reduce((acc, t) => ({ ...acc, [t.language]: true }), {} as Record<string, boolean>),
        };
    }
    return { embeddingNeedsUpdateMap };
};
