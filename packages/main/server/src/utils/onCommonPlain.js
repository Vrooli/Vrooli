import { Trigger } from "../events";
import { getLogic } from "../getters";
export const onCommonPlain = async ({ created, deletedIds, objectType, ownerOrganizationField, ownerUserField, prisma, updated, userData, }) => {
    const ownerMap = {};
    const createAndUpdateIds = [...created.map(c => c.id), ...updated.map(u => u.id)];
    const { delegate } = getLogic(["delegate"], objectType, userData.languages, "onCommonPlain");
    const select = { id: true };
    if (ownerOrganizationField) {
        select[ownerOrganizationField] = { select: { id: true } };
    }
    if (ownerUserField) {
        select[ownerUserField] = { select: { id: true } };
    }
    const ownersData = await delegate(prisma).findMany({
        where: { id: { in: createAndUpdateIds } },
        select,
    });
    for (let i = 0; i < createAndUpdateIds.length; i++) {
        const id = createAndUpdateIds[i];
        const owner = ownersData.find(o => o.id === id);
        if (owner) {
            if (ownerOrganizationField && owner[ownerOrganizationField]) {
                ownerMap[id] = { id: owner[ownerOrganizationField].id, __typename: "Organization" };
            }
            if (ownerUserField && owner[ownerUserField]) {
                ownerMap[id] = { id: owner[ownerUserField].id, __typename: "User" };
            }
        }
    }
    for (const c of created) {
        Trigger(prisma, userData.languages).objectCreated({
            createdById: userData.id,
            hasCompleteAndPublic: true,
            hasParent: false,
            owner: ownerMap[c.id],
            objectId: c.id,
            objectType,
        });
    }
    for (const u of updated) {
        Trigger(prisma, userData.languages).objectUpdated({
            updatedById: userData.id,
            hasCompleteAndPublic: true,
            hasParent: false,
            owner: ownerMap[u.id],
            objectId: u.id,
            objectType,
            wasCompleteAndPublic: true,
        });
    }
    for (const d of deletedIds) {
        Trigger(prisma, userData.languages).objectDeleted({
            deletedById: userData.id,
            wasCompleteAndPublic: true,
            hasBeenTransferred: true,
            hasParent: false,
            objectId: d,
            objectType,
        });
    }
};
//# sourceMappingURL=onCommonPlain.js.map