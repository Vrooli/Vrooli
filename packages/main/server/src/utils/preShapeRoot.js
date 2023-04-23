import { exists } from "@local/utils";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { transfer } from "../models";
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
        },
    },
};
export const preShapeRoot = async ({ createList, updateList, deleteList, objectType, prisma, userData, }) => {
    const { delegate } = getLogic(["delegate"], objectType, userData.languages, "preHasPublics");
    const versionMap = {};
    const triggerMap = {};
    const transferMap = {};
    for (const create of createList) {
        const hasCompleteVersion = create.versionsCreate?.some(v => v.isComplete) ?? false;
        versionMap[create.id] = {
            hasCompleteVersion,
            completedAt: hasCompleteVersion ? new Date() : null,
        };
        triggerMap[create.id] = {
            wasCompleteAndPublic: true,
            hasCompleteAndPublic: !create.isPrivate && (create.versionsCreate?.some(v => v.isComplete && !v.isPrivate) ?? false),
            hasBeenTransferred: false,
            hasParent: typeof create.parentConnect === "string",
            owner: {
                id: create.ownedByUser ?? create.ownedByOrganization,
                __typename: create.ownedByUser ? "User" : "Organization",
            },
        };
    }
    if (updateList.length > 0) {
        const originalData = await delegate(prisma).findMany({
            where: { id: { in: updateList.map(u => u.data.id) } },
            select: originalDataSelect,
        });
        for (const update of updateList) {
            const original = originalData.find(r => r.id === update.data.id);
            if (!original)
                throw new CustomError("0412", "InternalError", userData.languages, { id: update?.data?.id });
            const isRootPrivate = update.data.isPrivate ?? original.isPrivate;
            const updatedWithOriginal = original.versions.reduce((acc, v) => ({ ...acc, [v.id]: v }), {});
            if (update.data.versionsUpdate) {
                for (const v of update.data.versionsUpdate) {
                    updatedWithOriginal[v.id] = {
                        ...updatedWithOriginal[v.id],
                        ...v,
                    };
                }
            }
            const allVersions = Object.values(updatedWithOriginal).concat(update.data.versionsCreate ?? []);
            const versions = allVersions.filter(v => !update.data.versionsDelete?.includes(v.id));
            const hasCompleteVersion = versions.some(v => v.isComplete);
            versionMap[update.data.id] = {
                hasCompleteVersion,
                completedAt: hasCompleteVersion ? new Date() : null,
            };
            triggerMap[update.data.id] = {
                wasCompleteAndPublic: !original.isPrivate && original.versions.some((v) => v.isComplete && !v.isPrivate),
                hasCompleteAndPublic: !isRootPrivate && versions.some((v) => v.isComplete && !v.isPrivate),
                hasBeenTransferred: original.hasBeenTransferred,
                hasParent: exists(original.parent),
                owner: {
                    id: original.ownedByUser?.id ?? original.ownedByOrganization?.id,
                    __typename: original.ownedByUser ? "User" : "Organization",
                },
            };
        }
    }
    if (deleteList.length > 0) {
        const originalData = await delegate(prisma).findMany({
            where: { id: { in: deleteList } },
            select: originalDataSelect,
        });
        for (const id of deleteList) {
            const original = originalData.find(r => r.id === id);
            if (!original)
                throw new CustomError("0413", "InternalError", userData.languages, { id });
            triggerMap[id] = {
                wasCompleteAndPublic: !original.isPrivate && original.versions.some((v) => v.isComplete && !v.isPrivate),
                hasCompleteAndPublic: true,
                hasBeenTransferred: original.hasBeenTransferred,
                hasParent: exists(original.parent),
                owner: {
                    id: original.ownedByUser?.id ?? original.ownedByOrganization?.id,
                    __typename: original.ownedByUser ? "User" : "Organization",
                },
            };
        }
    }
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
};
//# sourceMappingURL=preShapeRoot.js.map