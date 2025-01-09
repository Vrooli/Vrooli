import { ModelType } from "@local/shared";
import { shapeHelper, ShapeHelperOutput, ShapeHelperProps } from "../../builders/shapeHelper";
import { RelationshipType } from "../../builders/types";
import { EmbeddingLanguageUpdateMap } from "./preShapeEmbeddableTranslatable";

type TranslationShapeHelperProps<
    Types extends readonly RelationshipType[],
> = Omit<ShapeHelperProps<false, Types, false>, "isOneToOne" | "joinData" | "objectType" | "parentRelationshipName" | "relation" | "softDelete"> & {
    embeddingNeedsUpdate?: EmbeddingLanguageUpdateMap,
}

/**
* Add, update, or remove translation data for an object.
*/
export const translationShapeHelper = async <
    Types extends readonly RelationshipType[],
>({
    data,
    embeddingNeedsUpdate = {}, // Only used for embeddable translations 
    ...rest
}: TranslationShapeHelperProps<Types>):
    Promise<ShapeHelperOutput<false, "id">> => {
    return shapeHelper({
        data: {
            translationsCreate: data.translationsCreate?.map(({ language, ...rest }) => ({
                ...rest,
                language,
                embeddingNeedsUpdate: embeddingNeedsUpdate[language] ?? undefined,
            })),
            translationsUpdate: data.translationsUpdate?.map(({ language, ...rest }) => ({
                ...rest,
                language,
                embeddingNeedsUpdate: embeddingNeedsUpdate[language] ?? undefined,
            })),
        },
        isOneToOne: false,
        objectType: "Translation" as ModelType,
        parentRelationshipName: "",
        relation: "translations",
        ...rest,
    });
};
