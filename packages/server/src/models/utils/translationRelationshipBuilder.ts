import { SessionUser } from "../../schema/types";
import { PrismaType } from "../../types";
import { BuiltRelationship, relationshipBuilderHelper } from "../builder";

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