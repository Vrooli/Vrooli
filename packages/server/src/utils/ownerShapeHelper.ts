import { GqlModelType, SessionUser } from "@shared/consts";
import { lowercaseFirstLetter, shapeHelper, ShapeHelperOutput } from "../builders";
import { PrismaType } from "../types";

type OwnerShapeHelperProps<
    FieldName extends 'ownedBy',
> = {
    data: any;
    isCreate: boolean;
    objectType: GqlModelType | `${GqlModelType}`;
    parentRelationshipName: string;
    preMap: { [x in `${GqlModelType}`]?: any };
    prisma: PrismaType;
    relation: FieldName;
    relTypes: readonly ('Connect' | 'Disconnect')[];
    userData: SessionUser;
}

/**
* Connect or disconnect owners to/from an object
*/
export const ownerShapeHelper = async <
    Types extends readonly ('Connect' | 'Disconnect')[],
    FieldName extends 'ownedBy',
>({
    data,
    isCreate,
    objectType,
    parentRelationshipName,
    relation,
    relTypes,
    ...rest
}: OwnerShapeHelperProps<FieldName>): Promise<
    (ShapeHelperOutput<true, false, Types[number], `${FieldName}Organization`, 'id'> &
    ShapeHelperOutput<true, false, Types[number], `${FieldName}User`, 'id'>) | {}
> => {
    console.log('in ownershape helper', relation, relTypes, objectType, isCreate)
    // Check preMap to see if we're allowed to set the owner
    const requiresTransfer = rest.preMap[objectType].transferMap[data.id];
    // If a transfer is required
    if (requiresTransfer) {
        console.log('ownershape requires transfer', relation);
        // If create, set owner to current user
        if (isCreate) {
            return { [`${relation}User`]: { connect: { id: rest.userData.id } } };
        }
        // Otherwise, return empty object
        return {};
    }
    return {
        ...(await shapeHelper({ relation: lowercaseFirstLetter(`${relation}Organization`), relTypes, isOneToOne: true, isRequired: false, objectType: 'Organization', parentRelationshipName, data, ...rest })),
        ...(await shapeHelper({ relation: lowercaseFirstLetter(`${relation}User`), relTypes, isOneToOne: true, isRequired: false, objectType: 'User', parentRelationshipName, data, ...rest })),
    }
}