import { GqlModelType } from "@local/shared";
import { shapeHelper, ShapeHelperInput, ShapeHelperOutput, ShapeHelperProps } from "../../builders/shapeHelper";
import { RelationshipType } from "../../builders/types";

type TranslationShapeHelperProps<
    Input extends ShapeHelperInput<false, IsRequired, Types[number], "translations">,
    IsRequired extends boolean,
    Types extends readonly RelationshipType[],
> = Omit<ShapeHelperProps<Input, false, IsRequired, Types, "translations", "id", false>, "isOneToOne" | "joinData" | "objectType" | "parentRelationshipName" | "primaryKey" | "relation" | "softDelete"> & {
    embeddingNeedsUpdate?: Record<string, { [language in string]: boolean }>,
}

/**
* Add, update, or remove translation data for an object.
*/
export const translationShapeHelper = async <
    Input extends ShapeHelperInput<false, IsRequired, Types[number], "translations">,
    IsRequired extends boolean,
    Types extends readonly RelationshipType[],
>({
    data,
    embeddingNeedsUpdate = {}, // Only used for embeddable translations 
    ...rest
}: TranslationShapeHelperProps<Input, IsRequired, Types>):
    Promise<ShapeHelperOutput<false, IsRequired, Types[number], "translations", "id">> => {
    return shapeHelper({
        data: {
            translationsCreate: (data as any).translationsCreate?.map(({ language, ...rest }) => ({
                ...rest,
                language,
                embeddingNeedsUpdate: embeddingNeedsUpdate[language] ?? undefined,
            })),
            translationsUpdate: (data as any).translationsUpdate?.map(({ language, ...rest }) => ({
                ...rest,
                language,
                embeddingNeedsUpdate: embeddingNeedsUpdate[language] ?? undefined,
            })),
        },
        isOneToOne: false,
        objectType: "Translation" as GqlModelType,
        parentRelationshipName: "",
        primaryKey: "id",
        relation: "translations",
        ...rest,
    });
};
