import { type ModelType } from "@local/shared";
import { type WithIdField } from "../../types.js";
import { type EmbeddingLanguageUpdateMap, preShapeEmbeddableTranslatable } from "./preShapeEmbeddableTranslatable.js";

type PreShapeVersionParams<IdField extends string = "id"> = {
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

export type PreShapeVersionResult = {
    embeddingNeedsUpdateMap: Record<string, EmbeddingLanguageUpdateMap>,
};

/**
 * Used in mutate.shape.pre of version objects. Has one purpose:
 * 1. Calculate embeddingNeedsUpdate flag for translations relationship
 * @returns object with embeddingNeedsUpdate flag
 */
export function preShapeVersion<IdField extends string = "id">({
    Create,
    Update,
    objectType,
}: PreShapeVersionParams<IdField>): PreShapeVersionResult {
    const { embeddingNeedsUpdateMap } = preShapeEmbeddableTranslatable<IdField>({ Create, Update, objectType });
    return { embeddingNeedsUpdateMap };
}
