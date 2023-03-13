import { GqlModelType, SessionUser } from "@shared/consts";
import { Trigger } from "../events";
import { TransferModel } from "../models";
import { PrismaType } from "../types";

/**
 * Handles trigger and transfer creation for root objects
 */
export const onRootUpdated = async ({ objectType, preMap, prisma, updated, userData }: {
    objectType: GqlModelType | `${GqlModelType}`,
    preMap: { [x in `${GqlModelType}`]?: any },
    prisma: PrismaType,
    updated: { [x: string]: any }[],
    userData: SessionUser,
}) => {
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
                toOrganizationConnect: owner.__typename === 'Organization' ? owner.id : undefined,
                toUserConnect: owner.__typename === 'User' ? owner.id : undefined,
            }, userData);
        }
    }
}