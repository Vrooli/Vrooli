import { lowercaseFirstLetter } from "@local/utils";
import { shapeHelper } from "../builders";
export const ownerShapeHelper = async ({ data, isCreate, objectType, parentRelationshipName, relation, relTypes, ...rest }) => {
    const requiresTransfer = rest.preMap[objectType].transferMap[data.id];
    if (requiresTransfer) {
        if (isCreate) {
            return { [`${relation}User`]: { connect: { id: rest.userData.id } } };
        }
        return {};
    }
    return {
        ...(await shapeHelper({ relation: lowercaseFirstLetter(`${relation}Organization`), relTypes, isOneToOne: true, isRequired: false, objectType: "Organization", parentRelationshipName, data, ...rest })),
        ...(await shapeHelper({ relation: lowercaseFirstLetter(`${relation}User`), relTypes, isOneToOne: true, isRequired: false, objectType: "User", parentRelationshipName, data, ...rest })),
    };
};
//# sourceMappingURL=ownerShapeHelper.js.map