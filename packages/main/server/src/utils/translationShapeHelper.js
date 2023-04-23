import { shapeHelper } from "../builders";
export const translationShapeHelper = async ({ data, isRequired, preMap, prisma, relTypes, userData, }) => {
    return shapeHelper({
        data,
        isOneToOne: false,
        isRequired,
        objectType: "Translation",
        parentRelationshipName: "",
        preMap,
        primaryKey: "id",
        prisma,
        relation: "translations",
        relTypes,
        userData,
    });
};
//# sourceMappingURL=translationShapeHelper.js.map