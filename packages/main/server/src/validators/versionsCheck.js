import { GqlModelType } from "@local/consts";
import { CustomError } from "../events";
import { getLogic } from "../getters";
const hasInternalField = (objectType) => [GqlModelType.RoutineVersion, GqlModelType.StandardVersion].includes(objectType);
export const versionsCheck = async ({ prisma, objectType, createList, updateList, deleteList, userData, }) => {
    const create = createList.filter(x => x.versionLabel).map(x => ({
        id: x.id,
        rootId: x.rootConnect || x.rootCreate?.id,
        versionLabel: x.versionLabel,
    }));
    const update = updateList.filter(x => x.data.versionLabel).map(x => ({
        id: x.where.id,
        versionLabel: x.data.versionLabel,
    }));
    const createRootIds = create.map(x => x.rootId);
    const uniqueRootIds = [...new Set(createRootIds)];
    const updateIds = update.map(x => x.id);
    const deleteIds = deleteList;
    const uniqueVersionIds = [...new Set([...updateIds, ...deleteIds])];
    const rootType = objectType.replace("Version", "");
    const { delegate } = getLogic(["delegate"], rootType, userData.languages, "versionsCheck");
    let existingRoots;
    let where = {};
    let select = {};
    try {
        where = {
            OR: [
                { id: { in: uniqueRootIds } },
                { versions: { some: { id: { in: uniqueVersionIds } } } },
            ],
        };
        select = {
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            ...(hasInternalField(objectType) && { isInternal: true }),
            isPrivate: true,
            versions: {
                select: {
                    id: true,
                    isComplete: true,
                    isDeleted: true,
                    isPrivate: true,
                    versionLabel: true,
                },
            },
        };
        existingRoots = await delegate(prisma).findMany({
            where,
            select,
        });
    }
    catch (error) {
        throw new CustomError("0414", "InternalError", userData.languages, { error, where, select, rootType });
    }
    for (const root of existingRoots) {
        if (root.isDeleted) {
            throw new CustomError("0377", "ErrorUnknown", userData.languages);
        }
        for (const version of root.versions) {
            if (updateIds.includes(version.id) && version.isDeleted) {
                throw new CustomError("0378", "ErrorUnknown", userData.languages);
            }
        }
        const versionIds = root.versions.filter(x => !deleteIds.includes(x.id)).map(x => x.id);
        const versionLabels = root.versions.filter(x => !deleteIds.includes(x.id)).map(x => x.versionLabel);
        const createLabels = create.filter(x => x.rootId === root.id).map(x => x.versionLabel);
        if (createLabels.some(x => versionLabels.includes(x))) {
            throw new CustomError("0379", "ErrorUnknown", userData.languages);
        }
        const updateLabels = update.filter(x => versionIds.includes(x.id)).map(x => x.versionLabel);
        const verionLabelsWithoutUpdate = root.versions.filter(x => !deleteIds.includes(x.id) && !updateIds.includes(x.id)).map(x => x.versionLabel);
        if (updateLabels.some(x => verionLabelsWithoutUpdate.includes(x))) {
            throw new CustomError("0380", "ErrorUnknown", userData.languages);
        }
        if (!root.isPrivate && !(hasInternalField(objectType) && root.isInternal)) {
            for (const version of root.versions) {
                if (updateIds.includes(version.id) && !version.isPrivate && version.isComplete) {
                    throw new CustomError("0381", "ErrorUnknown", userData.languages);
                }
            }
        }
    }
};
//# sourceMappingURL=versionsCheck.js.map