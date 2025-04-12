import { ModelType } from "@local/shared";
import { shapeHelper, ShapeHelperOutput, ShapeHelperProps } from "../../builders/shapeHelper.js";
import { RelationshipType } from "../../builders/types.js";

type NodeEndNextShapeHelperProps<
    Types extends readonly RelationshipType[],
> = Omit<ShapeHelperProps<false, Types, false>, "isOneToOne" | "joinData" | "objectType" | "parentRelationshipName" | "relation" | "softDelete">;

/**
* Connects and disconnects suggested next routine versions from end nodes, 
* which is actually creating and deleting node_end_next (i.e. join table) objects.
*/
export async function nodeEndNextShapeHelper<
    Types extends readonly RelationshipType[],
>({
    data,
    ...rest
}: NodeEndNextShapeHelperProps<Types>):
    Promise<ShapeHelperOutput<false, "id">> {
    return shapeHelper({
        data,
        isOneToOne: false,
        joinData: {
            fieldName: "toRoutineVersion",
            uniqueFieldName: "fromEndId_toRoutineVersionId",
            childIdFieldName: "toRoutineVersionId",
            parentIdFieldName: "fromEndId",
            parentId: (data as { id?: string | null | undefined }).id ?? null,
        },
        objectType: "RoutineVersion" as ModelType,
        parentRelationshipName: "suggestedNextByNode",
        relation: "suggestedNextRoutineVersions",
        ...rest,
    });
}
