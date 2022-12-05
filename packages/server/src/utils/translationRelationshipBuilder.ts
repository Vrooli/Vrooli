import { relationshipBuilderHelper } from "../builders";
import { BuiltRelationship } from "../builders/types";
import { SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";

/**
* Add, update, or remove translation data for an object.
*/
export const translationRelationshipBuilder = async <IsAdd extends boolean>(
    prisma: PrismaType,
    userData: SessionUser,
    data: { [x: string]: any },
    isAdd = true as IsAdd,
): Promise<BuiltRelationship<any, IsAdd, false, any, any>> => {
    return await relationshipBuilderHelper<any, IsAdd, false, any, any, any, any>({
        data,
        isAdd,
        isOneToOne: false,
        isTransferable: false,
        prisma,
        relationshipName: 'translations',
        userData,
    });
}