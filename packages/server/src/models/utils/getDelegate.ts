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
    languages: string[],
    errorTrace: string,
): PrismaDelegate {
    const prismaDelegate: PrismaDelegate | undefined = ObjectMap[objectType]?.delegate!(prisma);
    if (!prismaDelegate) {
        throw new CustomError('0281', 'InvalidArgs', languages, { errorTrace, objectType });
    }
    return prismaDelegate;
}