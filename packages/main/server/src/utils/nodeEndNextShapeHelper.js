import { shapeHelper } from "../builders";
export const nodeEndNextShapeHelper = async ({ data, preMap, prisma, relTypes, userData, }) => {
    return shapeHelper({
        data,
        isOneToOne: false,
        isRequired: false,
        joinData: {
            fieldName: "toRoutineVersion",
            uniqueFieldName: "fromEndId_toRoutineVersionId",
            childIdFieldName: "toRoutineVersionId",
            parentIdFieldName: "fromEndId",
            parentId: data.id ?? null,
        },
        objectType: "RoutineVersion",
        parentRelationshipName: "suggestedNextByNode",
        preMap,
        primaryKey: "id",
        prisma,
        relation: "suggestedNextRoutineVersions",
        relTypes,
        userData,
    });
};
//# sourceMappingURL=nodeEndNextShapeHelper.js.map