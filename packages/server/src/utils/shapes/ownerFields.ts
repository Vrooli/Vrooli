// AI_CHECK: TYPE_SAFETY=server-type-safety-fixes | LAST: 2025-07-01 - Fixed any type to proper Record<string, unknown>
import { lowercaseFirstLetter } from "@vrooli/shared";
import { shapeHelper, type ShapeHelperProps } from "../../builders/shapeHelper.js";

type OwnerShapeHelperProps<
    Types extends readonly ("Connect" | "Disconnect")[],
> = {
    isCreate: boolean;
    relation?: string;
} & Omit<ShapeHelperProps<false, Types, false>, "isOneToOne" | "joinData" | "relation" | "softDelete">

type OwnerShapeReturn = Record<string, unknown>;

/**
* Connect or disconnect owners to/from an object.
* NOTE: The result should be spread into the create/update object. 
* That's why it has a different naming convention than other shape helpers.
*/
export async function ownerFields<
    Types extends readonly ("Connect" | "Disconnect")[],
>({
    data,
    isCreate,
    objectType,
    parentRelationshipName,
    relation = "ownedBy",
    relTypes,
    ...rest
}: OwnerShapeHelperProps<Types>): Promise<OwnerShapeReturn> {
    // Check preMap to see if we're allowed to set the owner
    const requiresTransfer = rest.preMap[objectType].transferMap[data.id];
    // If a transfer is required
    if (requiresTransfer) {
        // If create, set owner to current user
        if (isCreate) {
            return { [`${relation}User`]: { connect: { id: BigInt(rest.userData.id) } } };
        }
        // Otherwise, return empty object
        return {};
    }
    const orgRelName = lowercaseFirstLetter(`${relation}Team`);
    const userRelName = lowercaseFirstLetter(`${relation}User`);
    return {
        [orgRelName]: await shapeHelper({ ...rest, relation: orgRelName, relTypes, isOneToOne: true, objectType: "Team", parentRelationshipName, data }),
        [userRelName]: await shapeHelper({ ...rest, relation: userRelName, relTypes, isOneToOne: true, objectType: "User", parentRelationshipName, data }),
    };
}
