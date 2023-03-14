import { GqlModelType, SessionUser } from "@shared/consts";
import { Trigger } from "../events";
import { getLogic } from "../getters";
import { PrismaType } from "../types";

/**
 * Used in mutate.trigger.onCommon of non-root and non-version objects. 
 * Calculate data for and calls objectCreated/Updated/Deleted triggers
 */
export const onCommonPlain = async ({ 
    created, 
    deletedIds, 
    objectType, 
    ownerOrganizationField,
    ownerUserField,
    prisma, 
    updated, 
    userData 
}: {
    created: { id: string }[],
    deletedIds: string[],
    objectType: GqlModelType | `${GqlModelType}`,
    ownerOrganizationField?: string,
    ownerUserField?: string,
    prisma: PrismaType,
    updated: { id: string }[]
    userData: SessionUser,
}) => {
    // Find owners of created and updated items
    const ownerMap: { [key: string]: { id: string, __typename: 'User' | 'Organization' } } = {};
    const createAndUpdateIds = [...created.map(c => c.id), ...updated.map(u => u.id)];
    const { delegate } = getLogic(['delegate'], objectType, userData.languages, 'onCommonPlain');
    // Create select object depending on whether ownerOrganizationField and ownerUserField are defined
    let select = { id: true };
    if (ownerOrganizationField) {
        select[ownerOrganizationField] = { select: { id: true } };
    }
    if (ownerUserField) {
        select[ownerUserField] = { select: { id: true } };
    }
    const ownersData = await delegate(prisma).findMany({
        where: { id: { in: createAndUpdateIds } },
        select,
    })
    // Loop through created and updated ids. Don't assume 
    // that ownersData is in the same order as createAndUpdateIds
    for (let i = 0; i < createAndUpdateIds.length; i++) {
        const id = createAndUpdateIds[i];
        const owner = ownersData.find(o => o.id === id);
        if (owner) {
            if (ownerOrganizationField && owner[ownerOrganizationField]) {
                ownerMap[id] = { id: owner[ownerOrganizationField].id, __typename: 'Organization' };
            }
            if (ownerUserField && owner[ownerUserField]) {
                ownerMap[id] = { id: owner[ownerUserField].id, __typename: 'User' };
            }
        }
    }
    // Loop through created items
    for (const c of created) {
        Trigger(prisma, userData.languages).objectCreated({
            createdById: userData.id,
            hasCompleteAndPublic: true, // N/A
            hasParent: false, // N/A
            owner: ownerMap[c.id],
            objectId: c.id as string,
            objectType,
        });
    }
    // Loop through updated items
    for (const u of updated) {
        Trigger(prisma, userData.languages).objectUpdated({
            updatedById: userData.id,
            hasCompleteAndPublic: true, // Not applicable
            hasParent: false, // Not applicable
            owner: ownerMap[u.id],
            objectId: u.id as string,
            objectType,
            wasCompleteAndPublic: true, // Not applicable
        });
    }
    // Loop through deleted items
    for (const d of deletedIds) {
        Trigger(prisma, userData.languages).objectDeleted({
            deletedById: userData.id,
            wasCompleteAndPublic: true, // Not applicable
            hasBeenTransferred: true, // Not applicable
            hasParent: false, // Not applicable
            objectId: d,
            objectType,
        });
    }
}