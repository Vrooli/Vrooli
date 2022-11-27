import { PrismaDelegate } from "../builders/types";
import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { GraphQLModelType } from "../models/types";
import { PrismaType } from "../types";

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