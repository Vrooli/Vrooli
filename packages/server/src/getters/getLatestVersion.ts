import { GraphQLModelType } from "../models/types";
import { PrismaType } from "../types";

/**
 * Finds the latest version of a versioned object. This includes apis, notes, routines, smart contracts, and standards
 * @returns The id of the latest version
 */
export async function getLatestVersion({
    includeIncomplete = true,
    objectType,
    prisma,
    versionGroupId,
}: {
    includeIncomplete?: boolean,
    objectType: string, // GraphQLModelType,
    prisma: PrismaType,
    versionGroupId: string,
}): Promise<string | undefined> {
    // Handle apis and notes, which don't have an "isComplete" field
    if (['Api', 'Note'].includes(objectType)) {
        const query = {
            where: {
                rootId: versionGroupId,
            },
            orderBy: { versionIndex: 'desc' as const },
            select: { id: true },
        }
        const latestVersion = objectType === 'Api' ? await prisma.api_version.findFirst(query) : await prisma.note_version.findFirst(query);
        return latestVersion?.id;
    }
    // Handle other objects, which have an "isComplete" field
    else {
        const query = {
            where: {
                rootId: versionGroupId,
                isComplete: includeIncomplete ? undefined : true,
            },
            orderBy: { versionIndex: 'desc' as const },
            select: { id: true },
        }
        const latestVersion = objectType === 'Routine' ?
            await prisma.routine_version.findFirst(query) : objectType === 'SmartContract' ?
                await prisma.smart_contract_version.findFirst(query) :
                await prisma.standard_version.findFirst(query);
        return latestVersion?.id;
    }
}