import { GqlModelType } from "@local/shared";
import { Trigger } from "../events";
import { TransferModel } from "../models";
import { PrismaType, SessionUserToken } from "../types";

/**
 * Used in mutate.trigger.onCommon of version objects. Has two purposes:
 * 1. Update versionIndex and isLatest flags. Cannot be done in pre 
 * because we might need to update additional versions not specified in the mutation
 * 2. Calculate data for and calls objectCreated/Updated/Deleted triggers
 */
export const onCommonRoot = async ({ created, deletedIds, objectType, preMap, prisma, updated, userData }: {
    created: { id: string }[],
    deletedIds: string[],
    objectType: GqlModelType | `${GqlModelType}`,
    preMap: { [key in GqlModelType]?: any },
    prisma: PrismaType,
    updated: { id: string }[]
    userData: SessionUserToken,
}) => {
    // Loop through created items
    for (let i = 0; i < created.length; i++) {
        const objectId = created[i].id;
        // Get trigger info
        const {
            hasCompleteAndPublic,
            hasParent,
            owner,
        } = preMap[objectType].triggerMap[objectId];
        // Trigger objectCreated
        await Trigger(prisma, userData.languages).objectCreated({
            createdById: userData.id,
            hasCompleteAndPublic,
            hasParent,
            owner,
            objectId,
            objectType,
            // Projects are attached to versions, not root objects
            projectId: undefined,
        });
        // Get transfer info 
        const requiresTransfer = preMap[objectType].transferMap[objectId];
        if (requiresTransfer) {
            // Create transfer
            await TransferModel.transfer(prisma).requestSend({
                objectConnect: objectId,
                objectType,
                toOrganizationConnect: owner.__typename === "Organization" ? owner.id : undefined,
                toUserConnect: owner.__typename === "User" ? owner.id : undefined,
            }, userData);
        }
    }
    // Loop through updated items
    for (let i = 0; i < updated.length; i++) {
        const objectId = updated[i].id;
        // Get trigger info
        const {
            hasCompleteAndPublic,
            hasParent,
            owner,
            wasCompleteAndPublic,
        } = preMap[objectType].triggerMap[objectId];
        // Trigger objectUpdated
        await Trigger(prisma, userData.languages).objectUpdated({
            updatedById: userData.id,
            hasCompleteAndPublic,
            hasParent,
            owner,
            objectId,
            objectType,
            // Projects are attached to versions, not root objects
            originalProjectId: undefined,
            projectId: undefined,
            wasCompleteAndPublic,
        });
        // Get transfer info 
        const requiresTransfer = preMap[objectType].transferMap[objectId];
        if (requiresTransfer) {
            // Create transfer
            await TransferModel.transfer(prisma).requestSend({
                objectConnect: objectId,
                objectType,
                toOrganizationConnect: owner.__typename === "Organization" ? owner.id : undefined,
                toUserConnect: owner.__typename === "User" ? owner.id : undefined,
            }, userData);
        }
    }
    // Loop through deleted items
    for (let i = 0; i < deletedIds.length; i++) {
        const objectId = deletedIds[i];
        // Get trigger info
        const {
            hasBeenTransferred,
            hasParent,
            wasCompleteAndPublic,
        } = preMap[objectType].triggerMap[objectId];
        // Trigger objectDeleted
        await Trigger(prisma, userData.languages).objectDeleted({
            deletedById: userData.id,
            hasBeenTransferred,
            hasParent,
            objectId,
            objectType,
            wasCompleteAndPublic,
        });
    }
};
