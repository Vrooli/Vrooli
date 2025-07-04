// AI_CHECK: TYPE_SAFETY=server-type-safety-maintenance-phase2 | LAST: 2025-07-04 - Added return type annotations to findLatestPublicVersionIndex, getChangedVersions, and prepareVersionUpdates functions
import { type ModelType, calculateVersionsFromString } from "@vrooli/shared";
import { type PrismaDelegate } from "../../builders/types.js";
import { DbProvider } from "../../db/provider.js";
import { ModelMap } from "../../models/base/index.js";

type Version = {
    id: bigint;
    isLatest: boolean;
    isLatestPublic: boolean,
    isPrivate: boolean;
    versionIndex: number;
    versionLabel: string;
}

/**
 * Sorts versions from lowest to highest
 */
export function sortVersions<T extends { versionLabel: string }>(versions: T[]): T[] {
    if (!Array.isArray(versions)) return [];
    return versions.sort((a, b) => {
        const { major: majorA, moderate: moderateA, minor: minorA } = calculateVersionsFromString(a.versionLabel);
        const { major: majorB, moderate: moderateB, minor: minorB } = calculateVersionsFromString(b.versionLabel);
        if (majorA > majorB) return 1;
        if (majorA < majorB) return -1;
        if (moderateA > moderateB) return 1;
        if (moderateA < moderateB) return -1;
        if (minorA > minorB) return 1;
        if (minorA < minorB) return -1;
        return 0;
    });
}

/**
 * Finds the index of the latest public version in a sorted list of versions.
 * Assumes the versions are sorted in ascending order by their labels.
 *
 * @param {Array} versions - The sorted array of version objects.
 * @return {number} - The index of the latest public version, or -1 if no public version exists.
 */
export function findLatestPublicVersionIndex(versions: Pick<Version, "isPrivate">[]): number {
    for (let i = versions.length - 1; i >= 0; i--) {
        if (!versions[i].isPrivate) {
            return i; // Return the index as soon as the first public version is found from the end
        }
    }
    return -1; // Return -1 if no public version is found
}

/**
 * Identifies which versions have changed between the original and updated lists.
 * @param originalVersions - The original list of versions.
 * @param updatedVersions - The updated list of versions.
 * @returns An array of versions that have changed.
 */
export function getChangedVersions(originalVersions: Version[], updatedVersions: Version[]): Version[] {
    const changedVersions: Version[] = [];

    // Create a map of original versions for quick lookup
    const originalMap = new Map(originalVersions.map(v => [v.id, v]));

    // Check for changes in updated versions compared to original
    for (const updatedVersion of updatedVersions) {
        const originalVersion = originalMap.get(updatedVersion.id);
        if (!originalVersion || // Check if the version is new
            originalVersion.isLatest !== updatedVersion.isLatest ||
            originalVersion.isLatestPublic !== updatedVersion.isLatestPublic ||
            originalVersion.versionIndex !== updatedVersion.versionIndex) {
            changedVersions.push(updatedVersion);
        }
    }

    // Optionally, check for deleted versions
    const updatedMap = new Map(updatedVersions.map(v => [v.id, v]));
    originalVersions.forEach(originalVersion => {
        if (!updatedMap.has(originalVersion.id)) {
            changedVersions.push(originalVersion); // Add deleted versions as changed
        }
    });

    return changedVersions;
}

/**
 * Processes versions for a single root object
 * @param root The root object containing versions to be updated.
 * @returns Data to be updated in a Prisma transaction.
 */
export function prepareVersionUpdates(root: { id: string, versions: Version[] }): Array<{ where: { id: bigint }, data: { isLatest: boolean, isLatestPublic: boolean, versionIndex: number } }> {
    // Sort versions by versionLabel (using copy to avoid mutation of original array)
    const versionsUpdated = sortVersions(root.versions.map(v => ({ ...v }))) as Version[];
    // Set version index for each version and reset flags
    for (let i = 0; i < versionsUpdated.length; i++) {
        versionsUpdated[i].versionIndex = i;
        versionsUpdated[i].isLatest = false;
        versionsUpdated[i].isLatestPublic = false;
    }
    // Set `isLatest` on latest version
    if (versionsUpdated.length) {
        versionsUpdated[versionsUpdated.length - 1].isLatest = true;
    }
    // Set `isLatestPublicVersion` on latest public version
    const lastPublicIndex = findLatestPublicVersionIndex(versionsUpdated);
    if (lastPublicIndex >= 0) {
        versionsUpdated[lastPublicIndex].isLatestPublic = true;
    }
    // If any versions were updated, update root object
    const changedVersions = getChangedVersions(root.versions, versionsUpdated);
    return changedVersions.map(({ id, isLatest, isLatestPublic, versionIndex }) => ({
        where: { id },
        data: { isLatest, isLatestPublic, versionIndex },
    }));
}

/**
 * Used in mutate.shape.post of version objects. Updates  
 * versionIndex, isLatest, and isLatestPublic flags. Cannot be done in pre 
 * because we might need to update additional versions not specified in the mutation
 */
export async function afterMutationsVersion({ createdIds, deletedIds, objectType, updatedIds }: {
    createdIds: string[],
    deletedIds: string[],
    objectType: ModelType | `${ModelType}`,
    updatedIds: string[]
}) {
    // Get db table for root object
    const { dbTable: dbTableRoot } = ModelMap.getLogic(["dbTable"], objectType.replace("Version", "") as ModelType);
    // Get ids from created, updated, and deletedIds
    const versionIds = [...createdIds, ...updatedIds, ...deletedIds];
    // Use version ids to query root objects
    const rootData = await (DbProvider.get()[dbTableRoot] as PrismaDelegate).findMany({
        where: { versions: { some: { id: { in: versionIds.map(id => BigInt(id)) } } } },
        select: {
            id: true,
            versions: {
                select: {
                    id: true,
                    isLatest: true,
                    isLatestPublic: true,
                    isPrivate: true,
                    versionIndex: true,
                    versionLabel: true,
                },
            },
        },
    }) as { id: string, versions: Version[] }[];
    // Find updated versions for each root object and compile into a list
    const updatedVersions = rootData.map(prepareVersionUpdates).flat();
    // Get db table for version object
    const { dbTable: dbTableVersion } = ModelMap.getLogic(["dbTable"], objectType.replace("Version", "") + "Version" as ModelType);
    // Update versions in a Prisma transaction
    const promises = updatedVersions.map(({ where, data }) => DbProvider.get()[dbTableVersion].update({ where, data }));
    await DbProvider.get().$transaction(promises);
}
