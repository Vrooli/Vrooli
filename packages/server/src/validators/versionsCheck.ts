import { GqlModelType } from "@shared/consts";
import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { PrismaType } from "../types";

type VersionsCheckProps = {
    prisma: PrismaType,
    objectType: `${GqlModelType.Api | GqlModelType.Note | GqlModelType.Project | GqlModelType.Routine | GqlModelType.SmartContract | GqlModelType.Standard}`,
    createMany: { 
        id: string, 
        rootConnect?: string | null | undefined,
        rootCreate?: { id: string } | null | undefined,
        versionLabel?: string | null | undefined 
    }[],
    updateMany: { 
        where: { id: string },
        data: { versionLabel?: string | null | undefined },
    }[],
    deleteMany: string[],
    languages: string[],
}

/**
 * Checks if versions of an object type can be created, updated. 
 * Throws error on failure.
 * Requirements:
 * 1. The root object and updating versions are not soft-deleted
 * 2. Version labels are unique per root object, including existing versions in the database
 * 3. Updating versions are not marked as complete, OR the version or root is private (i.e. isPrivate = true). 
 * This helps ensure that public data is immutable, while owners have full control over private data
 */
export const versionsCheck = async ({
    prisma,
    objectType,
    createMany,
    updateMany,
    deleteMany,
    languages,
}: VersionsCheckProps) => {
    // Filter unchanged versions from create and update data
    const create = createMany.filter(x => x.versionLabel).map(x => ({
        id: x.id,
        rootId: x.rootConnect || x.rootCreate?.id as string,
        versionLabel: x.versionLabel,
    }));
    const update = updateMany.filter(x => x.data.versionLabel).map(x => ({
        id: x.where.id,
        versionLabel: x.data.versionLabel,
    }));
    // Find unique root ids from create data
    const createRootIds = create.map(x => x.rootId);
    const uniqueRootIds = [...new Set(createRootIds)];
    // Find unique version ids from update and delete data
    const updateIds = update.map(x => x.id);
    const deleteIds = deleteMany;
    const uniqueVersionIds = [...new Set([...updateIds, ...deleteIds])];
    // Query the database for existing data (by root)
    const existingRoots = await ObjectMap[objectType]!.delegate(prisma).findMany({
        where: {
            OR: [
                { id: { in: uniqueRootIds } },
                { versions: { some: { id: { in: uniqueVersionIds } } } },
            ]
        },
        select: {
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            // Also query isInternal for routine and standard versions
            ...([GqlModelType.RoutineVersion, GqlModelType.StandardVersion].includes(objectType as any) && { isInternal: true }),
            isPrivate: true,
            versions: {
                select: {
                    id: true,
                    isComplete: true,
                    isDeleted: true,
                    isPrivate: true,
                    versionLabel: true,
                }
            }
        }
    });
    for (const root of existingRoots) {
        // Check 1
        // Root cannot already be deleted
        if (root.isDeleted) {
            throw new CustomError('0377', 'ErrorUnknown', languages);
        }
        // Updating versions cannot be deleted
        for (const version of root.versions) {
            if (updateIds.includes(version.id) && version.isDeleted) {
                throw new CustomError('0378', 'ErrorUnknown', languages);
            }
        }
        // Check 2
        // Group ids and labels of existing versions, which are not being deleted
        const versionIds = root.versions.filter(x => !deleteIds.includes(x.id)).map(x => x.id);
        const versionLabels = root.versions.filter(x => !deleteIds.includes(x.id)).map(x => x.versionLabel);
        // New versions cannot have the same label as existing versions
        const createLabels = create.filter(x => x.rootId === root.id).map(x => x.versionLabel);
        if (createLabels.some(x => versionLabels.includes(x))) {
            throw new CustomError('0379', 'ErrorUnknown', languages);
        }
        // Updating versions cannot have the same label as existing versions
        const updateLabels = update.filter(x => versionIds.includes(x.id)).map(x => x.versionLabel);
        // We must filter out updating labels from the existing labels, to support swapping
        const verionLabelsWithoutUpdate = root.versions.filter(x => !deleteIds.includes(x.id) && !updateIds.includes(x.id)).map(x => x.versionLabel);
        if (updateLabels.some(x => verionLabelsWithoutUpdate.includes(x))) {
            throw new CustomError('0380', 'ErrorUnknown', languages);
        }
        // Check 3
        // If the root is not private and not internal (if applicable)
        if (!root.isPrivate && !([GqlModelType.RoutineVersion, GqlModelType.StandardVersion].includes(objectType as any) && root.isInternal)) {
            // Updating versions (which are not private) cannot be marked as complete
            for (const version of root.versions) {
                if (updateIds.includes(version.id) && !version.isPrivate && version.isComplete) {
                    throw new CustomError('0381', 'ErrorUnknown', languages);
                }
            }
        }
    }

}

//TODO move to utils, and also call this in versioned models
/**
 * Reorders version indexes for a root object, when one or more 
 * versions are created, updated, or deleted. Also updates the 
 * isLatest flag for the latest version, and the hasCompleteVersion 
 * and completedAt fields for the root object.
 * @param prisma Prisma client to query existing data
 */
export const reorderVersions = () => {

}