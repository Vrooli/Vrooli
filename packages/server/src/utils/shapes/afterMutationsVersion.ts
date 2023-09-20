import { calculateVersionsFromString, GqlModelType } from "@local/shared";
import { getLogic } from "../../getters";
import { PrismaType, SessionUserToken } from "../../types";

/**
 * Used in mutate.shape.post of version objects. Updates  
 * versionIndex and isLatest flags. Cannot be done in pre 
 * because we might need to update additional versions not specified in the mutation
 */
export const afterMutationsVersion = async ({ created, deletedIds, objectType, prisma, updated, userData }: {
    created: { id: string }[],
    deletedIds: string[],
    objectType: GqlModelType | `${GqlModelType}`,
    prisma: PrismaType,
    updated: { id: string }[]
    userData: SessionUserToken,
}) => {
    // Get prisma delegate for root object
    const { delegate } = getLogic(
        ["delegate"],
        objectType.replace("Version", "") as GqlModelType,
        userData.languages,
        "afterMutationsVersion",
    );
    // Get ids from created, updated, and deletedIds
    const versionIds = [
        ...created.map(({ id }) => id),
        ...updated.map(({ id }) => id),
        ...deletedIds,
    ];
    // Use version ids to query root objects
    const rootData = await delegate(prisma).findMany({
        where: { versions: { some: { id: { in: versionIds } } } },
        select: {
            id: true,
            versions: {
                select: {
                    id: true,
                    isLatest: true,
                    versionIndex: true,
                    versionLabel: true,
                },
            },
        },
    });
    // Because of the versionsCheck function, we can assume that all versions 
    // of a root object have unique versionLabels. Using this, we can sort
    // them to determine the latest version and versionIndex
    const updatedRoots: any[] = [];
    for (const root of rootData) {
        // Sort versions by versionLabel
        const versionsUpdated = root.versions.sort((a, b) => {
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
        // Update versionIndex and isLatest flags
        let versionIndex = 0;
        for (const version of versionsUpdated) {
            version.versionIndex = versionIndex;
            version.isLatest = versionIndex === versionsUpdated.length - 1;
            versionIndex++;
        }
        // If any versions were updated, update root object
        if (versionsUpdated.some(version => version.isLatest !== version.isLatest || version.versionIndex !== version.versionIndex)) {
            updatedRoots.push({
                id: root.id,
                versions: versionsUpdated,
            });
        }
    }
    // Update root objects
    if (updatedRoots.length) {
        await (delegate as any)(prisma).updateMany({
            where: { id: { in: updatedRoots.map(({ id }) => id) } },
            data: updatedRoots,
        });
    }
};
