import { GqlModelType, SessionUser } from "@shared/consts";
import { getLogic } from "../getters";
import { PrismaType } from "../types";

/**
 * Used in mutate.shape.pre of root objects. Has two purposes:
 * 1. Calculate hasCompleteVersion flag and completedAt date to update object in database
 * 2. Calculate hasCompleteAndPublicVersion flag to update reputation
 * @returns hasCompleteVersionMap and hasCompleteAndPublicVersionMap
 */
export const preHasPublics = async ({ 
    createList, 
    updateList, 
    objectType,
    prisma, 
    userData,
}: {
    createList: { [x: string]: any }[],
    updateList: { where: { id: string }, data: { [x: string]: any } }[],
    objectType: GqlModelType | `${GqlModelType}`,
    prisma: PrismaType,
    userData: SessionUser,
}): Promise<{
    hasCompleteVersionMap: Record<string, { hasCompleteVersion: boolean, completedAt: Date | null }>,
    hasCompleteAndPublicVersionMap: Record<string, boolean>,
}> => {
    // Get prisma delegate
    const { delegate } = getLogic(['delegate'], objectType, userData.languages, 'preHasPublics');
    // Calculate hasCompleteVersion and hasCompleteAndPublic version flags
    const hasCompleteVersionMap: Record<string, { hasCompleteVersion: boolean, completedAt: Date | null }> = {};
    const hasCompleteAndPublicVersionMap: Record<string, boolean> = {};
    // For createList (very simple)
    for (const create of createList) {
        const hasCompleteVersion = create.versionsCreate?.some(v => v.isComplete) ?? false;
        hasCompleteVersionMap[create.id] = {
            hasCompleteVersion,
            completedAt: hasCompleteVersion ? new Date() : null,
        }
        hasCompleteAndPublicVersionMap[create.id] = !create.isPrivate && (create.versionsCreate?.some(v => v.isComplete && !v.isPrivate) ?? false);
    }
    // For updateList (much more complicated)
    if (updateList.length > 0) {
        // Find original flags
        const originalRoots = await delegate(prisma).findMany({
            where: { id: { in: updateList.map(u => u.data.id) } },
            select: { 
                id: true,
                isPrivate: true,
                versions: {
                    select: {
                        id: true,
                        isComplete: true,
                        isPrivate: true,
                    }
                }
            }
        });
        // Loop through updates
        for (const update of updateList) {
            // Find original
            const original = originalRoots.find(r => r.id === update.data.id);
            if (!original) throw new Error(`Could not find original routine with id ${update.data.id}`);
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
            hasCompleteVersionMap[update.data.id] = {
                hasCompleteVersion,
                completedAt: hasCompleteVersion ? new Date() : null,
            }
            hasCompleteAndPublicVersionMap[update.data.id] = !isRootPrivate && versions.some((v: any) => v.isComplete && !v.isPrivate);
        }
    }
    return { hasCompleteVersionMap, hasCompleteAndPublicVersionMap };
}