import { CustomError } from "../../events";
import { PrismaType } from "../../types";
import { ObjectMap } from "../builder";
import { GraphQLModelType, PrismaDelegate } from "../types";

/**
 * Finds all permissions for the given object ids
 */
export function getDelegate(
    objectType: GraphQLModelType,
    prisma: PrismaType,
    errorTrace: string,
): PrismaDelegate {
    // Find validator and prisma delegate for this object type
    const prismaDelegate: PrismaDelegate | undefined = ObjectMap[objectType]?.prismaObject!(prisma);
    if (!prismaDelegate) {
        throw new CustomError('InvalidArgs', `Invalid object type in ${errorTrace}: ${objectType}`, { trace: '0281' });
    }
    return prismaDelegate;
}