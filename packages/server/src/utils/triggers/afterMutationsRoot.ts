import { type ModelType, type SessionUser } from "@vrooli/shared";
import { CustomError } from "../../events/error.js";
import { Trigger } from "../../events/trigger.js";
import { TransferModel } from "../../models/base/transfer.js";
import { type PreMap } from "../../models/types.js";

/**
 * Used in mutate.trigger.onCommon of version objects. Has two purposes:
 * 1. Update versionIndex and isLatest flags. Cannot be done in pre 
 * because we might need to update additional versions not specified in the mutation
 * 2. Calculate data for and calls objectCreated/Updated/Deleted triggers
 */
export async function afterMutationsRoot({ createdIds, deletedIds, objectType, preMap, updatedIds, userData }: {
    createdIds: string[],
    deletedIds: string[],
    objectType: ModelType | `${ModelType}`,
    preMap: PreMap,
    updatedIds: string[]
    userData: SessionUser,
}) {
    // Loop through created items
    for (let i = 0; i < createdIds.length; i++) {
        const objectId = createdIds[i];
        // Get trigger info
        const {
            hasCompleteAndPublic,
            hasParent,
            owner,
        } = preMap[objectType].triggerMap[objectId];
        // Trigger objectCreated
        await Trigger(userData.languages).objectCreated({
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
            await TransferModel.transfer().requestSend({
                objectConnect: objectId,
                objectType,
                toTeamConnect: owner.__typename === "Team" ? owner.id : undefined,
                toUserConnect: owner.__typename === "User" ? owner.id : undefined,
            }, userData);
        }
    }
    // Loop through updated items
    for (let i = 0; i < updatedIds.length; i++) {
        const objectId = updatedIds[i];
        // Get trigger info
        const {
            hasCompleteAndPublic,
            hasParent,
            owner,
            wasCompleteAndPublic,
        } = preMap[objectType].triggerMap[objectId];
        // Trigger objectUpdated
        await Trigger(userData.languages).objectUpdated({
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
            await TransferModel.transfer().requestSend({
                objectConnect: objectId,
                objectType,
                toTeamConnect: owner.__typename === "Team" ? owner.id : undefined,
                toUserConnect: owner.__typename === "User" ? owner.id : undefined,
            }, userData);
        }
    }
    // Loop through deleted items
    for (let i = 0; i < deletedIds.length; i++) {
        const objectId = deletedIds[i];
        // Get trigger info
        const preData = preMap[objectType];
        if (!preData || !preData.triggerMap || !preData.triggerMap[objectId]) {
            throw new CustomError("0085", "InternalError");
        }
        const { transferredAt, hasParent, wasCompleteAndPublic } = preData.triggerMap[objectId];
        // Trigger objectDeleted
        await Trigger(userData.languages).objectDeleted({
            deletedById: userData.id,
            transferredAt,
            hasParent,
            objectId,
            objectType,
            wasCompleteAndPublic,
        });
    }
}
