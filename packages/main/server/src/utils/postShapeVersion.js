import { calculateVersionsFromString } from "@local/validation";
import { getLogic } from "../getters";
export const postShapeVersion = async ({ created, deletedIds, objectType, prisma, updated, userData }) => {
    const { delegate } = getLogic(["delegate"], objectType.replace("Version", ""), userData.languages, "onCommonVersion");
    const versionIds = [
        ...created.map(({ id }) => id),
        ...updated.map(({ id }) => id),
        ...deletedIds,
    ];
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
    const updatedRoots = [];
    for (const root of rootData) {
        const versionsUpdated = root.versions.sort((a, b) => {
            const { major: majorA, moderate: moderateA, minor: minorA } = calculateVersionsFromString(a.versionLabel);
            const { major: majorB, moderate: moderateB, minor: minorB } = calculateVersionsFromString(b.versionLabel);
            if (majorA > majorB)
                return 1;
            if (majorA < majorB)
                return -1;
            if (moderateA > moderateB)
                return 1;
            if (moderateA < moderateB)
                return -1;
            if (minorA > minorB)
                return 1;
            if (minorA < minorB)
                return -1;
            return 0;
        });
        let versionIndex = 0;
        for (const version of versionsUpdated) {
            version.versionIndex = versionIndex;
            version.isLatest = versionIndex === versionsUpdated.length - 1;
            versionIndex++;
        }
        if (versionsUpdated.some(version => version.isLatest !== version.isLatest || version.versionIndex !== version.versionIndex)) {
            updatedRoots.push({
                id: root.id,
                versions: versionsUpdated,
            });
        }
    }
    if (updatedRoots.length) {
        await delegate(prisma).updateMany({
            where: { id: { in: updatedRoots.map(({ id }) => id) } },
            data: updatedRoots,
        });
    }
};
//# sourceMappingURL=postShapeVersion.js.map