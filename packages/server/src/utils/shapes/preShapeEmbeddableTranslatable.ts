import { type ModelType } from "@local/shared";
import { ModelMap } from "../../models/base/index.js";
import { type WithIdField } from "../../types.js";

type PreShapeEmbeddableTranslatableParams<IdField extends string = "id"> = {
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
    objectType: ModelType | `${ModelType}`,
};

/** Map of language codes to boolean indicating if embedding needs update */
export type EmbeddingLanguageUpdateMap = Record<string, boolean>;
export type PreShapeEmbeddableTranslatableResult = {
    embeddingNeedsUpdateMap: Record<string, { [language in string]: boolean }>,
};

/**
 * Used in mutate.shape.pre of non-versioned objects which have translations
 * that come with search embeddings.
 * @returns object with embeddingNeedsUpdate flag
 */
export function preShapeEmbeddableTranslatable<IdField extends string = "id">({
    Create,
    Update,
    objectType,
}: PreShapeEmbeddableTranslatableParams<IdField>): PreShapeEmbeddableTranslatableResult {
    // Initialize map
    const embeddingNeedsUpdateMap: Record<string, { [language in string]: boolean }> = {};
    // Get id field
    const { idField } = ModelMap.getLogic(["idField"], objectType);
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
}
