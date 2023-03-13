import { GqlModelType, SessionUser } from "@shared/consts";
import { shapeHelper, ShapeHelperOutput } from "../builders";
import { PrismaType } from "../types";

type OwnerShapeHelperProps<
    FieldName extends 'ownedBy',
> = {
    data: any;
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
    objectType,
    parentRelationshipName,
    relation,
    relTypes,
    ...rest
}: OwnerShapeHelperProps<FieldName>): Promise<
    (ShapeHelperOutput<true, false, Types[number], `${FieldName}Organization`, 'id'> &
    ShapeHelperOutput<true, false, Types[number], `${FieldName}User`, 'id'>) | {}
> => {
    // Check preMap to see if we're allowed to set the owner
    const requiresTransfer = rest.preMap[objectType].transferMap[data.id];
    // If not allowed to set owner, return empty object. 
    // This object's trigger functions must initiate a transfer request
    if (requiresTransfer) return {};
    return {
        ...(await shapeHelper({ relation: `${relation}Organization`, relTypes, isOneToOne: true, isRequired: false, objectType: 'Organization', parentRelationshipName, data, ...rest })),
        ...(await shapeHelper({ relation: `${relation}User`, relTypes, isOneToOne: true, isRequired: false, objectType: 'User', parentRelationshipName, data, ...rest })),
    }
}