import { GqlModelType, SessionUser } from "@local/shared";
import { PrismaDelegate } from "../../builders/types";
import { prismaInstance } from "../../db/instance";
import { Trigger } from "../../events/trigger";
import { ModelMap } from "../../models/base";

/**
 * Used in mutate.trigger.afterMutations of non-root and non-version objects. 
 * Calculate data for and calls objectCreated/Updated/Deleted triggers
 */
export async function afterMutationsPlain({
    createdIds,
    deletedIds,
    objectType,
    ownerTeamField,
    ownerUserField,
    updatedIds,
    userData,
}: {
    createdIds: string[],
    deletedIds: string[],
    objectType: GqlModelType | `${GqlModelType}`,
    ownerTeamField?: string,
    ownerUserField?: string,
    updatedIds: string[]
    userData: SessionUser,
}) {
    // Find owners of created and updated items
    const ownerMap: { [key: string]: { id: string, __typename: "User" | "Team" } } = {};
    const createAndUpdateIds = [...createdIds, ...updatedIds];
    const { dbTable } = ModelMap.getLogic(["dbTable"], objectType);
    // Create select object depending on whether ownerTeamField and ownerUserField are defined
    const select = { id: true };
    if (ownerTeamField) {
        select[ownerTeamField] = { select: { id: true } };
    }
    if (ownerUserField) {
        select[ownerUserField] = { select: { id: true } };
    }
    const ownersData = await (prismaInstance[dbTable] as PrismaDelegate).findMany({
        where: { id: { in: createAndUpdateIds } },
        select,
    });
    // Loop through created and updated ids. Don't assume 
    // that ownersData is in the same order as createAndUpdateIds
    for (let i = 0; i < createAndUpdateIds.length; i++) {
        const id = createAndUpdateIds[i];
        const owner = ownersData.find(o => o.id === id);
        if (owner) {
            if (ownerTeamField && owner[ownerTeamField]) {
                ownerMap[id] = { id: owner[ownerTeamField].id, __typename: "Team" };
            }
            if (ownerUserField && owner[ownerUserField]) {
                ownerMap[id] = { id: owner[ownerUserField].id, __typename: "User" };
            }
        }
    }
    // Loop through created items
    for (const objectId of createdIds) {
        Trigger(userData.languages).objectCreated({
            createdById: userData.id,
            hasCompleteAndPublic: true, // N/A
            hasParent: false, // N/A
            owner: ownerMap[objectId],
            objectId,
            objectType,
        });
    }
    // Loop through updated items
    for (const objectId of updatedIds) {
        Trigger(userData.languages).objectUpdated({
            updatedById: userData.id,
            hasCompleteAndPublic: true, // Not applicable
            hasParent: false, // Not applicable
            owner: ownerMap[objectId],
            objectId,
            objectType,
            wasCompleteAndPublic: true, // Not applicable
        });
    }
    // Loop through deleted items
    for (const objectId of deletedIds) {
        Trigger(userData.languages).objectDeleted({
            deletedById: userData.id,
            wasCompleteAndPublic: true, // Not applicable
            hasBeenTransferred: true, // Not applicable
            hasParent: false, // Not applicable
            objectId,
            objectType,
        });
    }
};
