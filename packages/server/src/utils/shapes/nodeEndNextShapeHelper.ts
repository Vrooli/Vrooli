import { GqlModelType } from "@local/shared";
import { shapeHelper, ShapeHelperInput, ShapeHelperOutput, ShapeHelperProps } from "../../builders/shapeHelper";
import { RelationshipType } from "../../builders/types";

type NodeEndNextShapeHelperProps<
    Input extends ShapeHelperInput<false, false, Types[number], "suggestedNextRoutineVersions">,
    Types extends readonly RelationshipType[],
> = Omit<ShapeHelperProps<Input, false, false, Types, "suggestedNextRoutineVersions", "id", false>, "isRequired" | "isOneToOne" | "joinData" | "objectType" | "parentRelationshipName" | "primaryKey" | "relation" | "softDelete">;

/**
* Connects and disconnects suggested next routine versions from end nodes, 
* which is actually creating and deleting node_end_next (i.e. join table) objects.
*/
export const nodeEndNextShapeHelper = async <
    Types extends readonly RelationshipType[],
    Input extends ShapeHelperInput<false, false, Types[number], "suggestedNextRoutineVersions">,
>({
    data,
    ...rest
}: NodeEndNextShapeHelperProps<Input, Types>):
    Promise<ShapeHelperOutput<false, false, Types[number], "suggestedNextRoutineVersions", "id">> => {
    return shapeHelper({
        data,
        isOneToOne: false,
        isRequired: false,
        joinData: {
            fieldName: "toRoutineVersion",
            uniqueFieldName: "fromEndId_toRoutineVersionId",
            childIdFieldName: "toRoutineVersionId",
            parentIdFieldName: "fromEndId",
            parentId: (data as { id?: string | null | undefined }).id ?? null,
        },
        objectType: "RoutineVersion" as GqlModelType,
        parentRelationshipName: "suggestedNextByNode",
        primaryKey: "id",
        relation: "suggestedNextRoutineVersions",
        ...rest,
    });
};
