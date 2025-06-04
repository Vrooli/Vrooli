import { type ModelType } from "@local/shared";
import { shapeHelper, type ShapeHelperOutput, type ShapeHelperProps } from "../../builders/shapeHelper.js";
import { type RelationshipType } from "../../builders/types.js";
import { type EmbeddingLanguageUpdateMap } from "./preShapeEmbeddableTranslatable.js";

type TranslationShapeHelperProps<
    Types extends readonly RelationshipType[],
> = Omit<ShapeHelperProps<false, Types, false>, "isOneToOne" | "joinData" | "objectType" | "parentRelationshipName" | "relation" | "softDelete"> & {
    embeddingNeedsUpdate?: EmbeddingLanguageUpdateMap,
}

/**
* Add, update, or remove translation data for an object.
*/
export async function translationShapeHelper<
    Types extends readonly RelationshipType[],
>({
    data,
    embeddingNeedsUpdate = {}, // Only used for embeddable translations 
    ...rest
}: TranslationShapeHelperProps<Types>):
    Promise<ShapeHelperOutput<false, "id">> {
    return shapeHelper({
        data: {
            translationsCreate: data.translationsCreate?.map(({ language, ...rest }) => ({
                ...rest,
                id: BigInt(rest.id),
                language,
                embeddingExpiredAt: embeddingNeedsUpdate[language] === true ? new Date() : undefined,
            })),
            translationsUpdate: data.translationsUpdate?.map(({ language, ...rest }) => ({
                ...rest,
                id: BigInt(rest.id),
                language,
                embeddingExpiredAt: embeddingNeedsUpdate[language] === true ? new Date() : undefined,
            })),
        },
        isOneToOne: false,
        objectType: "Translation" as ModelType,
        parentRelationshipName: "",
        relation: "translations",
        ...rest,
    });
}

