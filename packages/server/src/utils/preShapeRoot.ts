import { GqlModelType } from "@shared/consts";
import { exists } from "@shared/utils";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { transfer } from "../models";
import { PrismaType, SessionUserToken } from "../types";

type HasCompleteVersionData = {
    hasCompleteVersion: boolean,
    completedAt: Date | null,
}

type ObjectTriggerData = {
    hasCompleteAndPublic: boolean,
    wasCompleteAndPublic: boolean,
    hasBeenTransferred: boolean,
    hasParent: boolean,
    owner: {
        id: string,
        __typename: 'User' | 'Organization',
    }
}

const originalDataSelect = {
    id: true,
    hasBeenTransferred: true,
    isPrivate: true,
    ownedByOrganization: { select: { id: true } },
    ownedByUser: { select: { id: true } },
    parent: { select: { id: true } },
    versions: {
        select: {
            id: true,
            isComplete: true,
            isPrivate: true,
        }
    }
}

/**
 * Used in mutate.shape.pre of root objects. Has three purposes:
 * 1. Calculate hasCompleteVersion flag and completedAt date to update object in database)
 * 2. Calculate data for objectCreated/Updated/Deleted trigger
 * 3. Determine which creates/updates require a transfer request
 * @returns versionMap and triggerMap
 */
export const preShapeRoot = async ({
    createList,
    updateList,
    deleteList,
    objectType,
    prisma,
    userData,
}: {
    createList: { [x: string]: any }[],
    updateList: { where: { id: string }, data: { [x: string]: any } }[],
    deleteList: string[],
    objectType: GqlModelType | `${GqlModelType}`,
    prisma: PrismaType,
    userData: SessionUserToken,
}): Promise<{
    versionMap: Record<string, HasCompleteVersionData>,
    triggerMap: Record<string, { hasCompleteAndPublic: boolean, wasCompleteAndPublic: boolean }>,
    transferMap: Record<string, boolean>,
}> => {
    // Get prisma delegate
    const { delegate } = getLogic(['delegate'], objectType, userData.languages, 'preHasPublics');
    // Calculate hasCompleteVersion and hasCompleteAndPublic version flags
    const versionMap: Record<string, HasCompleteVersionData> = {};
    const triggerMap: Record<string, ObjectTriggerData> = {};
    const transferMap: Record<string, boolean> = {};
    // For createList (very simple)
    for (const create of createList) {
        const hasCompleteVersion = create.versionsCreate?.some(v => v.isComplete) ?? false;
        versionMap[create.id] = {
            hasCompleteVersion,
            completedAt: hasCompleteVersion ? new Date() : null,
        }
        triggerMap[create.id] = {
            wasCompleteAndPublic: true, // Doesn't matter
            hasCompleteAndPublic: !create.isPrivate && (create.versionsCreate?.some(v => v.isComplete && !v.isPrivate) ?? false),
            hasBeenTransferred: false, // Doesn't matter
            hasParent: typeof create.parentConnect === 'string',
            owner: {
                id: create.ownedByUser ?? create.ownedByOrganization,
                __typename: create.ownedByUser ? 'User' : 'Organization',
            },
        }
    }
    // For updateList (much more complicated)
    if (updateList.length > 0) {
        // Find original data
        const originalData = await delegate(prisma).findMany({
            where: { id: { in: updateList.map(u => u.data.id) } },
            select: originalDataSelect,
        });
        // Loop through updates
        for (const update of updateList) {
            // Find original
            const original = originalData.find(r => r.id === update.data.id);
            if (!original) throw new CustomError('0412', 'InternalError', userData.languages, { id: update?.data?.id });
            const isRootPrivate = update.data.isPrivate ?? original.isPrivate;
            // Convert original verions to map for easy lookup
            let updatedWithOriginal = original.versions.reduce((acc, v) => ({ ...acc, [v.id]: v }), {} as Record<string, any>);
            // Combine updated versions with original versions
            if (update.data.versionsUpdate) {
                for (const v of update.data.versionsUpdate) {
                    updatedWithOriginal[v.id] = {
                        ...updatedWithOriginal[v.id],
                        ...v,
                    }
                }
            }
            // Combine new, updated, and original versions. Then remove deleting versions
            const allVersions = Object.values(updatedWithOriginal).concat(update.data.versionsCreate ?? []);
            const versions = allVersions.filter(v => !update.data.versionsDelete?.includes((v as any).id));
            // Calculate flags
            const hasCompleteVersion = versions.some(v => (v as any).isComplete);
            versionMap[update.data.id] = {
                hasCompleteVersion,
                completedAt: hasCompleteVersion ? new Date() : null,
            }
            triggerMap[update.data.id] = {
                wasCompleteAndPublic: !original.isPrivate && original.versions.some((v: any) => v.isComplete && !v.isPrivate),
                hasCompleteAndPublic: !isRootPrivate && versions.some((v: any) => v.isComplete && !v.isPrivate),
                hasBeenTransferred: original.hasBeenTransferred,
                hasParent: exists(original.parent),
                // TODO owner might be changed here depending on how triggers are implemented.
                // For now, using original owner
                owner: {
                    id: original.ownedByUser?.id ?? original.ownedByOrganization?.id,
                    __typename: original.ownedByUser ? 'User' : 'Organization',
                }
            }
        }
    }
    // For deleteList (fairly simple)
    if (deleteList.length > 0) {
        // Find original data
        const originalData = await delegate(prisma).findMany({
            where: { id: { in: deleteList } },
            select: originalDataSelect,
        });
        // Loop through deletes
        for (const id of deleteList) {
            // Find original
            const original = originalData.find(r => r.id === id);
            if (!original) throw new CustomError('0413', 'InternalError', userData.languages, { id });
            triggerMap[id] = {
                wasCompleteAndPublic: !original.isPrivate && original.versions.some((v: any) => v.isComplete && !v.isPrivate),
                hasCompleteAndPublic: true, // Doesn't matter
                hasBeenTransferred: original.hasBeenTransferred,
                hasParent: exists(original.parent),
                // TODO owner might be changed here depending on how triggers are implemented.
                // For now, using original owner
                owner: {
                    id: original.ownedByUser?.id ?? original.ownedByOrganization?.id,
                    __typename: original.ownedByUser ? 'User' : 'Organization',
                }
            }
        }
    }
    // Finally determine which creates/updates require a transfer request
    // Get create and update owners from triggerMap and map them to object IDs. Make sure to filter out deleteList owners
    const ownersEntries = Object.entries(triggerMap)
        .filter(([id]) => !deleteList.includes(id))
        .map(([id, data]) => [id, data.owner]);
    const ownersMap = Object.fromEntries(ownersEntries);
    const requireTransfers = await transfer(prisma).checkTransferRequests(Object.values(ownersMap), userData);
    for (let i = 0; i < requireTransfers.length; i++) {
        const id = Object.keys(ownersMap)[i];
        transferMap[id] = requireTransfers[i];
    }
    return { versionMap, triggerMap, transferMap };
}