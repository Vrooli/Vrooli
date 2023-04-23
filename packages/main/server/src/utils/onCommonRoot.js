import { Trigger } from "../events";
import { TransferModel } from "../models";
export const onCommonRoot = async ({ created, deletedIds, objectType, preMap, prisma, updated, userData }) => {
    for (let i = 0; i < created.length; i++) {
        const objectId = created[i].id;
        const { hasCompleteAndPublic, hasParent, owner, } = preMap[objectType].triggerMap[objectId];
        await Trigger(prisma, userData.languages).objectCreated({
            createdById: userData.id,
            hasCompleteAndPublic,
            hasParent,
            owner,
            objectId,
            objectType,
            projectId: undefined,
        });
        const requiresTransfer = preMap[objectType].transferMap[objectId];
        if (requiresTransfer) {
            await TransferModel.transfer(prisma).requestSend({
                objectConnect: objectId,
                objectType,
                toOrganizationConnect: owner.__typename === "Organization" ? owner.id : undefined,
                toUserConnect: owner.__typename === "User" ? owner.id : undefined,
            }, userData);
        }
    }
    for (let i = 0; i < updated.length; i++) {
        const objectId = updated[i].id;
        const { hasCompleteAndPublic, hasParent, owner, wasCompleteAndPublic, } = preMap[objectType].triggerMap[objectId];
        await Trigger(prisma, userData.languages).objectUpdated({
            updatedById: userData.id,
            hasCompleteAndPublic,
            hasParent,
            owner,
            objectId,
            objectType,
            originalProjectId: undefined,
            projectId: undefined,
            wasCompleteAndPublic,
        });
        const requiresTransfer = preMap[objectType].transferMap[objectId];
        if (requiresTransfer) {
            await TransferModel.transfer(prisma).requestSend({
                objectConnect: objectId,
                objectType,
                toOrganizationConnect: owner.__typename === "Organization" ? owner.id : undefined,
                toUserConnect: owner.__typename === "User" ? owner.id : undefined,
            }, userData);
        }
    }
    for (let i = 0; i < deletedIds.length; i++) {
        const objectId = deletedIds[i];
        const { hasBeenTransferred, hasParent, wasCompleteAndPublic, } = preMap[objectType].triggerMap[objectId];
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
//# sourceMappingURL=onCommonRoot.js.map