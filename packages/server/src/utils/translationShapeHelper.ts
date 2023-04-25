import { GqlModelType } from "@local/shared";
import { shapeHelper, ShapeHelperInput, ShapeHelperOutput, ShapeHelperProps } from "../builders";
import { RelationshipType } from "../builders/types";

type TranslationShapeHelperProps<
    Input extends ShapeHelperInput<false, IsRequired, Types[number], 'translations'>,
    IsRequired extends boolean,
    Types extends readonly RelationshipType[],
> = Omit<ShapeHelperProps<Input, false, IsRequired, Types, 'translations', 'id', false>, 'isOneToOne' | 'joinData' | 'objectType' | 'parentRelationshipName' | 'primaryKey' | 'relation' | 'softDelete'>;

/**
* Add, update, or remove translation data for an object.
*/
export const translationShapeHelper = async <
    Input extends ShapeHelperInput<false, IsRequired, Types[number], 'translations'>,
    IsRequired extends boolean,
    Types extends readonly RelationshipType[],
>({
    data,
    isRequired,
    preMap,
    prisma,
    relTypes,
    userData,
}: TranslationShapeHelperProps<Input, IsRequired, Types>):
    Promise<ShapeHelperOutput<false, IsRequired, Types[number], 'translations', 'id'>> => {
    return shapeHelper({
        data,
        isOneToOne: false,
        isRequired,
        objectType: 'Translation' as GqlModelType,
        parentRelationshipName: '',
        preMap,
        primaryKey: 'id',
        prisma,
        relation: 'translations',
        relTypes,
        userData,
    })
}