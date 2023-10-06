import { lowercaseFirstLetter } from "@local/shared";
import { shapeHelper, ShapeHelperOutput, ShapeHelperProps } from "../../builders/shapeHelper";

type OwnerShapeHelperProps<
    FieldName extends "ownedBy",
    Types extends readonly ("Connect" | "Disconnect")[],
> =
    Omit<ShapeHelperProps<{ id: string }, false, false, Types, FieldName, "id", false>, "isOneToOne" | "isRequired" | "joinData" | "primaryKey" | "softDelete"> & {
        isCreate: boolean;
    }

/**
* Connect or disconnect owners to/from an object
*/
export const ownerShapeHelper = async <
    Types extends readonly ("Connect" | "Disconnect")[],
    FieldName extends "ownedBy",
>({
    data,
    isCreate,
    objectType,
    parentRelationshipName,
    relation,
    relTypes,
    ...rest
}: OwnerShapeHelperProps<FieldName, Types>): Promise<
    (ShapeHelperOutput<true, false, Types[number], `${FieldName}Organization`, "id"> &
        ShapeHelperOutput<true, false, Types[number], `${FieldName}User`, "id">) | object
> => {
    // Check preMap to see if we're allowed to set the owner
    const requiresTransfer = rest.preMap[objectType].transferMap[data.id];
    // If a transfer is required
    if (requiresTransfer) {
        // If create, set owner to current user
        if (isCreate) {
            return { [`${relation}User`]: { connect: { id: rest.userData.id } } };
        }
        // Otherwise, return empty object
        return {};
    }
    return {
        ...(await shapeHelper({ ...rest, relation: lowercaseFirstLetter(`${relation}Organization`), relTypes, isOneToOne: true, isRequired: false, objectType: "Organization", parentRelationshipName, data })),
        ...(await shapeHelper({ ...rest, relation: lowercaseFirstLetter(`${relation}User`), relTypes, isOneToOne: true, isRequired: false, objectType: "User", parentRelationshipName, data })),
    };
};
